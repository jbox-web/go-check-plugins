# go-check-plugins

[![CI](https://github.com/jbox-web/go-check-plugins/actions/workflows/ci.yml/badge.svg)](https://github.com/jbox-web/go-check-plugins/actions/workflows/ci.yml)

Monitoring check plugins written in Go, for Icinga / Nagios. Each plugin prints
one status line and exits with the standard codes: 0 OK, 1 WARNING, 2 CRITICAL,
3 UNKNOWN.

| Plugin | Checks |
|---|---|
| [`check-diff-time`](check-diff-time/README.md) | clock difference with a remote host, over SSH |
| [`check-file-age`](check-file-age/README.md) | age and size of a file |
| [`check-memory`](check-memory/README.md) | used memory percentage |
| [`check-mountpoints`](check-mountpoints/README.md) | NFS mountpoints mounted, and writable when read-write |
| [`check-mysql`](check-mysql/README.md) | MySQL / MariaDB replication, connections, uptime, read-only flag |
| [`check-swap`](check-swap/README.md) | used swap percentage |

## Layout

Each plugin is its own Go module (`check-*/go.mod`): a `main.go` that calls
`lib.Do()`, the logic and its tests under `lib/`, a `Makefile` and a README with
the plugin's options and statuses. Statuses and output go through
[`mackerelio/checkers`](https://github.com/mackerelio/checkers); system data
through [`gopsutil`](https://github.com/shirou/gopsutil) v4.

## Build

The plugins target Linux amd64 hosts. `make build` produces static binaries
(`CGO_ENABLED=0`, `-trimpath`, stripped) for that platform, whatever the build
machine:

```sh
make build          # every plugin, binary in check-*/check-*
make test           # every plugin's tests
make clean
make -C check-swap build
```

Go 1.27 or later is needed (`go` directive of each `go.mod`). Override `GOOS` /
`GOARCH` to build for another platform, e.g. `make build GOARCH=arm64`.

## Tests and CI

Tests live next to the code (`lib/*_test.go`) and need nothing but Go: the
system and network parts (gopsutil, SSH, MySQL) stay out of them, the logic that
turns data into a status is extracted and tested on its own. `check-mysql` also
keeps Docker-based end-to-end scripts (`test.sh`, `test_57/`, `test_8/`), not run
in CI.

[GitHub Actions](.github/workflows/ci.yml) checks every plugin on each push to
`master` and each pull request: `gofmt`, `go vet`, `staticcheck`, `go test`, and
the linux/amd64 build.
