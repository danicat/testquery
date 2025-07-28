# TestQuery Improvement Report

## 1. Executive Summary

TestQuery (`tq`) is a promising tool with a solid foundation, but it currently lacks the robustness and polish of a professional-grade project. This report outlines several areas for improvement, focusing on functionality, user experience, code quality, and development practices. Since backward compatibility is not a concern, we have a unique opportunity to implement significant enhancements.

The most critical areas for improvement are:

*   **Error Handling:** The tool silently ignores errors from `go test`, which can lead to incorrect or incomplete data.
*   **Code Organization:** The monolithic `main` package structure hinders maintainability and scalability.
*   **Testing:** The project lacks a dedicated test suite for its own logic, which is essential for ensuring correctness.
*   **Performance:** The per-test coverage analysis is inefficient and will not scale to larger projects.

This report provides a roadmap for evolving `tq` into a more powerful, reliable, and user-friendly tool.

## 2. Functionality

### 2.1. Inefficient Coverage Analysis

The current implementation runs `go test` for each individual test to calculate test-specific coverage. This approach is extremely inefficient and will be prohibitively slow for projects with a large number of tests.

**Recommendation:**

Instead of re-running tests, parse the main `coverage.out` file more intelligently. It is possible to determine which lines of code are covered by which tests by analyzing the coverage profiles.

### 2.2. Fragile Function Name Retrieval

The `getFunctionName` function parses the source code on every call to determine the function at a given line. This is inefficient and can be unreliable, especially with complex code structures.

**Recommendation:**

Use the `go/analysis` framework or a similar static analysis approach to build a more robust and efficient mapping of code locations to function names.

### 2.3. Lack of Support for Build Tags

The tool does not account for Go build tags, which means it may fail to test or analyze code that is conditionally compiled.

**Recommendation:**

Add a mechanism to pass build tags to the underlying `go test` commands. This could be a command-line flag (e.g., `--tags`).

### 2.4. Limited Database Schema

The database schema is functional but could be extended to capture more detailed information, such as:

*   Test execution time (per test and per package).
*   Memory and CPU usage of tests.
*   A history of test runs to track changes over time.

**Recommendation:**

Expand the database schema to include these additional metrics. This will enable more powerful and insightful queries.

## 3. User Interface and User Experience

### 3.1. Ambiguous Command-Line Interface

The current flag-based CLI is ambiguous and not scalable. The interaction between the `--persist` and `--open` flags is confusing, as it implicitly changes the tool's mode of operation from data collection to data querying. This can lead to unexpected behavior.

**Recommendation:**

Redesign the CLI to use explicit subcommands, which makes the user's intent clear and provides a more scalable structure.

*   **`tq collect`**: Runs tests and collects data into a database.
    *   Example: `tq collect --pkg ./testdata --output testdata.db`
*   **`tq query`**: Runs a single query against a specified database.
    *   Example: `tq query --db testdata.db "SELECT * FROM failed_tests"`
*   **`tq shell`**: Launches the interactive SQL shell for a given database.
    *   Example: `tq shell --db testdata.db`

This approach eliminates the need for the confusing `--persist` and `--open` flags in favor of a more intuitive and explicit workflow.

### 3.2. Basic Interactive Prompt

The interactive prompt is a good start, but it lacks features that users would expect from a modern CLI, such as autocompletion for table and column names.

**Recommendation:**

Integrate a more advanced readline library or build a custom completer to provide a better interactive experience.

### 3.3. Inflexible Output Formats

The tool only outputs data in a fixed table format. This is not ideal for scripting or integration with other tools.

**Recommendation:**

Add support for other output formats, such as JSON and CSV, controlled by a command-line flag (e.g., `--output-format json`).

### 3.4. Configuration Management

For larger projects, it would be beneficial to have a configuration file to avoid passing the same flags on every invocation.

**Recommendation:**

Implement support for a configuration file (e.g., `.tq.yaml`) where users can define the package directory, database file, and other default settings.

## 4. Code Organization and Maintainability

### 4.1. Monolithic `main` Package

All the code resides in the `main` package, which makes it difficult to navigate, test, and maintain.

**Recommendation:**

Refactor the code into a more modular structure with distinct packages, for example:

*   `cmd/tq`: The main application entry point.
*   `internal/db`: Database-related logic.
*   `internal/runner`: Logic for running `go test` and collecting results.
*   `internal/parser`: Logic for parsing test and coverage output.
*   `internal/analysis`: More advanced analysis features.

### 4.2. Insufficient Error Handling

The tool ignores errors from `cmd.Output()` when running `go test`. This is a critical flaw that can lead to silent failures.

**Recommendation:**

Implement robust error handling for all external commands and library calls. Errors should be logged, and the program should exit with a non-zero status code when an error occurs.

## 5. Testing and Quality Assurance

### 5.1. No Internal Test Suite

The project lacks a test suite for its own logic. This makes it impossible to verify the correctness of the tool and prevent regressions.

**Recommendation:**

Create a comprehensive test suite that covers all aspects of the tool, including:

*   Unit tests for individual functions.
*   Integration tests that run `tq` against the `testdata` directory and verify the database content.
*   Tests for the command-line interface and flag parsing.

### 5.2. Basic CI/CD Pipeline

The existing GitHub Actions workflow only builds and tests the code. It does not include linting, static analysis, or release automation.

**Recommendation:**

Enhance the CI/CD pipeline to include:

*   A linting step using `golangci-lint`.
*   A static analysis step using `go vet`.
*   Automated releases with versioning and changelog generation.

## 6. Project and Development Practices

### 6.1. No Versioning Strategy

The project is not versioned, which makes it difficult to track changes and manage releases.

**Recommendation:**

Implement a versioning strategy using Git tags and set the version at build time using linker flags. For example:

```bash
go build -ldflags="-X main.Version=$(git describe --tags)"
```

### 6.2. Incomplete `LICENSE` File

The `LICENSE` file is empty. This creates legal ambiguity for potential users and contributors.

**Recommendation:**

Choose an appropriate open-source license (e.g., MIT, Apache 2.0) and add it to the `LICENSE` file.