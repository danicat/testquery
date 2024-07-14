# TestQuery

Test Query (tq) is a command line tool to query Go test results with a SQL interface. The idea of using SQL was inspired by a similar tool called OSQuery, which does the same for operating system metrics.

## Demo

https://github.com/user-attachments/assets/b6ed5637-392c-4686-9405-fd174e559582

## History

During Gophercon 2024, specially after my talk on mutation testing, many people came to me to talk about their challenges with testing. One particular thought that stuck with me was that in older codebases it can become hard to keep track of the need for each individual test, and we can potentually end up with dozens - maybe even hundreds - of tests that are obsolete.

This tool was designed to make extracting information from tests easier for the average developer (as long as you know SQL of course - but everyone should learn SQL anyway ^^).

It is currently under development so it doesn't support a lot of information yet, but it is already possible to query basic information about tests, including:

- What tests are passing or not (all_tests, passed_tests, failed_tests)
- What is the overall coverage (all_coverage)
- What is the coverage provided by an individual test (test_coverage)

## Usage

Disclaimer: this tool is currently a prototype that has been through no more than 48 hours of active development, so use it at your own risk. Ironically, I haven't written any tests for it `:)`

To use it, compile the code with `make build` (or `go build` if you are being wild) and run `tq` from the command line. By default `tq` will run data collection on the current directory but you can pass a package to it by using the `--pkg` flag.

To run the examples (in `sql/queriesl.sql`), clone this project and run the following command:

```sh
$ bin/tq --pkg ./testdata/
```
