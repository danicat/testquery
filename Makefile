# Makefile for testquery

VERSION := 0.1.0

.PHONY: build
build:
	@mkdir -p bin
	go build -ldflags="-X 'github.com/danicat/testquery/cmd.Version=$(VERSION)'" -o bin/tq .

.PHONY: setup
setup:
	go install golang.org/x/tools/gopls@v0.23.0
	go install github.com/danicat/godoctor@latest
