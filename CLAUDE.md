# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repository is

Six independent Icinga / Nagios check plugins, one Go module each
(`check-*/go.mod`), no root module and no `go.work`. Every command runs **inside
a plugin directory**, or through the root `Makefile`, which loops over
`check-*`. The repository is public: keep host names, IPs and anything specific
to the infrastructure that runs the plugins out of code, tests and docs.

## Commands

```sh
make test                         # go test in every plugin
make build                        # static linux/amd64 binary in check-*/check-*
make -C check-memory test
cd check-memory && go vet ./... && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./...
gofmt -l .                        # must print nothing
```

Go 1.27 or later (`go` directive). The CI (`.github/workflows/ci.yml`) runs, per
plugin: `gofmt -l`, `go vet`, `staticcheck` (pinned), `go test`, `make build`.
Keep a change green on all five locally before pushing.

## Plugin structure

```
check-<x>/
├── main.go          package main, only calls check<x>.Do()
├── lib/<x>.go       package check<x>: flags, data gathering, status logic
├── lib/<x>_test.go  tests of the pure functions
├── Makefile         build (CGO_ENABLED=0 GOOS=linux GOARCH=amd64), test, clean
└── README.md        options (copied from --help), statuses
```

`Do()` parses flags with `jessevdk/go-flags` and ends with `checkers.Exit()`
from `mackerelio/checkers`, whose `Status` values map to the exit codes
(OK 0, WARNING 1, CRITICAL 2, UNKNOWN 3). **`checkers.OK` is the zero value of
`Status`**: a status variable left unassigned silently reads as OK. Always
return an explicit status.
A command-line error exits UNKNOWN through `exitCode()` (3), `--help` exits 0:
never `os.Exit(1)`, which Icinga reads as WARNING.

## Conventions

- **Logic out of I/O.** Gathering (gopsutil, SSH, SQL, the filesystem) stays in
  `run()`; turning data into a status is a separate pure function
  (`checkStatus`, `diffStatus`, `clockDiff`, `getGlobalStatus`, `checkMount`,
  `secondsToHuman`…), which is what the tests cover. A new check follows the same
  split.
- **Thresholds: critical before warning.** A value above the critical threshold
  is above the warning one too; testing warning first never yields CRITICAL —
  the bug check-memory and check-swap had.
- **Never drop an error.** A threshold that does not parse, an unreadable remote
  output, a CA file without certificate: each returns UNKNOWN with the reason,
  never a zero value that reads as a valid measure.
- **Multi-item checks** (check-mountpoints) report one line per item and the
  worst status: CRITICAL, then UNKNOWN, then OK.
- **SSH** (check-diff-time): `golang.org/x/crypto/ssh` must stay recent enough to
  sign RSA keys with `rsa-sha2-*`: OpenSSH servers that no longer accept the
  SHA-1 `ssh-rsa` signature reject older versions. The deadline covers the whole
  exchange, not only the TCP connection. Host keys are only verified with
  `--known-hosts`.
- Tests write expected values as literals from the plugin's intended behavior;
  a test that passes on the first run is checked once against a deliberately
  broken implementation before being trusted.
- After changing flags, regenerate the Options block of the plugin's README
  from `--help`.
