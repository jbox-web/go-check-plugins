# check-mountpoints

Checks NFS mountpoints (v3 and v4) listed in `/etc/fstab`: that each one is
mounted, and with `--write-test` that a read-write mount accepts a file being
created, written and removed.

## Synopsis

```
check-mountpoints -w -m /data/shared -m /data/backups
```

## Options

```
Usage:
  check-mountpoints [OPTIONS]

Application Options:
  -w, --write-test  run write test
  -m=               list of mountpoints to check. Can be repeated multiple times
  -t, --timeout=    Seconds before a mountpoint that does not answer is
                    reported CRITICAL (default: 10)
```

## Statuses

Each mountpoint gets its own line in the output; the global status is the worst
one, CRITICAL before UNKNOWN before OK.

- CRITICAL when a mountpoint is not mounted, does not exist, or fails a write
  test step.
- CRITICAL as well when a mountpoint does not answer within `--timeout` (a hard
  NFS mount whose server is gone), or cannot be stat-ed.
- UNKNOWN when a mountpoint is not an NFS entry of `/etc/fstab`, or when no
  `-m` is given.
- OK when mounted read-only, read-write untested, or read-write and writable.

## Build

`make build` produces a static linux/amd64 binary named `check-mountpoints`; `make test` runs
the tests. See the [repository README](../README.md).
