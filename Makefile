SHELL=/bin/bash

export BTRFS_ROOT ?= /tmp/uback-btrfs-tests
export ZFS_ROOT ?= /tmp/uback-zfs-tests
export ZFS_POOL ?= ubackpool

ifdef SUDO_USER
	TESTS_USER ?= $(SUDO_USER)
else
	TESTS_USER ?= $(USER)
endif

.PHONY: all
all: uback

uback: go.mod go.sum Makefile **/*.go *.go sources/scripts/*.sh
	go build -ldflags="-X github.com/sloonz/uback/cmd.commit=$$(git rev-parse --short HEAD) -X github.com/sloonz/uback/cmd.buildDate=$$(date -u +%Y-%m-%dT%H:%M:%SZ)"

.PHONY: tests
tests: unit-tests integration-tests

.PHONY: unit-tests
unit-tests:
	go test ./...

.PHONY: setup-btrfs-root
setup-btrfs-root:
	tmp=$$(mktemp) || exit 1; \
	truncate -s 128MiB "$$tmp" || (unlink "$$tmp"; exit 1); \
	dev="$$(losetup --show -f "$$tmp")" || (unlink "$$tmp"; exit 1); \
	unlink "$$tmp" || (losetup -d "$$dev"; exit 1); \
	mkfs.btrfs "$$dev" || (losetup -d "$$dev"; exit 1); \
	mkdir "$(BTRFS_ROOT)" 2>/dev/null; \
	mount "$$dev" "$(BTRFS_ROOT)" || (losetup -d "$$dev"; exit 1); \
	chown "$(TESTS_USER)" "$(BTRFS_ROOT)"

.PHONY: setup-zfs-root
setup-zfs-root:
	tmp=$$(mktemp) || exit 1; \
	truncate -s 128MiB "$$tmp" || (unlink "$$tmp"; exit 1); \
	dev="$$(losetup --show -f "$$tmp")" || (unlink "$$tmp"; exit 1); \
	unlink "$$tmp" || (losetup -d "$$dev"; exit 1); \
	zpool create -f -o cachefile=none -m "$(ZFS_ROOT)" "$(ZFS_POOL)" "$$dev" || (losetup -d "$$dev"; exit 1); \
	zfs allow "$(TESTS_USER)" create,destroy,mount,receive,send,snapshot,bookmark,send:raw,hold,release "$(ZFS_POOL)" && \
	chown "$(TESTS_USER)" "$(ZFS_ROOT)"

.PHONY: integration-tests
integration-tests: uback
	python -m unittest tests/*_tests.py

.PHONY: lint
lint: .bin/golangci-lint
	./.bin/golangci-lint run ./...

.PHONY: clean
clean:
	rm -f uback ./.bin/golangci-lint

.bin/golangci-lint:
	GOBIN="$$(pwd)/.bin" go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

.bin/goreleaser:
	GOBIN="$$(pwd)/.bin" go install github.com/goreleaser/goreleaser@latest

release: .bin/goreleaser
	./.bin/goreleaser release
