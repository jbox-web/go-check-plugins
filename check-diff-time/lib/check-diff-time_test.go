package checkdifftime

import (
	"time"

	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/jessevdk/go-flags"

	"github.com/mackerelio/checkers"
	"golang.org/x/crypto/ssh"
)

func TestClockDiff(t *testing.T) {
	remote, diff, err := clockDiff([]byte("1000\n"), 1003)
	if err != nil || remote != 1000 || diff != 3 {
		t.Fatalf("clockDiff(1000, now 1003) = %d, %d, %v; want 1000, 3, nil", remote, diff, err)
	}
	_, diff, err = clockDiff([]byte("1005\r\n"), 1003)
	if err != nil || diff != 2 {
		t.Fatalf("clockDiff(remote ahead) diff = %d, %v; want 2, nil", diff, err)
	}
}

func TestClockDiffRejectsGarbage(t *testing.T) {
	if _, _, err := clockDiff([]byte("date: command not found\n"), 1003); err == nil {
		t.Fatal("clockDiff(garbage): want an error, got none")
	}
}

func TestDiffStatus(t *testing.T) {
	cases := []struct {
		diff int64
		want checkers.Status
	}{
		{0, checkers.OK},
		{4, checkers.OK},
		{5, checkers.WARNING},
		{9, checkers.WARNING},
		{10, checkers.CRITICAL},
		{3600, checkers.CRITICAL},
	}
	for _, tc := range cases {
		if got := diffStatus(tc.diff, 5, 10); got != tc.want {
			t.Errorf("diffStatus(%d, 5, 10) = %v, want %v", tc.diff, got, tc.want)
		}
	}
}

// writeKey writes an ed25519 private key in OpenSSH format, encrypted when
// passphrase is not empty.
func writeKey(t *testing.T, passphrase string) string {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	var block *pem.Block
	if passphrase == "" {
		block, err = ssh.MarshalPrivateKey(priv, "")
	} else {
		block, err = ssh.MarshalPrivateKeyWithPassphrase(priv, "", []byte(passphrase))
	}
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestMakeClientConfigKeys(t *testing.T) {
	plain := &sshOpts{User: "nagios", IdentityFile: writeKey(t, "")}
	if _, err := plain.makeClientConfig(); err != nil {
		t.Errorf("unencrypted OpenSSH key: %v", err)
	}
	encrypted := &sshOpts{User: "nagios", IdentityFile: writeKey(t, "s3cret"), PassPhrase: "s3cret"}
	if _, err := encrypted.makeClientConfig(); err != nil {
		t.Errorf("encrypted OpenSSH key with its passphrase: %v", err)
	}
	wrong := &sshOpts{User: "nagios", IdentityFile: writeKey(t, "s3cret"), PassPhrase: "nope"}
	if _, err := wrong.makeClientConfig(); err == nil {
		t.Error("encrypted OpenSSH key with a wrong passphrase: want an error, got none")
	}
}

func TestHostKeyCallback(t *testing.T) {
	if _, err := (&sshOpts{}).hostKeyCallback(); err != nil {
		t.Errorf("no known_hosts: %v", err)
	}
	missing := &sshOpts{KnownHosts: filepath.Join(t.TempDir(), "known_hosts")}
	if _, err := missing.hostKeyCallback(); err == nil {
		t.Error("missing known_hosts file: want an error, got none")
	}
}

// A passphrase exported for every check must not break an unencrypted key.
func TestMakeClientConfigUnencryptedKeyWithPassphrase(t *testing.T) {
	opts := &sshOpts{User: "nagios", IdentityFile: writeKey(t, ""), PassPhrase: "unused"}
	if _, err := opts.makeClientConfig(); err != nil {
		t.Fatalf("unencrypted key with a passphrase set: %v", err)
	}
}

func TestDeadline(t *testing.T) {
	now := time.Unix(1000, 0)
	if got := (&sshOpts{Timeout: 30}).deadline(now); !got.Equal(now.Add(30 * time.Second)) {
		t.Errorf("deadline(-t 30) = %v, want %v", got, now.Add(30*time.Second))
	}
	if got := (&sshOpts{Timeout: 0}).deadline(now); !got.IsZero() {
		t.Errorf("deadline(-t 0) = %v, want no deadline", got)
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
