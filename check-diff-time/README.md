# check-diff-time

Compares the clock of a remote host with the local one, over SSH: it runs
`date +%s` on the remote host and reports the absolute difference in seconds.

## Synopsis

```
check-diff-time -H 10.0.0.2 -u nagios -i ~/.ssh/id_rsa -w 2 -c 5
```

## Options

```
Usage:
  check-diff-time [OPTIONS]

Application Options:
  -H, --hostname=    Host name or IP Address (default: localhost)
  -P, --port=        Port number (default: 22)
  -t, --timeout=     Seconds before the whole SSH exchange (connection,
                     handshake, command) times out (default: 30)
  -w, --warning=     Clock difference to result in warning status (seconds)
                     (default: 5)
  -c, --critical=    Clock difference to result in critical status (seconds)
                     (default: 10)
  -u, --user=        Login user name [$USER]
  -p, --password=    Login password [$LOGIN_PASSWORD]
  -i, --identity=    Identity file (ssh private key)
      --passphrase=  Identity passphrase [$CHECK_SSH_IDENTITY_PASSPHRASE]
      --known-hosts= known_hosts file to verify the host key against (not
                     verified when empty); it must hold the key type the server
                     negotiates first
```

## Statuses

- OK below `--warning`, WARNING from `--warning`, CRITICAL from `--critical`.
- CRITICAL when the connection or the SSH session cannot be opened.
- UNKNOWN when the key cannot be read, the command fails or its output is not a
  timestamp.

The host key is not verified unless `--known-hosts` is given.

## Build

`make build` produces a static linux/amd64 binary named `check-diff-time`; `make test` runs
the tests. See the [repository README](../README.md).
