package checkdifftime

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/mackerelio/checkers"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type sshOpts struct {
	Hostname     string  `short:"H" long:"hostname" default:"localhost" description:"Host name or IP Address"`
	Port         int     `short:"P" long:"port" default:"22" description:"Port number"`
	Timeout      float64 `short:"t" long:"timeout" default:"30" description:"Seconds before the whole SSH exchange (connection, handshake, command) times out"`
	Warning      int64   `short:"w" long:"warning" default:"5" description:"Clock difference to result in warning status (seconds)"`
	Critical     int64   `short:"c" long:"critical" default:"10" description:"Clock difference to result in critical status (seconds)"`
	User         string  `short:"u" long:"user" description:"Login user name" env:"USER"`
	Password     string  `short:"p" long:"password" description:"Login password" env:"LOGIN_PASSWORD"`
	IdentityFile string  `short:"i" long:"identity" description:"Identity file (ssh private key)"`
	PassPhrase   string  `long:"passphrase" description:"Identity passphrase" env:"CHECK_SSH_IDENTITY_PASSPHRASE"`
	KnownHosts   string  `long:"known-hosts" description:"known_hosts file to verify the host key against (not verified when empty); it must hold the key type the server negotiates first"`
}

// Do the plugin
func Do() {
	opts, err := parseArgs(os.Args[1:])
	if err != nil {
		os.Exit(exitCode(err))
	}

	ckr := opts.run()
	ckr.Name = "Diff Time"
	ckr.Exit()
}

func parseArgs(args []string) (*sshOpts, error) {
	opts := &sshOpts{}
	_, err := flags.ParseArgs(opts, args)
	return opts, err
}

// readSigner loads the private key (OpenSSH and PEM formats). The passphrase
// is only used when the key asks for one: an unencrypted key still loads when
// a passphrase is set, e.g. exported for every check.
func readSigner(file, passphrase string) (ssh.Signer, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(data)
	var missing *ssh.PassphraseMissingError
	if errors.As(err, &missing) && passphrase != "" {
		return ssh.ParsePrivateKeyWithPassphrase(data, []byte(passphrase))
	}
	return signer, err
}

func (opts *sshOpts) hostKeyCallback() (ssh.HostKeyCallback, error) {
	if opts.KnownHosts == "" {
		return ssh.InsecureIgnoreHostKey(), nil
	}
	return knownhosts.New(opts.KnownHosts)
}

func (opts *sshOpts) makeClientConfig() (*ssh.ClientConfig, error) {
	authenticities := make([]ssh.AuthMethod, 0, 2)
	if opts.Password != "" {
		authenticities = append(authenticities, ssh.Password(opts.Password))
	}
	if opts.IdentityFile != "" {
		signer, err := readSigner(opts.IdentityFile, opts.PassPhrase)
		if err != nil {
			return nil, err
		}
		authenticities = append(authenticities, ssh.PublicKeys(signer))
	}

	hostKeyCallback, err := opts.hostKeyCallback()
	if err != nil {
		return nil, err
	}

	return &ssh.ClientConfig{
		User:            opts.User,
		Auth:            authenticities,
		HostKeyCallback: hostKeyCallback,
		// The dial deadline bounds the handshake; this is a second guard.
		Timeout: opts.timeout(),
	}, nil
}

func (opts *sshOpts) timeout() time.Duration {
	return time.Duration(opts.Timeout * float64(time.Second))
}

// dial connects under one deadline covering the TCP connection, the
// handshake and the command: a host that accepts the connection and then
// stalls must not hang the plugin past -t.
func (opts *sshOpts) dial(config *ssh.ClientConfig) (*ssh.Client, error) {
	addr := net.JoinHostPort(opts.Hostname, strconv.Itoa(opts.Port))
	deadline := opts.deadline(time.Now())
	conn, err := (&net.Dialer{Deadline: deadline}).Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	if err := conn.SetDeadline(deadline); err != nil {
		conn.Close()
		return nil, err
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return ssh.NewClient(c, chans, reqs), nil
}

func (opts *sshOpts) run() *checkers.Checker {
	// prevent changing output of some commands
	os.Setenv("LANG", "C")
	os.Setenv("LC_ALL", "C")

	config, err := opts.makeClientConfig()
	if err != nil {
		return checkers.Unknown(err.Error())
	}

	client, err := opts.dial(config)
	if err != nil {
		return checkers.Critical(err.Error())
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return checkers.Critical(err.Error())
	}
	defer session.Close()

	// Compare with the local time at the middle of the exchange, so that the
	// SSH round trip does not count as clock difference.
	before := time.Now()
	output, err := session.Output("date +%s")
	if err != nil {
		return checkers.Unknown(err.Error())
	}
	startDate := before.Add(time.Since(before) / 2).Unix()
	remoteDate, diffTime, err := clockDiff(output, startDate)
	if err != nil {
		return checkers.Unknown(err.Error())
	}

	checkState := diffStatus(diffTime, opts.Warning, opts.Critical)

	message := fmt.Sprintf("Current date: %d (%s) - Remote date: %d (%s) - Diff time: %d", startDate, time.Unix(startDate, 0), remoteDate, time.Unix(remoteDate, 0), diffTime)
	return checkers.NewChecker(checkState, message)
}

// clockDiff parses the remote `date +%s` output and returns the remote date
// and its absolute difference with now.
func clockDiff(output []byte, now int64) (int64, int64, error) {
	raw := strings.TrimRight(string(output), "\r\n")
	remoteDate, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot read the remote date from %q: %w", raw, err)
	}
	return remoteDate, Abs(now - remoteDate), nil
}

// diffStatus maps a clock difference to a status: below warning is OK.
func diffStatus(diffTime, warning, critical int64) checkers.Status {
	switch {
	case diffTime >= critical:
		return checkers.CRITICAL
	case diffTime >= warning:
		return checkers.WARNING
	}
	return checkers.OK
}

// Abs returns the absolute value of x.
func Abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// deadline bounds the whole exchange from now; -t 0 means no deadline.
func (opts *sshOpts) deadline(now time.Time) time.Time {
	if opts.Timeout <= 0 {
		return time.Time{}
	}
	return now.Add(opts.timeout())
}

// exitCode turns a flag parsing failure into the plugin's exit code: 0 for
// --help, UNKNOWN for a bad command line (exit 1 would read as WARNING).
func exitCode(err error) int {
	if flags.WroteHelp(err) {
		return 0
	}
	return int(checkers.UNKNOWN)
}
