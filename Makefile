.PHONY: build
build:
	go build -o bin/tq -ldflags="-X 'main.Version=v0.1'"

.PHONY: unit-test
unit-test:
	go test ./... -coverprofile=coverage.out -covermode=atomic

.PHONY: integration-test
integration-test: build
	cd testdata && ../bin/tq < ../sql/queries.sql && cd ..

.PHONY: test
test: unit-test integration-test

.PHONY: clean
clean:
	rm -f bin/tq
	rm -f *.out
	rm -f testdata/*.out