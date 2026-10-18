# check-memory

Checks the used memory percentage.

## Synopsis

```
check-memory -w 95 -c 98
```

## Options

```
Usage:
  check-memory [OPTIONS]

Application Options:
  -w, --warning=  Sets warning value for Memory Usage. Default is 95% (default:
                  95)
  -c, --critical= Sets critical value for Memory Usage. Default is 98%
                  (default: 98)
```

## Statuses

- OK below `--warning`, WARNING from `--warning`, CRITICAL from `--critical`
  (percentages).
- UNKNOWN when a threshold is not a number or the memory cannot be read.

## Build

`make build` produces a static linux/amd64 binary named `check-memory`; `make test` runs
the tests. See the [repository README](../README.md).
