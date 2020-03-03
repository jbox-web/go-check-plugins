# check-memory

## Description
Check system memory


## Synopsis
```
check-memory
```

## Installation

First, build this program.

```
go get github.com/jbox-web/go-check-plugins/check-memory
cd $(go env GOPATH)/src/github.com/jbox-web/go-check-plugins/check-memory
make
```

Next, you can execute this program :-)

```
check-memory
```

## Usage

### Options

```
  -w, --warning=  Sets warning value for Memory Usage. Default is 95% (default: 95)
  -c, --critical= Sets critical value for Memory Usage. Default is 98% (default: 98)
```


## For more information

Please execute `check-memory -h` and you can get command line options.
