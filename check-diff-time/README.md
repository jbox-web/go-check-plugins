# check-diff-time

## Description
Check system diff-time


## Synopsis
```
check-diff-time
```

## Installation

First, build this program.

```
go get github.com/jbox-web/go-check-plugins/check-diff-time
cd $(go env GOPATH)/src/github.com/jbox-web/go-check-plugins/check-diff-time
make
```

Next, you can execute this program :-)

```
check-diff-time
```

## Usage

### Options

```
  -H, --hostname=   Host name or IP Address (default: localhost)
  -P, --port=       Port number (default: 22)
  -t, --timeout=    Seconds before connection times out (default: 30)
  -w, --warning=    Response time to result in warning status (seconds) (default: 5)
  -c, --critical=   Response time to result in critical status (seconds) (default: 10)
  -u, --user=       Login user name [$USER]
  -p, --password=   Login password [$LOGIN_PASSWORD]
  -i, --identity=   Identity file (ssh private key)
      --passphrase= Identity passphrase [$CHECK_SSH_IDENTITY_PASSPHRASE]
```


## For more information

Please execute `check-diff-time -h` and you can get command line options.
