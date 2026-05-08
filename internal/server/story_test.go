package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chromedp/cdproto/network"
)

type fakeStoryRenderer struct {
	mu       sync.Mutex
	response []byte
	err      error
	calls    int
	req      storyRenderRequest
}

func (f *fakeStoryRenderer) Render(
	_ context.Context, req storyRenderRequest,
) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.req = req
	if f.err != nil {
		return nil, f.err
	}
	return f.response, nil
}

func (f *fakeStoryRenderer) snapshot() (int, storyRenderRequest) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls, f.req
}

func TestParseStoryAppPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{
			name: "default path",
			raw:  "",
			want: "/usage",
		},
		{
			name: "path with query",
			raw:  "/usage?from=2026-05-08&to=2026-05-08",
			want: "/usage?from=2026-05-08&to=2026-05-08",
		},
		{
			name:    "absolute URL rejected",
			raw:     "https://example.com/usage",
			wantErr: true,
		},
		{
			name:    "protocol relative URL rejected",
			raw:     "//example.com/usage",
			wantErr: true,
		},
		{
			name:    "relative route rejected",
			raw:     "usage",
			wantErr: true,
		},
		{
			name:    "api route rejected",
			raw:     "/api/v1/stats",
			wantErr: true,
		},
		{
			name:    "cleaned api route rejected",
			raw:     "/usage/../api/v1/stats",
			wantErr: true,
		},
		{
			name:    "story route rejected",
			raw:     "/story",
			wantErr: true,
		},
		{
			name:    "fragment rejected",
			raw:     "/usage#cards",
			wantErr: true,
		},
		{
			name:    "control character rejected",
			raw:     "/usage\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseStoryAppPath(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.String() != tt.want {
				t.Fatalf("path = %q, want %q", got.String(), tt.want)
			}
		})
	}
}

func TestHandleStoryRendersPNG(t *testing.T) {
	t.Parallel()

	renderer := &fakeStoryRenderer{
		response: []byte{0x89, 'P', 'N', 'G'},
	}
	s := testServer(t, time.Second,
		WithBasePath("/viewer"),
		withStoryRenderer(renderer),
	)

	nestedPath := "/usage?from=2026-05-08&to=2026-05-08"
	req := httptest.NewRequest(
		http.MethodGet,
		"/viewer/api/v1/story?path="+url.QueryEscape(nestedPath),
		nil,
	)
	req.Host = net.JoinHostPort("127.0.0.1", "0")
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Authorization", "Bearer story-token")
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	assertRecorderStatus(t, w, http.StatusOK)
	assertContentType(t, w, "image/png")
	if got := w.Body.Bytes(); string(got) != string(renderer.response) {
		t.Fatalf("body = %v, want %v", got, renderer.response)
	}

	calls, gotReq := renderer.snapshot()
	if calls != 1 {
		t.Fatalf("renderer calls = %d, want 1", calls)
	}
	wantURL := "http://127.0.0.1:0/viewer/usage?from=2026-05-08&story=1&to=2026-05-08"
	if gotReq.URL != wantURL {
		t.Fatalf("URL = %q, want %q", gotReq.URL, wantURL)
	}
	wantSeedURL := "http://127.0.0.1:0/viewer/"
	if gotReq.SeedURL != wantSeedURL {
		t.Fatalf("SeedURL = %q, want %q", gotReq.SeedURL, wantSeedURL)
	}
	if gotReq.AuthToken != "story-token" {
		t.Fatalf("AuthToken = %q, want story-token", gotReq.AuthToken)
	}
	if got := w.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
}

