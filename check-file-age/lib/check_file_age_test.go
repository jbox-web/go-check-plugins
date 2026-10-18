package checkfileage

import (
	"strings"
	"testing"

	"github.com/jessevdk/go-flags"
)

func TestSecondsToHuman(t *testing.T) {
	const day = 86400
	cases := []struct {
		seconds int64
		want    string
	}{
		{0, "0 second"},
		{59, "59 seconds"},
		{90, "1 minute 30 seconds"},
		{3600, "1 hour 0 minute 0 second"},
		{2*day + 3*3600, "2 days 3 hours 0 minute 0 second"},
		{8 * day, "1 week 1 day 0 hour 0 minute 0 second"},
		{31 * day, "1 month 0 week 1 day 0 hour 0 minute 0 second"},
		{400 * day, "1 year 1 month 0 week 5 days 0 hour 0 minute 0 second"},
	}
	for _, tc := range cases {
		got := strings.TrimSpace(secondsToHuman(tc.seconds))
		if got != tc.want {
			t.Errorf("secondsToHuman(%d) = %q, want %q", tc.seconds, got, tc.want)
		}
	}
}

func TestMonitorThresholds(t *testing.T) {
	m := newMonitor(240, 0, 600, 10)
	if m.CheckWarning(100, 50) {
		t.Error("100s old, 50 bytes: want no warning")
	}
	if !m.CheckWarning(300, 50) {
		t.Error("300s old: want a warning")
	}
	if !m.CheckCritical(700, 50) {
		t.Error("700s old: want a critical")
	}
	if !m.CheckCritical(100, 5) {
		t.Error("5 bytes, below the 10 bytes critical size: want a critical")
	}
}

func TestExitCode(t *testing.T) {
	if got := exitCode(&flags.Error{Type: flags.ErrHelp}); got != 0 {
		t.Errorf("exitCode(--help) = %d, want 0", got)
	}
	if got := exitCode(&flags.Error{Type: flags.ErrUnknownFlag}); got != 3 {
		t.Errorf("exitCode(unknown flag) = %d, want 3 (UNKNOWN)", got)
	}
}
