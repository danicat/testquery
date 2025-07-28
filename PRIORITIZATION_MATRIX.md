# Prioritization Matrix and Backlog

This document contains a prioritized list of improvements for the TestQuery (`tq`) tool, based on a matrix of Technical Certainty vs. Business Value.

---

## Quadrant 1: Top Priority (Do Now)

*These items represent the highest-impact work that can be done with the most confidence. They address critical bugs, major usability issues, and foundational code quality problems.*

| Priority | Improvement | Business Value | Technical Certainty | Rationale |
| :--- | :--- | :--- | :--- | :--- |
| **1** | **Insufficient Error Handling** | **High** | **High** | Silent failures are critical bugs that undermine the tool's reliability. Fixing this is essential for user trust. The implementation is straightforward. |
| **2** | **Ambiguous CLI** | **High** | **High** | The current CLI is confusing. Moving to a subcommand structure (`collect`, `query`, `shell`) makes the tool vastly more intuitive and scalable. This is a standard task with libraries like `cobra`. |
| **3** | **No Internal Test Suite** | **High** | **High** | We cannot safely refactor or add features without a test suite to prevent regressions. This is a foundational requirement for a healthy project. |
| **4** | **Monolithic `main` Package** | **Medium** | **High** | Refactoring into a modular structure is crucial for long-term maintainability and will make implementing all other features easier and cleaner. |
| **5** | **Inflexible Output Formats** | **Medium** | **High** | Adding JSON/CSV output makes `tq` usable in scripts and CI/CD pipelines, significantly expanding its utility beyond interactive use. |
| **6** | **No Versioning Strategy** | **Medium** | **High** | A proper versioning strategy is a hallmark of a professional project, enabling clear communication about releases and changes. Implementation via linker flags is simple. |
| **7** | **Incomplete `LICENSE` File** | **Medium** | **High** | An empty `LICENSE` file creates legal ambiguity. Adding a standard open-source license is a simple but critical step for community adoption. |
| **8** | **Basic CI/CD Pipeline** | **Medium** | **High** | Adding linting (`golangci-lint`) and static analysis (`go vet`) to the CI pipeline will automatically enforce code quality and catch common bugs early. |

---

## Quadrant 2: Needs Exploration

*These items are highly valuable but have technical uncertainties. A research "spike" is required for each to understand the complexity and define a clear implementation path before committing to the work.*

| Item | Improvement | Business Value | Technical Certainty | Action Required |
| :--- | :--- | :--- | :--- | :--- |
| **A** | **Inefficient Coverage Analysis** | **High** | **Medium** | The current per-test coverage analysis is not scalable. **Explore** how to intelligently parse a single, comprehensive `coverage.out` file to determine per-test coverage without re-running tests. |
| **B** | **Fragile Function Name Retrieval** | **Medium** | **Low** | The current AST parsing is inefficient. **Explore** using the `go/analysis` framework to create a more robust and performant way to map code locations to function names. |
| **C** | **Basic Interactive Prompt** | **Low** | **Medium** | Autocompletion would improve the user experience. **Explore** advanced readline libraries and the effort required to implement SQL-aware autocompletion for table and column names. |

---

## Quadrant 3: Optional (Do Later / Backlog)

*These items are "nice-to-haves" that provide value but are not critical. They are easy to implement and can be picked up when higher-priority work is complete.*

| Item | Improvement | Business Value | Technical Certainty | Rationale |
| :--- | :--- | :--- | :--- | :--- |
| **D** | **Lack of Support for Build Tags** | **Medium** | **High** | This is a feature limitation for some projects. It's a straightforward addition of a flag that is passed to the `go test` command. |
| **E** | **Configuration Management** | **Low** | **High** | A `.tq.yaml` file would be a convenience for power users but is not essential for core functionality. Libraries like Viper make this easy to add later. |

---

## Quadrant 4: Deprioritized (Ignore for now)

*These items have low business value and/or significant technical uncertainty. They should not be worked on at this time.*

| Item | Improvement | Business Value | Technical Certainty | Rationale |
| :--- | :--- | :--- | :--- | :--- |
| **F** | **Limited Database Schema** | **Low** | **High** | Expanding the schema for more metrics is a feature enhancement, but the core value is already provided. This can be revisited if users request it. |