func TestHandleStoryPageRendersBrowserWrapper(t *testing.T) {
	t.Parallel()

	s := testServer(t, time.Second, WithBasePath("/viewer"))

	nestedPath := "/usage?from=2026-05-08&to=2026-05-08"
	req := httptest.NewRequest(
		http.MethodGet,
		"/viewer/story?path="+url.QueryEscape(nestedPath),
		nil,
	)
	req.Host = net.JoinHostPort("127.0.0.1", "0")
	req.RemoteAddr = "127.0.0.1:1234"
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	assertRecorderStatus(t, w, http.StatusOK)
	assertContentType(t, w, "text/html; charset=utf-8")
	body := w.Body.String()
	for _, want := range []string{
		`width: 1080px;`,
		`height: 1920px;`,
		`width: 540px;`,
		`height: 960px;`,
		`transform: scale(2);`,
		`src="/viewer/usage?from=2026-05-08&amp;story=1&amp;to=2026-05-08"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("story page missing %q in body:\n%s", want, body)
		}
	}
	if got := w.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
}

func TestHandleStoryPageRejectsInvalidPath(t *testing.T) {
	t.Parallel()

	s := testServer(t, time.Second)

	req := httptest.NewRequest(
		http.MethodGet,
		"/story?path="+url.QueryEscape("/api/v1/stats"),
		nil,
	)
	req.Host = net.JoinHostPort("127.0.0.1", "0")
	req.RemoteAddr = "127.0.0.1:1234"
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	assertRecorderStatus(t, w, http.StatusBadRequest)
	if !strings.Contains(w.Body.String(), "app page") {
		t.Fatalf("body = %q, want app page error", w.Body.String())
	}
}

func TestHandleStoryRejectsInvalidPath(t *testing.T) {
	t.Parallel()

	renderer := &fakeStoryRenderer{
		response: []byte("unused"),
	}
	s := testServer(t, time.Second, withStoryRenderer(renderer))

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/story?path="+url.QueryEscape("/api/v1/stats"),
		nil,
	)
	req.Host = net.JoinHostPort("127.0.0.1", "0")
	req.RemoteAddr = "127.0.0.1:1234"
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	assertRecorderStatus(t, w, http.StatusBadRequest)
	if !strings.Contains(w.Body.String(), "app page") {
		t.Fatalf("body = %q, want app page error", w.Body.String())
	}
	calls, _ := renderer.snapshot()
	if calls != 0 {
		t.Fatalf("renderer calls = %d, want 0", calls)
	}
}

func TestHandleStoryRendererDeadline(t *testing.T) {
	t.Parallel()

	renderer := &fakeStoryRenderer{
		err: context.DeadlineExceeded,
	}
	s := testServer(t, time.Second, withStoryRenderer(renderer))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/story", nil)
	req.Host = net.JoinHostPort("127.0.0.1", "0")
	req.RemoteAddr = "127.0.0.1:1234"
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	assertRecorderStatus(t, w, http.StatusGatewayTimeout)
	if !strings.Contains(w.Body.String(), "gateway timeout") {
		t.Fatalf("body = %q, want gateway timeout", w.Body.String())
	}
}

func TestHandleStoryRendererError(t *testing.T) {
	t.Parallel()

	renderer := &fakeStoryRenderer{
		err: errors.New("chrome unavailable"),
	}
	s := testServer(t, time.Second, withStoryRenderer(renderer))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/story", nil)
	req.Host = net.JoinHostPort("127.0.0.1", "0")
	req.RemoteAddr = "127.0.0.1:1234"
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	assertRecorderStatus(t, w, http.StatusInternalServerError)
	if !strings.Contains(w.Body.String(), "chrome unavailable") {
		t.Fatalf("body = %q, want renderer error", w.Body.String())
	}
}

func TestStoryLoopbackHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		host string
		want string
	}{
		{host: "", want: "127.0.0.1"},
		{host: "0.0.0.0", want: "127.0.0.1"},
		{host: "::", want: "127.0.0.1"},
		{host: "localhost", want: "localhost"},
		{host: "::1", want: "::1"},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			t.Parallel()
			if got := storyLoopbackHost(tt.host); got != tt.want {
				t.Fatalf("host = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStoryNetworkIdleIgnoresLongLivedResources(t *testing.T) {
	t.Parallel()

	waiter := newStoryNetworkIdleWaiter()
	waiter.listen(&network.EventRequestWillBeSent{
		RequestID: "events",
		Type:      network.ResourceTypeEventSource,
	})
	waiter.listen(&network.EventRequestWillBeSent{
		RequestID: "api",
		Type:      network.ResourceTypeFetch,
	})
	if waiter.isIdle(0) {
		t.Fatalf("waiter was idle with fetch in flight")
	}
	waiter.listen(&network.EventLoadingFinished{
		RequestID: "api",
	})
	if !waiter.isIdle(0) {
		t.Fatalf("waiter did not ignore event source")
	}
}

func TestSetLocalStorageExpressionEscapesValues(t *testing.T) {
	t.Parallel()

	expr := setLocalStorageExpression(
		`agentsview-auth-token`,
		`quote"slash\newline
`,
	)
	if !strings.Contains(expr, `localStorage.setItem`) {
		t.Fatalf("expression = %q, want localStorage call", expr)
	}
	if strings.Contains(expr, "\n") {
		t.Fatalf("expression contains raw newline: %q", expr)
	}
}

func TestStoryNetworkIdleWaitRespectsContext(t *testing.T) {
	t.Parallel()

	waiter := newStoryNetworkIdleWaiter()
	waiter.listen(&network.EventRequestWillBeSent{
		RequestID: "api",
		Type:      network.ResourceTypeFetch,
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	if err := waiter.wait(time.Hour, time.Hour).Do(ctx); err == nil {
		t.Fatalf("expected context error")
	}
}
