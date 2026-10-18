package checkmemory

import (
	"testing"

	"github.com/jessevdk/go-flags"

	"github.com/mackerelio/checkers"
)

func TestCheckStatus(t *testing.T) {
	cases := []struct {
		name string
		used float64
		want checkers.Status
	}{
		{"below warning", 50, checkers.OK},
		{"at warning", 95, checkers.WARNING},
		{"between thresholds", 97, checkers.WARNING},
		{"at critical", 98, checkers.CRITICAL},
		{"above critical", 99.5, checkers.CRITICAL},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := checkStatus(tc.used, "95", "98")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("checkStatus(%v) = %v, want %v", tc.used, got, tc.want)
			}
		})
	}
}

func TestCheckStatusInvalidThreshold(t *testing.T) {
	for _, th := range [][2]string{{"x", "98"}, {"95", "y"}} {
		got, err := checkStatus(50, th[0], th[1])
		if err == nil {
			t.Errorf("checkStatus with thresholds %v: want an error, got none", th)
		}
		if got != checkers.UNKNOWN {
			t.Errorf("checkStatus with thresholds %v = %v, want UNKNOWN", th, got)
		}
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
