package server

import (
	"context"
	"errors"
	"testing"
	"time"
)

// The story endpoint serializes Chrome renders through a 1-slot semaphore
// (acquireStoryRenderSlot / releaseStoryRenderSlot) so concurrent /story
// requests don't spawn overlapping headless browsers. These tests pin that
// behavior directly; the handler tests in story_test.go only exercise it
// implicitly.

// TestAcquireStoryRenderSlotLazyInit verifies acquire lazily allocates the
// semaphore when the Server was constructed without one and that the first
// acquisition on a free slot returns nil.
func TestAcquireStoryRenderSlotLazyInit(t *testing.T) {
	s := &Server{} // storyRenderSem is nil

	if err := s.acquireStoryRenderSlot(context.Background()); err != nil {
		t.Fatalf("first acquire should succeed: %v", err)
	}
	if s.storyRenderSem == nil {
		t.Fatal("acquire should lazily initialize storyRenderSem")
	}
}

// TestAcquireStoryRenderSlotBlocksWhenFull verifies that once the single slot
// is taken, a second acquire respects context cancellation instead of granting
// a second concurrent render.
func TestAcquireStoryRenderSlotBlocksWhenFull(t *testing.T) {
	s := &Server{storyRenderSem: make(chan struct{}, 1)}

	if err := s.acquireStoryRenderSlot(context.Background()); err != nil {
		t.Fatalf("first acquire should succeed: %v", err)
	}

	// Slot is full. With a canceled context the send case is not ready, so
	// only the ctx.Done case can fire — deterministic, no random select.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := s.acquireStoryRenderSlot(ctx)
	if err == nil {
		t.Fatal("second acquire should fail while the slot is held")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

// TestReleaseStoryRenderSlotUnblocks verifies that releasing the slot makes it
// available to the next acquirer. A regression that failed to drain the
// channel would make the second acquire block until the deadline, which the
// timeout converts into a clean failure rather than a hung test.
func TestReleaseStoryRenderSlotUnblocks(t *testing.T) {
	s := &Server{storyRenderSem: make(chan struct{}, 1)}

	if err := s.acquireStoryRenderSlot(context.Background()); err != nil {
		t.Fatalf("first acquire should succeed: %v", err)
	}
	s.releaseStoryRenderSlot()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := s.acquireStoryRenderSlot(ctx); err != nil {
		t.Errorf("acquire after release should succeed, got: %v", err)
	}
}

// TestReleaseStoryRenderSlotNoSlotHeld verifies release is a safe no-op when no
// slot is currently held (the default branch of its non-blocking receive) and
// leaves the slot acquirable.
func TestReleaseStoryRenderSlotNoSlotHeld(t *testing.T) {
	s := &Server{storyRenderSem: make(chan struct{}, 1)}

	s.releaseStoryRenderSlot() // nothing held: must not block or panic

	if err := s.acquireStoryRenderSlot(context.Background()); err != nil {
		t.Errorf("acquire after no-op release should succeed: %v", err)
	}
}

// TestReleaseStoryRenderSlotNilSem verifies release on a Server whose semaphore
// was never initialized is a no-op (the nil guard) rather than a nil-channel
// panic.
func TestReleaseStoryRenderSlotNilSem(t *testing.T) {
	s := &Server{} // storyRenderSem is nil
	s.releaseStoryRenderSlot()
}
