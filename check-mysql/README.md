# check-mysql

Checks a MySQL or MariaDB server through one of four subcommands:
`replication`, `connection`, `uptime` and `readonly`. Each takes the connection
options below plus its own thresholds (`check-mysql <subcommand> --help`).

## Synopsis

```
check-mysql replication -S /run/mysqld/mysqld.sock -u monitor -w 200 -c 250
```

## Options

Those of `replication`; the connection options (from `-H` to `--tls-skip-verify`)
are shared by every subcommand.

```
Usage:
  check-mysql replication [OPTIONS]

Application Options:
  -H, --host=            Hostname (default: localhost)
  -p, --port=            Port (default: 3306)
  -S, --socket=          Path to unix socket
  -u, --user=            Username (default: root)
  -P, --password=        Password [$MYSQL_PASSWORD]
      --tls              Enables TLS connection
      --tls-root-cert=   The root certificate used for TLS certificate
                         verification
      --tls-skip-verify  Disable TLS certificate verification
  -c, --critical=        critical if the seconds behind master is over
                         (default: 250)
  -w, --warning=         warning if the seconds behind master is over (default:
                         200)
```

## Statuses

- `replication`: CRITICAL when the IO or SQL thread is stopped, otherwise by
  seconds behind the source; OK when the server is not a replica.
- `connection`: by number of connected threads.
- `uptime`: CRITICAL or WARNING when the server started less than the given
  seconds ago.
- `readonly`: CRITICAL when `read_only` differs from the expected `ON` or `OFF`.
- UNKNOWN when the server cannot be reached or queried.

## Build

`make build` produces a static linux/amd64 binary named `check-mysql`; `make test` runs
the tests. See the [repository README](../README.md).
