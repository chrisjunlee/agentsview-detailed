package server

// story_negotiation_test.go: characterization tests for content-negotiation
// and path-parsing behavior of the story screenshot endpoint.
//
// Gaps covered here (confirmed against story_test.go and middleware_test.go):
//   - wantsBrowserPage: unit tests for every Accept/format combination
//   - parseStoryAppPath: exact returned *url.URL and error for additional
//     representative inputs not covered by story_test.go's table
//   - End-to-end content negotiation via GET /api/v1/story with Accept:
//     text/html  → HTML page branch (injected fake renderer never called)
//   - End-to-end negotiation with format=html query param  → HTML branch
//   - GET /api/v1/story with format=png  → PNG branch
//   - Missing Authorization header  → AuthToken == ""
//
// Deliberately skipped (already in story_test.go):
//   - default path "", path with query, absolute URL, protocol-relative,
//     relative, /api/…, /story, fragment, control character in parseStoryAppPath
//   - TestHandleStoryRendersPNG, TestHandleStoryRendererDeadline,
//     TestHandleStoryRendererError, TestHandleStoryRejectsInvalidPath,
//     TestHandleStoryPageRendersBrowserWrapper, TestHandleStoryPageRejectsInvalidPath
//   - TestStoryLoopbackHost

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// wantsBrowserPage unit tests
// ---------------------------------------------------------------------------

// TestWantsBrowserPageAcceptHeader tests the exact boolean returned by
// wantsBrowserPage for every Accept-header variant, when no format param
// is present (so the Accept-header branch is reached).
func TestWantsBrowserPageAcceptHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		accept     string
		wantResult bool
	}{
		{
			name:       "text/html returns true",
			accept:     "text/html",
			wantResult: true,
		},
		{
			name:       "text/html with quality suffix returns true",
			accept:     "text/html,application/xhtml+xml,application/xml;q=0.9",
			wantResult: true,
		},
		{
			name:       "application/json returns false",
			accept:     "application/json",
			wantResult: false,
		},
		{
			name:       "wildcard star-slash-star returns false",
			accept:     "*/*",
			wantResult: false,
		},
		{
			name:       "empty Accept returns false",
			accept:     "",
			wantResult: false,
		},
		{
			name:       "image/png returns false",
			accept:     "image/png",
			wantResult: false,
		},
		{
			name:       "application/json then text/html returns true",
			accept:     "application/json, text/html",
			wantResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/story", nil)
			if tt.accept != "" {
				req.Header.Set("Accept", tt.accept)
			}
			got := wantsBrowserPage(req)
			if got != tt.wantResult {
				t.Fatalf(
					"wantsBrowserPage with Accept=%q = %v, want %v",
					tt.accept, got, tt.wantResult,
				)
			}
		})
	}
}

