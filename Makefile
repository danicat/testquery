# Makefile for testquery

.PHONY: build
build:
	@mkdir -p bin
	go build -o bin/tq .
