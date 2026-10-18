NAME := check-memory

# Icinga runs the plugins on Linux amd64 hosts: build a static binary for them
# (no cgo, so no dependency on the host's libc), whatever the build machine.
GOOS ?= linux
GOARCH ?= amd64

.PHONY: build test clean

build:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -trimpath -ldflags '-s -w' -o $(NAME) .

test:
	go test ./...

clean:
	rm -f $(NAME)
