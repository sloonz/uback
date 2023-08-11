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
	bash <(curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh) -d -b .bin

.bin/goreleaser:
	bash <(curl -sfL https://install.goreleaser.com/github.com/goreleaser/goreleaser.sh) -d -b .bin

release: .bin/goreleaser
	./.bin/goreleaser release
