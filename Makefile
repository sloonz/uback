SHELL=/bin/bash

.PHONY: all
all: uback

uback: go.mod go.sum Makefile **/*.go *.go sources/scripts/*.sh
	go build -ldflags="-X github.com/sloonz/uback/cmd.commit=$$(git rev-parse --short HEAD) -X github.com/sloonz/uback/cmd.buildDate=$$(date -u +%Y-%m-%dT%H:%M:%SZ)"

.PHONY: test
test: uback
	go test ./...
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
