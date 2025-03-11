SHELL := /bin/bash

.PHONY: default clean build fmt lint vet cyclo ineffassign shellcheck errcheck goconst gosec abcgo json-check openapi-check style run test cover integration_tests rest_api_tests license before_commit help install_addlicense

SOURCES:=$(shell find . -name '*.go')
BINARY:=insights-results-aggregator
DOCFILES:=$(addprefix docs/packages/, $(addsuffix .html, $(basename ${SOURCES})))

default: build

clean: ## Run go clean
	@go clean

build: ${BINARY} ## Build binary containing service executable

build-cover:	${SOURCES}  ## Build binary with code coverage detection support
	./build.sh -cover

${BINARY}: ${SOURCES}
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
	./goconst.sh ${VERBOSE}

gosec: ## Run gosec checker
	@echo "Running gosec checker"
	./gosec.sh ${VERBOSE}

abcgo: ## Run ABC metrics checker
	@echo "Run ABC metrics checker"
	./abcgo.sh ${VERBOSE}

json-check: ## Check all JSONs for basic syntax
	@echo "Run JSON checker"
	python3 utils/json_check.py

openapi-check:
	./check_openapi.sh

style: fmt vet lint cyclo shellcheck errcheck goconst gosec ineffassign abcgo json-check ## Run all the formatting related commands (fmt, vet, lint, cyclo) + check shell scripts

run: ${BINARY} ## Build the project and executes the binary
	./$^

automigrate: ${BINARY}
	./$^ migrate latest

test: ${BINARY} ## Run the unit tests
	./unit-tests.sh

cover: test ## Generate HTML pages with code coverage
	@go tool cover -html=coverage.out

coverage: ## Display code coverage on terminal
	@go tool cover -func=coverage.out

integration_tests: ${BINARY} ## Run all integration tests
	@echo "Running all integration tests"
	@./test.sh

rest_api_tests: ${BINARY} ## Run REST API tests
	@echo "Running REST API tests"
	@./test.sh rest_api

license: install_addlicense
	addlicense -c "Red Hat, Inc" -l "apache" -v ./

before_commit: style test integration_tests openapi-check license ## Checks done before commit
	./check_coverage.sh

help: ## Show this help screen
	@echo 'Usage: make <OPTIONS> ... <TARGETS>'
	@echo ''
	@echo 'Available targets are:'
	@echo ''
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ''

function_list: ${BINARY} ## List all functions in generated binary file
	go tool objdump ${BINARY} | grep ^TEXT | sed "s/^TEXT\s//g"

install_addlicense: export GO111MODULE=off
install_addlicense:
	[[ `command -v addlicense` ]] || GO111MODULE=off go get -u github.com/google/addlicense
