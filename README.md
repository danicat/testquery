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

```sh
% tq --help
Usage of tq:
  -dbfile string
    	database file name for use with --persist and --open (default "testquery.db")
  -open
    	open a database from a previous run
  -persist
    	persist database between runs
  -pkg string
    	directory of the package to test (default ".")
  -query string
    	runs a single query and returns the result

```
By default tq will launch in iterative mode unless you pass a `--query` flag:

```sh
% tq --persist --open --query "select * from code_coverage where file = 'div.go'"
+--------+-------------+-----------------------------------------------------------+---------+
| FILE   | LINE_NUMBER | CONTENT                                                   | COVERED |
+--------+-------------+-----------------------------------------------------------+---------+
| div.go |           1 | package testdata                                          |       0 |
| div.go |           2 |                                                           |       0 |
| div.go |           3 | import "errors"                                           |       0 |
| div.go |           4 |                                                           |       0 |
| div.go |           5 | var ErrDivideByZero = errors.New("cannot divide by zero") |       0 |
| div.go |           6 |                                                           |       0 |
| div.go |           7 | func divide(dividend, divisor int) (int, error) {         |       1 |
| div.go |           8 |     if divisor == 0 {                                     |       1 |
| div.go |           9 |         return 0, ErrDivideByZero                         |       1 |
| div.go |          10 |     }                                                     |       1 |
| div.go |          11 |                                                           |       0 |
| div.go |          12 |     return dividend / divisor, nil                        |       1 |
| div.go |          13 | }                                                         |       0 |
| div.go |          14 |                                                           |       0 |
+--------+-------------+-----------------------------------------------------------+---------+
```


To use it, compile the code with `make build` (or `go build` if you are being wild) and run `tq` from the command line. By default `tq` will run data collection on the current directory but you can pass a package to it by using the `--pkg` flag.

To run the examples (in `sql/queriesl.sql`), clone this project and run the following command:

```sh
$ bin/tq --pkg ./testdata/
```
