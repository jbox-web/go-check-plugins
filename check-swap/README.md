# check-swap

Checks the used swap percentage.

## Synopsis

```
check-swap -w 95 -c 98
```

## Options

```
Usage:
  check-swap [OPTIONS]

Application Options:
  -w, --warning=  Sets warning value for Swap Usage. Default is 95% (default:
                  95)
  -c, --critical= Sets critical value for Swap Usage. Default is 98% (default:
                  98)
```

## Statuses

- OK below `--warning`, WARNING from `--warning`, CRITICAL from `--critical`
  (percentages).
- UNKNOWN when a threshold is not a number or the swap cannot be read.

## Build

`make build` produces a static linux/amd64 binary named `check-swap`; `make test` runs
the tests. See the [repository README](../README.md).
