---
layout: page
nav_order: 12
---
# CI

[Travis CI](https://travis-ci.com/) is configured for this repository. Several tests and checks are
started for all pull requests:

* Unit tests that use the standard tool `go test`.
* `go fmt` tool to check code formatting. That tool is run with `-s` flag to perform
[following transformations](https://golang.org/cmd/gofmt/#hdr-The_simplify_command)
* `go vet` to report likely mistakes in source code, for example suspicious constructs, such as
`Printf` calls whose arguments do not align with the format string.
* `golint` as a linter for all Go sources stored in this repository
* `gocyclo` to report all functions and methods with too high cyclomatic complexity. The cyclomatic
complexity of a function is calculated according to the following rules: 1 is the base complexity of
a function +1 for each `if`, `for`, `case`, `&&` or `||` Go Report Card warns on functions with
cyclomatic complexity > 9
* `goconst` to find repeated strings that could be replaced by a constant
* `gosec` to inspect source code for security problems by scanning the Go AST
* `ineffassign` to detect and print all ineffectual assignments in Go code
* `errcheck` for checking for all unchecked errors in go programs
* `shellcheck` to perform static analysis for all shell scripts used in this repository
* `abcgo` to measure ABC metrics for Go source code and check if the metrics does not exceed
  specified threshold
* `golangci-lint` as Go linters aggregator with lot of linters enabled: https://golangci-lint.run/usage/linters/
* BDD tests that checks the overall Insights Results Aggregator behaviour.

Please note that all checks mentioned above have to pass for the change to be merged into master
branch.

History of checks performed by CI is available at
[RedHatInsights / insights-results-aggregator](https://travis-ci.org/RedHatInsights/insights-results-aggregator).
