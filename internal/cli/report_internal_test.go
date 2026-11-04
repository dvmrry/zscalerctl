package cli

import (
	"errors"
	"testing"
	"time"
)

// TestResolveReportWindow pins window resolution: --since ends now (whole
// seconds, UTC), days are accepted, and --from/--to win only as a pair
// without --since.
func TestResolveReportWindow(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 26, 12, 30, 45, 900, time.FixedZone("x", 3600))
	for _, c := range []struct {
		name     string
		flags    reportFlags
		from, to string
		usage    bool
	}{
		{name: "default", flags: reportFlags{since: "2h"}, from: "2026-10-26T09:30:45Z", to: "2026-10-26T11:30:45Z"},
		{name: "days", flags: reportFlags{since: "7d", sinceSet: true}, from: "2026-10-19T11:30:45Z", to: "2026-10-26T11:30:45Z"},
		{name: "minutes", flags: reportFlags{since: "90m", sinceSet: true}, from: "2026-10-26T10:00:45Z", to: "2026-10-26T11:30:45Z"},
		{name: "explicit", flags: reportFlags{since: "2h", from: "2026-10-01T00:00:00+02:00", to: "2026-10-02T00:00:00Z"},
			from: "2026-09-30T22:00:00Z", to: "2026-10-02T00:00:00Z"},
		{name: "from without to", flags: reportFlags{since: "2h", from: "2026-10-01T00:00:00Z"}, usage: true},
		{name: "since with pair", flags: reportFlags{since: "1h", sinceSet: true, from: "2026-10-01T00:00:00Z", to: "2026-10-02T00:00:00Z"}, usage: true},
		{name: "reversed", flags: reportFlags{since: "2h", from: "2026-10-02T00:00:00Z", to: "2026-10-01T00:00:00Z"}, usage: true},
		{name: "bad time", flags: reportFlags{since: "2h", from: "yesterday", to: "2026-10-01T00:00:00Z"}, usage: true},
		{name: "bad since", flags: reportFlags{since: "two hours", sinceSet: true}, usage: true},
		{name: "tiny since", flags: reportFlags{since: "30s", sinceSet: true}, usage: true},
		{name: "negative", flags: reportFlags{since: "-2h", sinceSet: true}, usage: true},
		// 213504d once overflowed to a 25-minute window.
		{name: "overflowing days", flags: reportFlags{since: "213504d", sinceSet: true}, usage: true},
		{name: "huge days", flags: reportFlags{since: "9999999d", sinceSet: true}, usage: true},
		{name: "past the cap", flags: reportFlags{since: "91d", sinceSet: true}, usage: true},
		{name: "at the cap", flags: reportFlags{since: "90d", sinceSet: true}, from: "2026-07-28T11:30:45Z", to: "2026-10-26T11:30:45Z"},
		{name: "fractional pair", flags: reportFlags{since: "2h", from: "2026-10-01T00:00:00.5Z", to: "2026-10-01T00:00:00.9Z"}, usage: true},
		{name: "truncated pair", flags: reportFlags{since: "2h", from: "2026-10-01T00:00:00.5Z", to: "2026-10-01T00:00:02.9Z"},
			from: "2026-10-01T00:00:00Z", to: "2026-10-01T00:00:02Z"},
		{name: "epoch start", flags: reportFlags{since: "2h", from: "1970-01-01T00:00:00Z", to: "2026-10-01T00:00:00Z"}, usage: true},
		{name: "future end", flags: reportFlags{since: "2h", from: "2026-10-26T00:00:00Z", to: "2026-10-27T00:00:00Z"}, usage: true},
	} {
		from, to, err := resolveReportWindow(now, c.flags)
		if c.usage {
			if !errors.As(err, new(UsageError)) {
				t.Errorf("%s: error = %v, want UsageError", c.name, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s: error = %v", c.name, err)
		}
		if got := from.Format(time.RFC3339); got != c.from {
			t.Errorf("%s: from = %s, want %s", c.name, got, c.from)
		}
		if got := to.Format(time.RFC3339); got != c.to {
			t.Errorf("%s: to = %s, want %s", c.name, got, c.to)
		}
	}
}
