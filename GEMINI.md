# TestQuery Gemini Guide

This document provides instructions on how to understand, build, test, and use the TestQuery (`tq`) project.

## Project Purpose

TestQuery (`tq`) is a command-line tool that allows you to query Go test results using a SQL interface. It is designed to help developers understand and analyze tests in their projects, especially in large and mature codebases.

The tool works by running `go test` commands, collecting the output, and storing it in a SQLite database. You can then query this database to get insights into your tests, such as which tests are failing, what the code coverage is for a specific test, and more.

## Code Organization

The project is organized into several Go files in the root directory, with SQL queries and test data in separate directories.

*   `main.go`: The main entry point for the application. It handles command-line arguments, sets up the database, and starts the interactive prompt or executes a single query.
*   `data.go`: Contains the logic for creating the database schema and populating the tables with data from the test runs.
*   `all_tests.go`: This file contains the logic to run `go test -json` to collect the results of all tests in the specified package.
*   `all_coverage.go`: This file contains the logic to parse the `coverage.out` file to get the overall code coverage.
*   `test_coverage.go`: This file contains the logic to run each test individually to get the code coverage for each specific test.
*   `all_code.go`: This file contains the logic to read all the Go files in the package and store their content in the database.
*   `sql/`: This directory contains the database schema (`schema.sql`) and some example queries (`queries.sql`).
*   `testdata/`: This directory contains a sample Go project that can be used to test `tq`.

## How to Build

To build the `tq` executable, you can use the standard `go build` command.

```bash
go build
```

This will create a `tq` (or `tq.exe` on Windows) executable in the root directory of the project.

## How to Test

To test the `tq` tool itself, you can use the standard `go test` command.

```bash
go test ./...
```

To run `tq` on the provided `testdata` sample, you can use the following command:

```bash
./tq --pkg ./testdata/
```

## How to Use

You can run `tq` in two modes: interactive mode or single-query mode.

### Interactive Mode

To start `tq` in interactive mode, simply run the executable without the `--query` flag.

```bash
./tq
```

This will start an interactive prompt where you can type SQL queries.

```
> SELECT * FROM all_tests;
```

### Single-Query Mode

To run a single query and exit, use the `--query` flag.

```bash
./tq --query "SELECT * FROM all_tests WHERE action = 'fail';"
```

### Command-Line Flags

*   `--pkg <directory>`: Specifies the directory of the package to test. Defaults to the current directory (`.`).
*   `--persist`: Persists the database to a file between runs.
*   `--dbfile <filename>`: Specifies the name of the database file to use with `--persist` and `--open`. Defaults to `testquery.db`.
*   `--open`: Opens a database from a previous run.
*   `--query <query>`: Runs a single query and returns the result.
*   `--version`: Shows the version information.
