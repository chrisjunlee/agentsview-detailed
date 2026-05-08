package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

const (
	storyViewportWidth  int64 = 1080
	storyViewportHeight int64 = 1920
	storyRenderTimeout        = 15 * time.Second
	storyNetworkIdle          = 500 * time.Millisecond
	storyNetworkWait          = 10 * time.Second
	storySettleDelay          = 300 * time.Millisecond
	storyAuthTokenKey         = "agentsview-auth-token"
)

type storyRenderRequest struct {
	URL       string
	SeedURL   string
	AuthToken string
}

type storyRenderer interface {
	Render(context.Context, storyRenderRequest) ([]byte, error)
}

func withStoryRenderer(renderer storyRenderer) Option {
	return func(s *Server) {
		if renderer != nil {
			s.storyRenderer = renderer
		}
	}
}

func (s *Server) handleStory(w http.ResponseWriter, r *http.Request) {
	appPath, err := parseStoryAppPath(r.URL.Query().Get("path"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	targetURL := s.storyTargetURL(appPath)
	req := storyRenderRequest{
		URL:       targetURL,
		SeedURL:   s.storySeedURL(),
		AuthToken: bearerToken(r),
	}

	ctx, cancel := context.WithTimeout(r.Context(), storyRenderTimeout)
	defer cancel()

	if err := s.acquireStoryRenderSlot(ctx); err != nil {
		handleContextError(w, err)
		return
	}
	defer s.releaseStoryRenderSlot()

	png, err := s.storyRenderer.Render(ctx, req)
	if err != nil {
		if handleContextError(w, err) || handleContextError(w, ctx.Err()) {
			return
		}
		writeError(w, http.StatusInternalServerError,
			fmt.Sprintf("rendering story screenshot: %v", err))
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(png)
}

func (s *Server) acquireStoryRenderSlot(ctx context.Context) error {
	if s.storyRenderSem == nil {
		s.storyRenderSem = make(chan struct{}, 1)
	}
	select {
	case s.storyRenderSem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Server) releaseStoryRenderSlot() {
	if s.storyRenderSem == nil {
		return
	}
	select {
	case <-s.storyRenderSem:
	default:
	}
}

func parseStoryAppPath(raw string) (*url.URL, error) {
	if raw == "" {
		raw = "/usage"
	}
	if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") {
		return nil, errors.New("path must be a same-origin app path")
	}
	for _, r := range raw {
		if unicode.IsControl(r) {
			return nil, errors.New("path contains control characters")
		}
	}

	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}
	if u.Scheme != "" || u.Host != "" || u.User != nil || u.Opaque != "" {
		return nil, errors.New("path must be relative to this app")
	}
	if u.Fragment != "" {
		return nil, errors.New("path fragments are not supported")
	}
	if u.Path == "" || !strings.HasPrefix(u.Path, "/") ||
		strings.HasPrefix(u.Path, "//") {
		return nil, errors.New("path must start with /")
	}

	clean := path.Clean(u.Path)
	if clean == "/api" || strings.HasPrefix(clean, "/api/") ||
		clean == "/story" || strings.HasPrefix(clean, "/story/") {
		return nil, errors.New("path must target an app page")
	}
	return u, nil
}

func (s *Server) storyTargetURL(appPath *url.URL) string {
	u := s.storyBaseURL()
	u.Path = s.basePath + appPath.Path
	u.RawQuery = appPath.RawQuery
	return u.String()
}

func (s *Server) storySeedURL() string {
	u := s.storyBaseURL()
	if s.basePath != "" {
		u.Path = s.basePath + "/"
	} else {
		u.Path = "/"
	}
	return u.String()
}

func (s *Server) storyBaseURL() url.URL {
	host := storyLoopbackHost(s.cfg.Host)
	return url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(host, strconv.Itoa(s.cfg.Port)),
	}
}

func storyLoopbackHost(host string) string {
	switch host {
	case "", "0.0.0.0", "::":
		return "127.0.0.1"
	default:
		return host
	}
}

func bearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	token, ok := strings.CutPrefix(auth, "Bearer ")
	if !ok {
		return ""
	}
	return token
}

type chromedpStoryRenderer struct{}

