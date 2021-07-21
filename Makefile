SHELL   := /bin/bash
VERSION := v1.0.0
GOOS    := $(shell go env GOOS)
GOARCH  := $(shell go env GOARCH)

.PHONY: all
all: vet build

.PHONY: build
build:
	go build -ldflags "-X main.version=$(VERSION)" ./cmd/pgslap

.PHONY: vet
vet:
	go vet

.PHONY: package
package: clean vet build
ifeq ($(GOOS),windows)
	zip pgslap_$(VERSION)_$(GOOS)_$(GOARCH).zip pgslap.exe
	sha1sum pgslap_$(VERSION)_$(GOOS)_$(GOARCH).zip > pgslap_$(VERSION)_$(GOOS)_$(GOARCH).zip.sha1sum
else
	gzip pgslap -c > pgslap_$(VERSION)_$(GOOS)_$(GOARCH).gz
	sha1sum pgslap_$(VERSION)_$(GOOS)_$(GOARCH).gz > pgslap_$(VERSION)_$(GOOS)_$(GOARCH).gz.sha1sum
endif

.PHONY: clean
clean:
	rm -f pgslap pgslap.exe
