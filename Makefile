.PHONY: default clean build fmt lint vet cyclo ineffassign shellcheck errcheck goconst gosec abcgo json-check openapi-check style run test test-postgres cover integration_tests rest_api_tests rules_content sqlite_db license before_commit help

SOURCES:=$(shell find . -name '*.go')

default: build

clean: ## Run go clean
	@go clean

build: ## Run go build
	./build.sh

fmt: ## Run go fmt -w for all sources
	@echo "Running go formatting"
	./gofmt.sh

lint: ## Run golint
	@echo "Running go lint"
	./golint.sh

vet: ## Run go vet. Report likely mistakes in source code
	@echo "Running go vet"
	./govet.sh

cyclo: ## Run gocyclo
	@echo "Running gocyclo"
	./gocyclo.sh

ineffassign: ## Run ineffassign checker
	@echo "Running ineffassign checker"
	./ineffassign.sh

shellcheck: ## Run shellcheck
	./shellcheck.sh

errcheck: ## Run errcheck
	@echo "Running errcheck"
	./goerrcheck.sh

goconst: ## Run goconst checker
	@echo "Running goconst checker"
	./goconst.sh

gosec: ## Run gosec checker
	@echo "Running gosec checker"
	./gosec.sh

abcgo: ## Run ABC metrics checker
	@echo "Run ABC metrics checker"
	./abcgo.sh

json-check: ## Check all JSONs for basic syntax
	@echo "Run JSON checker"
	python3 utils/json_check.py

openapi-check:
	./check_openapi.sh

style: fmt vet lint cyclo shellcheck errcheck goconst gosec ineffassign abcgo json-check ## Run all the formatting related commands (fmt, vet, lint, cyclo) + check shell scripts

run: clean build ## Build the project and executes the binary
	./insights-results-aggregator

automigrate: clean build
	./insights-results-aggregator migrate latest

test: clean build ## Run the unit tests
	./unit-tests.sh

test-postgres: clean build
	./unit-tests.sh postgres

cover: test
	@go tool cover -html=coverage.out

integration_tests: ## Run all integration tests
	@echo "Running all integration tests"
	@./test.sh

rest_api_tests: ## Run REST API tests
	@echo "Running REST API tests"
	@./test.sh rest_api

rules_content: ## Update tests/content/ok directory with latest rules content
	./update_rules_content.sh

sqlite_db:
	mv aggregator.db aggragator.db.backup
	local_storage/create_database_sqlite.sh

license:
	GO111MODULE=off go get -u github.com/google/addlicense && \
		addlicense -c "Red Hat, Inc" -l "apache" -v ./

before_commit: style
	./unit-tests.sh
	./unit-tests.sh postgres
	make integration_tests
	make openapi-check
	make license
	./check_coverage.sh

help: ## Show this help screen
	@echo 'Usage: make <OPTIONS> ... <TARGETS>'
	@echo ''
	@echo 'Available targets are:'
	@echo ''
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ''
