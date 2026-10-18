# check-file-age

Checks the age and the size of a file, typically one a scheduled job rewrites.

## Synopsis

```
check-file-age -f /var/backups/last-run -w 240 -c 600 -C 1
```

## Options

```
Usage:
  check-file-age [OPTIONS]

Application Options:
  -f, --file=           monitor file name
  -w, --warning-age=    warning if more old than (default: 240)
  -W, --warning-size=   warning if file size less than
  -c, --critical-age=   critical if more old than (default: 600)
  -C, --critical-size=  critical if file size less than (default: 0)
  -i, --ignore-missing  skip alert if file doesn't exist
```

## Statuses

- WARNING when the file is older than `--warning-age` seconds or smaller than
  `--warning-size` bytes; CRITICAL likewise with the critical options. A size
  threshold of 0 disables it.
- UNKNOWN when the file cannot be read, OK instead with `--ignore-missing`.

## Build

`make build` produces a static linux/amd64 binary named `check-file-age`; `make test` runs
the tests. See the [repository README](../README.md).
