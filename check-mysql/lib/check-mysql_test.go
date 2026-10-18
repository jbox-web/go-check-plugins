package checkmysql

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jessevdk/go-flags"
)

func TestSeparateSub(t *testing.T) {
	cases := []struct {
		argv     []string
		wantSub  string
		wantArgs []string
	}{
		{[]string{"uptime", "-c", "60"}, "uptime", []string{"-c", "60"}},
		{[]string{"-H", "db"}, "", []string{"-H", "db"}},
		{nil, "", nil},
	}
	for _, tc := range cases {
		sub, args := separateSub(tc.argv)
		if sub != tc.wantSub || !reflect.DeepEqual(args, tc.wantArgs) {
			t.Errorf("separateSub(%v) = %q, %v; want %q, %v", tc.argv, sub, args, tc.wantSub, tc.wantArgs)
		}
	}
}

func TestUptime2str(t *testing.T) {
	if got := uptime2str(2*86400 + 3*3600 + 4*60 + 5); got != "2 days, 03:04:05" {
		t.Errorf("uptime2str() = %q, want %q", got, "2 days, 03:04:05")
	}
}

func TestNewDBRejectsUnreadableRootCert(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(path, []byte("not a certificate"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := newDB(mysqlSetting{EnableTLS: true, TLSRootCert: path}); err == nil {
		t.Fatal("newDB with a root cert file holding no certificate: want an error, got none")
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
