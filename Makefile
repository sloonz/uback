SHELL=/bin/bash

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
	chown "$(USER)" "$(BTRFS_ROOT)"

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
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b .bin

.bin/goreleaser:
	GOBIN="$$(pwd)/.bin" go install github.com/goreleaser/goreleaser@latest

release: .bin/goreleaser
	./.bin/goreleaser release
