.PHONY: build
build:
	go build -o bin/tq -ldflags="-X 'main.Version=v0.1'"

.PHONY: test
test: build
	cd testdata
	../bin/tq < ../sql/queries.sql
	cd ..

.PHONY: clean
clean:
	rm bin/*
	rm *.out
	rm testdata/*.out