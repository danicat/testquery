.PHONY: build
build:
	go build -o bin/tq

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