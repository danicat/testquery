# Makefile for testquery

VERSION := 0.2.0
UNIT_TEST_PACKAGES := ./internal/...

.PHONY: all
all: help

.PHONY: help
help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  help                  Show this help message."
	@echo "  build                 Compile the tq binary into the bin/ directory."
	@echo "  test                  Run the unit tests for the project."
	@echo "  unit-test             Run the fast, isolated unit tests."
	@echo "  integration-test      Run the integration test suite."
	@echo "  test-cover            Run all tests and produce an aggregated coverage report."
	@echo "  setup                 Install the necessary Go tools for the project."
	@echo "  clean                 Remove all build and test artifacts."

.PHONY: build
build:
	@mkdir -p bin
	go build -ldflags="-X 'github.com/danicat/testquery/cmd.Version=$(VERSION)'" -o bin/tq .

.PHONY: setup
setup:
	go install golang.org/x/tools/gopls@v0.23.0
	go install github.com/danicat/godoctor@latest

.PHONY: test
test: unit-test

.PHONY: unit-test
unit-test:
	go test -coverprofile=unit.cover -covermode=count $(UNIT_TEST_PACKAGES)

.PHONY: integration-test
integration-test:
	@mkdir -p bin
	@rm -f testquery.db covmeta.*
	go build -cover -o bin/tq.cover .
	GOCOVERDIR=. ./bin/tq.cover query --pkg ./testdata/ "SELECT 1" > /dev/null 2>&1
	GOCOVERDIR=. ./bin/tq.cover query --pkg . "SELECT 1" > /dev/null 2>&1
	GOCOVERDIR=. ./bin/tq.cover query --pkg ./... "SELECT 1" > /dev/null 2>&1
	GOCOVERDIR=. ./bin/tq.cover query --pkg ./testdata/ --force "SELECT 1" > /dev/null 2>&1
	go tool covdata textfmt -i=. -o=integration.cover

.PHONY: test-cover
test-cover: unit-test integration-test
	@echo "mode: count" > coverage.out
	@tail -q -n +2 unit.cover integration.cover >> coverage.out
	@go tool cover -func=coverage.out

.PHONY: clean
clean:
	@rm -rf bin coverage *.cover *.out covmeta.* covcounters.*
