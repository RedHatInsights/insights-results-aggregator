.PHONY: help clean build fmt lint vet run test style cyclo

SOURCES:=$(shell find . -name '*.go')

default: build

clean: ## Run go clean
	@go clean

build: ## Run go build
	@go build

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

style: fmt vet lint cyclo ## Run all the formatting related commands (fmt, vet, lint, cyclo)

run: clean build ## Build the project and executes the binary
	./insights-results-aggregator

test: clean build ## Run the unit tests
	@go test -coverprofile coverage.out $(shell go list ./... | grep -v tests)

integration_tests: ## Run all integration tests
	@echo "Running all integration tests"
	@./test.sh

rest_api_tests: ## Run REST API tests
	@echo "Running REST API tests"
	@./test.sh rest_api

sqlite_db:
	mv aggregator.db aggragator.db.backup
	local_storage/create_database_sqlite.sh

help: ## Show this help screen
	@echo 'Usage: make <OPTIONS> ... <TARGETS>'
	@echo ''
	@echo 'Available targets are:'
	@echo ''
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ''
