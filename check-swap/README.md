# check-swap

## Description
Check system swap


## Synopsis
```
check-swap
```

## Installation

First, build this program.

```
go get github.com/jbox-web/go-check-plugins/check-swap
cd $(go env GOPATH)/src/github.com/jbox-web/go-check-plugins/check-swap
make
```

Next, you can execute this program :-)

```
check-swap
```

## Usage

### Options

```
  -w, --warning=  Sets warning value for Swap Usage. Default is 95% (default: 95)
  -c, --critical= Sets critical value for Swap Usage. Default is 98% (default: 98)
```


## For more information

Please execute `check-swap -h` and you can get command line options.
