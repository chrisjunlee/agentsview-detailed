package server

import (
	"strings"
	"testing"
)

// TestParseUsageFilterTimeParams exercises the from_time/to_time
// parameter parsing in parseUsageFilter: valid values must
// populate UsageFilter.FromTime/ToTime, and invalid values must
// return HTTP 400 with a body that names the offending parameter.
func TestParseUsageFilterTimeParams(t *testing.T) {
	t.Run("valid from_time and to_time populate filter", func(t *testing.T) {
		w, r := newTestRequest(t,
			"from=2026-05-30&to=2026-05-30&from_time=09:15&to_time=14:30")
		f, ok := parseUsageFilter(w, r)
		if !ok {
			t.Fatalf("parseUsageFilter returned false, status %d: %s",
				w.Code, w.Body.String())
		}
		if f.FromTime != "09:15" {
			t.Errorf("FromTime = %q, want %q", f.FromTime, "09:15")
		}
		if f.ToTime != "14:30" {
			t.Errorf("ToTime = %q, want %q", f.ToTime, "14:30")
		}
	})

	t.Run("invalid from_time returns 400 mentioning from_time", func(t *testing.T) {
		w, r := newTestRequest(t,
			"from=2026-05-30&to=2026-05-30&from_time=25:00")
		_, ok := parseUsageFilter(w, r)
		if ok {
			t.Fatal("parseUsageFilter returned true, want false for bad from_time")
		}
		assertRecorderStatus(t, w, 400)
		body := w.Body.String()
		if !strings.Contains(body, "from_time") {
			t.Errorf("400 body = %q, want it to mention %q", body, "from_time")
		}
	})

	t.Run("invalid to_time returns 400 mentioning to_time", func(t *testing.T) {
		w, r := newTestRequest(t,
			"from=2026-05-30&to=2026-05-30&to_time=9:9")
		_, ok := parseUsageFilter(w, r)
		if ok {
			t.Fatal("parseUsageFilter returned true, want false for bad to_time")
		}
		assertRecorderStatus(t, w, 400)
		body := w.Body.String()
		if !strings.Contains(body, "to_time") {
			t.Errorf("400 body = %q, want it to mention %q", body, "to_time")
		}
	})
}
