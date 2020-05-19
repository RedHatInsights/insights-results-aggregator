---
layout: page
nav_order: 10
---
# Testing
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

tl;dr: `make before_commit` will run most of the checks by magic

The following tests can be run to test your code in `insights-results-aggregator`.
Detailed information about each type of test is included in the corresponding subsection:

1. Unit tests: checks behavior of all units in source code (methods, functions)
1. REST API Tests: test the real REST API of locally deployed application with database initialized
with test data only
1. Integration tests: the integration tests for `insights-results-aggregator` service
1. Metrics tests: test whether Prometheus metrics are exposed as expected

## Unit tests

Set of unit tests checks all units of source code. Additionally the code coverage is computed and
displayed. Code coverage is stored in a file `coverage.out` and can be checked by a script named
`check_coverage.sh`.

To run unit tests use the following command:

`make test`

If you have postgres running on port from `./config-devel.toml` file it will also run tests against
it

## All integration tests

`make integration_tests`

### Only REST API tests

Set of tests to check REST API of locally deployed application with database initialized with test
data only.

To run REST API tests use the following command:

`make rest_api_tests`

By default all logs from the application aren't shown, if you want to see them, run:

`./test.sh rest_api --verbose`

### Coverage reports

To make a coverage report you need to start `./make-coverage.sh` tool with one of these arguments:

1. `unit-sqlite` unit tests with sqlite in memory database
1. `unit-posgres` unit tests with postgres database(don't forget to start `docker-compose up` with the DB)
1. `rest` REST API tests from `test.sh` file
1. `integration` Any external tests, for example from iqe-ccx-plugin.
Only this option requires you to run tests manually and stop the script by `Ctrl+C` when they are done

For example:

`./make-coverage.sh unit-sqlite` will generate a report file `coverage.out`
which you can investigate by either of those commands:

- `go tool cover -func=coverage.out`
- `go tool cover -html=coverage.out` if your system supports it, this command will open a browser with a nice colored report