// TestWantsBrowserPageFormatParam tests that the format query param takes
// priority over the Accept header in wantsBrowserPage.
func TestWantsBrowserPageFormatParam(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		format     string
		accept     string // extra Accept header, may conflict to confirm priority
		wantResult bool
	}{
		{
			name:       "format=html returns true",
			format:     "html",
			wantResult: true,
		},
		{
			name:       "format=png returns false",
			format:     "png",
			wantResult: false,
		},
		{
			name:       "format=json returns false",
			format:     "json",
			wantResult: false,
		},
		{
			name:       "format=html overrides application/json Accept",
			format:     "html",
			accept:     "application/json",
			wantResult: true,
		},
		{
			name:       "format=png overrides text/html Accept",
			format:     "png",
			accept:     "text/html",
			wantResult: false,
		},
		{
			name:       "empty format falls through to Accept header (text/html)",
			format:     "",
			accept:     "text/html",
			wantResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			target := "/api/v1/story"
			if tt.format != "" {
				target += "?format=" + tt.format
			}
			req := httptest.NewRequest(http.MethodGet, target, nil)
			if tt.accept != "" {
				req.Header.Set("Accept", tt.accept)
			}
			got := wantsBrowserPage(req)
			if got != tt.wantResult {
				t.Fatalf(
					"wantsBrowserPage format=%q accept=%q = %v, want %v",
					tt.format, tt.accept, got, tt.wantResult,
				)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// parseStoryAppPath additional cases (gaps only)
// ---------------------------------------------------------------------------

// TestParseStoryAppPathGaps covers exact returned values for inputs not
// exercised by the existing TestParseStoryAppPath in story_test.go.
func TestParseStoryAppPathGaps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    string // expected url.URL.String(); checked only when wantErr=false
		wantErr bool
	}{
		{
			// /story/sub path must be rejected (starts with /story/)
			name:    "story sub-path rejected",
			raw:     "/story/highlights",
			wantErr: true,
		},
		{
			// Pure /usage path with no query: url.URL.String() == "/usage"
			name: "plain path no query",
			raw:  "/usage",
			want: "/usage",
		},
		{
			// Path traversal that resolves away from /api or /story is allowed.
			// /sessions/../usage cleans to /usage — allowed.
			name: "traversal resolving to allowed path is accepted",
			raw:  "/sessions/../usage",
			want: "/sessions/../usage",
		},
		{
			// /sessions path with multiple query params: URL preserved as-is.
			name: "path with multiple query params",
			raw:  "/sessions?from=2026-01-01&to=2026-01-31&model=gpt-4o",
			want: "/sessions?from=2026-01-01&to=2026-01-31&model=gpt-4o",
		},
		{
			// path.Clean("/usage/") == "/usage" but parseStoryAppPath preserves
			// the original u.Path from url.Parse which keeps the trailing slash.
			// The cleaned check only applies to the /api and /story guards.
			name: "trailing slash path",
			raw:  "/usage/",
			want: "/usage/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseStoryAppPath(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got nil (url=%v)", tt.raw, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.raw, err)
			}
			if got.String() != tt.want {
				t.Fatalf("parseStoryAppPath(%q).String() = %q, want %q",
					tt.raw, got.String(), tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// End-to-end content-negotiation routing through /api/v1/story
// ---------------------------------------------------------------------------

// TestHandleStoryAcceptHTMLRoutesToPage verifies that a GET /api/v1/story
// request with Accept: text/html is routed to the handleStoryPage branch:
// it returns HTML, never calls the fake renderer, and includes the story
// iframe in the body.
func TestHandleStoryAcceptHTMLRoutesToPage(t *testing.T) {
	t.Parallel()

	renderer := &fakeStoryRenderer{
		response: []byte{0x89, 'P', 'N', 'G'},
	}
	s := testServer(t, time.Second, withStoryRenderer(renderer))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/story", nil)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Host = net.JoinHostPort("127.0.0.1", "0")
	req.RemoteAddr = "127.0.0.1:1234"
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	// Must return HTML, not PNG
	assertRecorderStatus(t, w, http.StatusOK)
	assertContentType(t, w, "text/html; charset=utf-8")

	// The fake renderer must NOT have been called
	calls, _ := renderer.snapshot()
	if calls != 0 {
		t.Fatalf("renderer calls = %d, want 0 for HTML branch", calls)
	}

	// Body must contain the iframe story wrapper
	body := w.Body.String()
	if !strings.Contains(body, "<iframe") {
		t.Fatalf("expected iframe in HTML story page body; got:\n%s", body)
	}
	if !strings.Contains(body, "agentsview story") {
		t.Fatalf("expected 'agentsview story' title in HTML page body; got:\n%s", body)
	}
	if got := w.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
}

// TestHandleStoryFormatHTMLRoutesToPage verifies that format=html query
// param routes to the browser page branch, regardless of Accept header.
func TestHandleStoryFormatHTMLRoutesToPage(t *testing.T) {
	t.Parallel()

	renderer := &fakeStoryRenderer{
		response: []byte{0x89, 'P', 'N', 'G'},
	}
	s := testServer(t, time.Second, withStoryRenderer(renderer))

	req := httptest.NewRequest(
		http.MethodGet, "/api/v1/story?format=html", nil,
	)
	// Deliberately set Accept to application/json to confirm format= wins
	req.Header.Set("Accept", "application/json")
	req.Host = net.JoinHostPort("127.0.0.1", "0")
	req.RemoteAddr = "127.0.0.1:1234"
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	assertRecorderStatus(t, w, http.StatusOK)
	assertContentType(t, w, "text/html; charset=utf-8")

	calls, _ := renderer.snapshot()
	if calls != 0 {
		t.Fatalf("renderer calls = %d, want 0 for HTML branch", calls)
	}

	if !strings.Contains(w.Body.String(), "<iframe") {
		t.Fatalf("expected iframe in HTML story page body")
	}
}

// TestHandleStoryFormatPNGRoutesToRenderer verifies that format=png (or any
// non-html format param) routes to the renderer branch and returns PNG.
func TestHandleStoryFormatPNGRoutesToRenderer(t *testing.T) {
	t.Parallel()

	pngBytes := []byte{0x89, 'P', 'N', 'G', 0x0d}
	renderer := &fakeStoryRenderer{response: pngBytes}
	s := testServer(t, time.Second, withStoryRenderer(renderer))

	// Use format=png and also set Accept: text/html to confirm format= wins
	req := httptest.NewRequest(
		http.MethodGet, "/api/v1/story?format=png", nil,
	)
	req.Header.Set("Accept", "text/html")
	req.Host = net.JoinHostPort("127.0.0.1", "0")
	req.RemoteAddr = "127.0.0.1:1234"
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	assertRecorderStatus(t, w, http.StatusOK)
	assertContentType(t, w, "image/png")

	calls, _ := renderer.snapshot()
	if calls != 1 {
		t.Fatalf("renderer calls = %d, want 1 for PNG branch", calls)
	}

	if got := w.Body.Bytes(); string(got) != string(pngBytes) {
		t.Fatalf("body = %v, want %v", got, pngBytes)
	}
}

// TestHandleStoryApplicationJSONAcceptRoutesToRenderer verifies that an
// explicit Accept: application/json header (no format param) does NOT route
// to the HTML page branch and instead calls the renderer.
func TestHandleStoryApplicationJSONAcceptRoutesToRenderer(t *testing.T) {
	t.Parallel()

	pngBytes := []byte{0x89, 'P', 'N', 'G'}
	renderer := &fakeStoryRenderer{response: pngBytes}
	s := testServer(t, time.Second, withStoryRenderer(renderer))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/story", nil)
	req.Header.Set("Accept", "application/json")
	req.Host = net.JoinHostPort("127.0.0.1", "0")
	req.RemoteAddr = "127.0.0.1:1234"
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	assertRecorderStatus(t, w, http.StatusOK)
	assertContentType(t, w, "image/png")

	calls, _ := renderer.snapshot()
	if calls != 1 {
		t.Fatalf("renderer calls = %d, want 1 for non-HTML Accept", calls)
	}
}

// TestHandleStoryNoAcceptHeaderRoutesToRenderer verifies that an absent
// Accept header routes to the renderer branch (not HTML).
func TestHandleStoryNoAcceptHeaderRoutesToRenderer(t *testing.T) {
	t.Parallel()

	pngBytes := []byte{0x89, 'P', 'N', 'G'}
	renderer := &fakeStoryRenderer{response: pngBytes}
	s := testServer(t, time.Second, withStoryRenderer(renderer))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/story", nil)
	// Explicitly do NOT set Accept
	req.Host = net.JoinHostPort("127.0.0.1", "0")
	req.RemoteAddr = "127.0.0.1:1234"
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	assertRecorderStatus(t, w, http.StatusOK)
	assertContentType(t, w, "image/png")

	calls, _ := renderer.snapshot()
	if calls != 1 {
		t.Fatalf("renderer calls = %d, want 1 for absent Accept", calls)
	}
}

// TestHandleStoryNoAuthTokenPassesEmptyToken verifies that when no
// Authorization header is present the AuthToken forwarded to the renderer
// is exactly the empty string "".
func TestHandleStoryNoAuthTokenPassesEmptyToken(t *testing.T) {
	t.Parallel()

	renderer := &fakeStoryRenderer{
		response: []byte{0x89, 'P', 'N', 'G'},
	}
	s := testServer(t, time.Second, withStoryRenderer(renderer))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/story", nil)
	req.Host = net.JoinHostPort("127.0.0.1", "0")
	req.RemoteAddr = "127.0.0.1:1234"
	// No Authorization header
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	assertRecorderStatus(t, w, http.StatusOK)

	_, gotReq := renderer.snapshot()
	if gotReq.AuthToken != "" {
		t.Fatalf("AuthToken = %q, want empty string for missing header", gotReq.AuthToken)
	}
}

// TestHandleStoryWildcardAcceptRoutesToRenderer verifies that Accept: */*
// (the default browser implicit accept) routes to the renderer, not HTML.
func TestHandleStoryWildcardAcceptRoutesToRenderer(t *testing.T) {
	t.Parallel()

	pngBytes := []byte{0x89, 'P', 'N', 'G'}
	renderer := &fakeStoryRenderer{response: pngBytes}
	s := testServer(t, time.Second, withStoryRenderer(renderer))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/story", nil)
	req.Header.Set("Accept", "*/*")
	req.Host = net.JoinHostPort("127.0.0.1", "0")
	req.RemoteAddr = "127.0.0.1:1234"
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	assertRecorderStatus(t, w, http.StatusOK)
	assertContentType(t, w, "image/png")

	calls, _ := renderer.snapshot()
	if calls != 1 {
		t.Fatalf("renderer calls = %d, want 1 for */* Accept", calls)
	}
}
