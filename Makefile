SHELL := /usr/bin/env sh

ROOT := $(subst \,/,$(CURDIR))
GO ?= go

ifeq ($(OS),Windows_NT)
EXEEXT := .exe
HOME_DIR ?= $(USERPROFILE)
else
EXEEXT :=
HOME_DIR ?= $(HOME)
endif

BINARY := mcp-proxy$(EXEEXT)
CMD_DIR := ./cmd/mcp-proxy
LOCAL_BIN ?= $(subst \,/,$(HOME_DIR))/.local/bin
LOCAL_BIN_WIN := $(subst /,\,$(LOCAL_BIN))

export GOCACHE := $(ROOT)/.gocache
export GOPATH := $(ROOT)/.gopath
export GOMODCACHE := $(ROOT)/.gopath/pkg/mod

.PHONY: bootstrap fix fmt lint test coverage build run install uninstall clean

bootstrap:
	$(GO) get -tool github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GO) get -tool github.com/onsi/ginkgo/v2/ginkgo@latest
	$(GO) mod tidy

fix:
	$(GO) fix ./...

fmt:
	$(GO) fmt ./...

lint:
	$(GO) tool golangci-lint run

test:
	$(GO) tool ginkgo -r -p --race --randomize-all --randomize-suites --fail-on-pending --keep-going

coverage:
	$(GO) tool ginkgo -r -p --cover --covermode=atomic --coverprofile=coverage.out --output-dir=.
	$(GO) tool cover -func=coverage.out

build:
	$(GO) build -buildvcs=false -o "$(BINARY)" $(CMD_DIR)

run:
	$(GO) run -buildvcs=false $(CMD_DIR)

install: build
ifeq ($(OS),Windows_NT)
	if not exist "$(LOCAL_BIN_WIN)" mkdir "$(LOCAL_BIN_WIN)"
	copy /Y "$(BINARY)" "$(LOCAL_BIN_WIN)\$(BINARY)" >NUL || (echo "install failed: $(LOCAL_BIN_WIN)\$(BINARY) is likely in use; stop running mcp-proxy processes and retry." && exit /B 1)
	@echo "installed $(BINARY) to $(LOCAL_BIN_WIN)\$(BINARY)"
else
	mkdir -p "$(LOCAL_BIN)"
	cp "$(BINARY)" "$(LOCAL_BIN)/$(BINARY)"
	@echo "installed $(BINARY) to $(LOCAL_BIN)/$(BINARY)"
endif

uninstall:
ifeq ($(OS),Windows_NT)
	if exist "$(LOCAL_BIN_WIN)\$(BINARY)" del /Q "$(LOCAL_BIN_WIN)\$(BINARY)"
	@echo "removed $(LOCAL_BIN_WIN)\$(BINARY)"
else
	$(RM) "$(LOCAL_BIN)/$(BINARY)"
	@echo "removed $(LOCAL_BIN)/$(BINARY)"
endif

clean:
	$(RM) "$(BINARY)"
