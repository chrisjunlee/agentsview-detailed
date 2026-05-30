package server

import (
	"strings"
	"testing"
)

// TestParseAnalyticsFilter_ValidTimeParams asserts that valid
// from_time and to_time query params are parsed into the filter
// without error.
func TestParseAnalyticsFilter_ValidTimeParams(t *testing.T) {
	w, r := newTestRequest(t,
		"from=2026-05-30&to=2026-05-30&from_time=09:15&to_time=14:30")
	f, ok := parseAnalyticsFilter(w, r)
	if !ok {
		t.Fatalf(
			"parseAnalyticsFilter returned false (status %d): %s",
			w.Code, w.Body.String(),
		)
	}
	if f.FromTime != "09:15" {
		t.Errorf("FromTime = %q, want %q", f.FromTime, "09:15")
	}
	if f.ToTime != "14:30" {
		t.Errorf("ToTime = %q, want %q", f.ToTime, "14:30")
	}
}

// TestParseAnalyticsFilter_EmptyTimeParams asserts that omitting
// from_time and to_time leaves those fields empty (no error).
func TestParseAnalyticsFilter_EmptyTimeParams(t *testing.T) {
	w, r := newTestRequest(t, "from=2026-05-30&to=2026-05-30")
	f, ok := parseAnalyticsFilter(w, r)
	if !ok {
		t.Fatalf(
			"parseAnalyticsFilter returned false (status %d): %s",
			w.Code, w.Body.String(),
		)
	}
	if f.FromTime != "" {
		t.Errorf("FromTime = %q, want empty", f.FromTime)
	}
	if f.ToTime != "" {
		t.Errorf("ToTime = %q, want empty", f.ToTime)
	}
}

// TestParseAnalyticsFilter_InvalidFromTime asserts that an invalid
// from_time returns HTTP 400 with the exact body
// "invalid from_time: use HH:MM".
func TestParseAnalyticsFilter_InvalidFromTime(t *testing.T) {
	invalidCases := []struct {
		name  string
		value string
	}{
		{"hour out of range", "25:00"},
		{"minute out of range", "12:60"},
		{"single-digit hour", "9:15"},
		{"includes seconds", "09:15:00"},
		{"empty string treated as not-present so skip", ""},
	}

	for _, tc := range invalidCases {
		tc := tc
		if tc.value == "" {
			// Empty from_time is not invalid — skip.
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			w, r := newTestRequest(t,
				"from=2026-05-30&to=2026-05-30&from_time="+tc.value)
			_, ok := parseAnalyticsFilter(w, r)
			if ok {
				t.Fatal(
					"parseAnalyticsFilter returned true, want false for invalid from_time",
				)
			}
			assertRecorderStatus(t, w, 400)
			body := w.Body.String()
			wantMsg := "invalid from_time: use HH:MM"
			if !strings.Contains(body, wantMsg) {
				t.Errorf("body = %q, want it to contain %q", body, wantMsg)
			}
		})
	}
}

// TestParseAnalyticsFilter_InvalidToTime asserts that an invalid
// to_time returns HTTP 400 with the exact body
// "invalid to_time: use HH:MM".
func TestParseAnalyticsFilter_InvalidToTime(t *testing.T) {
	invalidCases := []struct {
		name  string
		value string
	}{
		{"hour out of range", "24:00"},
		{"single-digit minute", "12:5"},
		{"includes seconds", "14:30:00"},
	}

	for _, tc := range invalidCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			w, r := newTestRequest(t,
				"from=2026-05-30&to=2026-05-30&to_time="+tc.value)
			_, ok := parseAnalyticsFilter(w, r)
			if ok {
				t.Fatal(
					"parseAnalyticsFilter returned true, want false for invalid to_time",
				)
			}
			assertRecorderStatus(t, w, 400)
			body := w.Body.String()
			wantMsg := "invalid to_time: use HH:MM"
			if !strings.Contains(body, wantMsg) {
				t.Errorf("body = %q, want it to contain %q", body, wantMsg)
			}
		})
	}
}

// TestParseAnalyticsFilter_ErrorMessageExact pins the exact error
// message text returned for from_time and to_time validation failures.
// If a sibling changes the error strings this test immediately breaks.
func TestParseAnalyticsFilter_ErrorMessageExact(t *testing.T) {
	t.Run("from_time exact message", func(t *testing.T) {
		w, r := newTestRequest(t,
			"from=2026-05-30&to=2026-05-30&from_time=99:99")
		_, ok := parseAnalyticsFilter(w, r)
		if ok {
			t.Fatal("expected false, got true")
		}
		body := w.Body.String()
		// The exact message (as a JSON-encoded string within the body).
		want := "invalid from_time: use HH:MM"
		if !strings.Contains(body, want) {
			t.Errorf("body = %q; want it to contain %q", body, want)
		}
		// Must NOT contain the to_time message.
		notWant := "invalid to_time"
		if strings.Contains(body, notWant) {
			t.Errorf("body = %q; should NOT contain %q", body, notWant)
		}
	})

	t.Run("to_time exact message", func(t *testing.T) {
		w, r := newTestRequest(t,
			"from=2026-05-30&to=2026-05-30&to_time=99:99")
		_, ok := parseAnalyticsFilter(w, r)
		if ok {
			t.Fatal("expected false, got true")
		}
		body := w.Body.String()
		want := "invalid to_time: use HH:MM"
		if !strings.Contains(body, want) {
			t.Errorf("body = %q; want it to contain %q", body, want)
		}
		notWant := "invalid from_time"
		if strings.Contains(body, notWant) {
			t.Errorf("body = %q; should NOT contain %q", body, notWant)
		}
	})
}