func (chromedpStoryRenderer) Render(
	ctx context.Context, req storyRenderRequest,
) ([]byte, error) {
	allocatorCtx, cancel := chromedp.NewExecAllocator(
		ctx,
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.WindowSize(
				int(storyViewportWidth),
				int(storyViewportHeight),
			),
		)...,
	)
	defer cancel()

	browserCtx, cancel := chromedp.NewContext(allocatorCtx)
	defer cancel()

	idle := newStoryNetworkIdleWaiter()
	chromedp.ListenTarget(browserCtx, idle.listen)

	actions := []chromedp.Action{
		network.Enable(),
		emulation.SetDeviceMetricsOverride(
			storyViewportWidth,
			storyViewportHeight,
			1,
			false,
		),
	}
	if req.AuthToken != "" {
		actions = append(actions,
			chromedp.Navigate(req.SeedURL),
			chromedp.WaitReady("body", chromedp.ByQuery),
			chromedp.Evaluate(
				setLocalStorageExpression(storyAuthTokenKey, req.AuthToken),
				nil,
			),
		)
	}

	var png []byte
	actions = append(actions,
		chromedp.ActionFunc(func(context.Context) error {
			idle.reset()
			return nil
		}),
		chromedp.Navigate(req.URL),
		chromedp.WaitReady("body", chromedp.ByQuery),
		idle.wait(storyNetworkIdle, storyNetworkWait),
		chromedp.Evaluate(waitFontsExpression, nil),
		chromedp.Sleep(storySettleDelay),
		chromedp.CaptureScreenshot(&png),
	)

	if err := chromedp.Run(browserCtx, actions...); err != nil {
		return nil, err
	}
	return png, nil
}

const waitFontsExpression = `
new Promise((resolve) => {
  const done = () => requestAnimationFrame(() => requestAnimationFrame(resolve));
  if (document.fonts && document.fonts.ready) {
    document.fonts.ready.then(done, done);
  } else {
    done();
  }
})
`

func setLocalStorageExpression(key, value string) string {
	return fmt.Sprintf(
		"localStorage.setItem(%s, %s)",
		strconv.Quote(key),
		strconv.Quote(value),
	)
}

type storyNetworkIdleWaiter struct {
	mu       sync.Mutex
	inFlight map[network.RequestID]struct{}
	lastSeen time.Time
}

func newStoryNetworkIdleWaiter() *storyNetworkIdleWaiter {
	return &storyNetworkIdleWaiter{
		inFlight: make(map[network.RequestID]struct{}),
		lastSeen: time.Now(),
	}
}

func (w *storyNetworkIdleWaiter) listen(ev any) {
	switch ev := ev.(type) {
	case *network.EventRequestWillBeSent:
		if shouldIgnoreStoryResource(ev.Type) {
			return
		}
		w.mu.Lock()
		w.inFlight[ev.RequestID] = struct{}{}
		w.lastSeen = time.Now()
		w.mu.Unlock()
	case *network.EventLoadingFinished:
		w.finish(ev.RequestID)
	case *network.EventLoadingFailed:
		w.finish(ev.RequestID)
	}
}

func (w *storyNetworkIdleWaiter) reset() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.inFlight = make(map[network.RequestID]struct{})
	w.lastSeen = time.Now()
}

func (w *storyNetworkIdleWaiter) finish(id network.RequestID) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.inFlight, id)
	w.lastSeen = time.Now()
}

func (w *storyNetworkIdleWaiter) wait(
	idleFor time.Duration,
	maxWait time.Duration,
) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		timer := time.NewTimer(maxWait)
		defer timer.Stop()
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()

		for {
			if w.isIdle(idleFor) {
				return nil
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return context.DeadlineExceeded
			case <-ticker.C:
			}
		}
	})
}

func (w *storyNetworkIdleWaiter) isIdle(idleFor time.Duration) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.inFlight) == 0 && time.Since(w.lastSeen) >= idleFor
}

func shouldIgnoreStoryResource(t network.ResourceType) bool {
	return t == network.ResourceTypeEventSource ||
		t == network.ResourceTypeWebSocket ||
		t == network.ResourceTypePing
}
