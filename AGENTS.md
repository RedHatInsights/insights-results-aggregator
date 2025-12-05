# AGENTS.md

## Project Overview
insights-results-aggregator is a Go-based service for the Red Hat Insights ecosystem that aggregates and exposes OpenShift cluster recommendations data. It consumes Insights OCP data (recommendations) from message brokers, stores them in various storage backends (PostgreSQL, Redis), and exposes them via REST API endpoints for consumption by OpenShift Cluster Manager (OCM), Advanced Cluster Manager (ACM), and OCP Web Console via Insights Operator.

**Tech Stack**: Go 1.22+, PostgreSQL, Redis, Kafka (Sarama), Prometheus, Sentry/Glitchtip, REST API

## Repository Structure
```text
/broker/                 - Kafka broker integration (consumer implementation)
/consumer/               - Message consumer logic and processing
/producer/               - Message producer for outgoing data
/storage/                - Database layer (PostgreSQL, Redis implementations)
/server/                 - HTTP server and REST API handlers
/metrics/                - Prometheus metrics definitions and collectors
/migration/              - Database migration scripts
  /ocpmigrations/        - OpenShift-specific migrations
  /dvomigrations/        - DVO-specific migrations
/types/                  - Type definitions and data structures
/utils/                  - Utility functions and helpers
/local_storage/          - Local storage implementations
/rules/                  - Rules processing logic
/tests/                  - Integration and REST API tests
  /rest/                 - REST API test suites
  /helpers/              - Test helpers and utilities
  /data/                 - Test data files
  /content/              - Test content fixtures

/docs/                   - Documentation (GitHub Pages)
  /db-description/       - Database schema documentation
  /presentations/        - Architecture presentations
  /assets/               - Documentation assets

/deploy/                 - Deployment configurations
/dashboards/             - Grafana dashboards
/.tekton/                - Tekton CI/CD pipelines
/.github/                - GitHub Actions workflows

aggregator.go            - Main aggregator service implementation
consumer.go              - Consumer interface and base implementation
config.toml              - Production configuration example
config-devel.toml        - Development configuration
Dockerfile               - Container image build script
Makefile                 - Build and development targets
```

## Development Workflow

### Setup
- **Go version**: 1.22+ required
- **Database**: PostgreSQL
- **Cache**: Redis (optional)

### Running Tests
- `make test` - Unit tests
- `make cover` - HTML coverage report
- `make coverage` - Coverage in the terminal
- `make integration_tests` - Integration tests
- `make rest_api_tests` - REST API tests
- `go test -v ./...` - Direct test execution

### Code Quality
- `make check` - golangci-lint (goconst, gocyclo, gosec, etc.)
- `make shellcheck` - Shell script linting
- `make abcgo` - ABC metrics checker
- `make json-check` - JSON validation
- `make openapi-check` - OpenAPI validation
- `make style` - Formatting and style checks
- `make before_commit` - Pre-commit suite (style, tests, coverage, license)

### Building and Running
- `make build` - Build binary
- `make build-cover` - Build with coverage support
- `make run` - Build and execute

### CLI Commands
- `./insights-results-aggregator` - Start service
- `./insights-results-aggregator print-config` - Print config
- `./insights-results-aggregator print-env` - Show env vars
- `./insights-results-aggregator migration` - Check migration status
- `./insights-results-aggregator migration <version>` - Run migrations

## Key Architectural Patterns

### Data Flow Architecture

This service acts as an **aggregator and cache layer**. The main flow is: consume from Kafka → store in DB → serve via REST API.

1. **Message Consumption**: Kafka consumer receives pre-processed results from upstream services (rules applied elsewhere, not here)
2. **Data Validation**: Messages validated and processed
3. **Storage/Caching**: PostgreSQL (persistent) + optional Redis (cache)
4. **API Exposure**: REST API serves data to OCM, ACM, OCP WebConsole
5. **Metrics**: Prometheus tracks service health

### Storage Architecture

- **PostgreSQL**: Primary store with OCP and DVO schemas (separate migrations)
- **Redis**: Optional cache

### Configuration

- `config.toml` - Production config
- `config-devel.toml` - Development config
- Environment variables override config files

## Working with this Repository

**As an agent, you should create a TODO list** when working on tasks to track progress and ensure all steps are completed systematically.

## Code Conventions

### Go Style
- **Linters**: golangci-lint (goconst, gocyclo, gofmt, goimports, gosec, gosimple, nilerr, prealloc, revive, staticcheck, unconvert, unused, whitespace, zerologlint)
- **Formatting**: gofmt
- **Documentation**: GoDoc comments required for exported symbols
- **Error handling**: Explicit returns, custom error types for domain-specific errors

### Naming Patterns
- Test files: `*_test.go`
- Storage: `*Storage` interfaces/structs
- HTTP handlers: `Handle*` pattern
- Metrics: snake_case
- Config: lowercase_with_underscores in TOML

### Error Handling
- Return errors explicitly, no panics in production
- Custom error types for domain errors
- Log with context (zerolog) and request IDs

### Database Migrations
- SQL migrations in `/migration/` (separate paths for OCP/DVO)
- Test both up and down
- Never modify existing migrations

## Important Notes

### Dependencies
- **Kafka client**: Shopify/sarama (NOT confluent-kafka or kafka-go)
- **Database driver**: lib/pq for PostgreSQL
- **Redis client**: go-redis/v9
- **Logging**: rs/zerolog for structured logging
- **Metrics**: prometheus/client_golang
- **Common utilities**: redhatinsights/app-common-go for platform integration

### Testing
- Unit tests: standard Go testing
- Integration tests: require PostgreSQL and Kafka
- REST API tests: frisby framework (BDD-style)
- Mocking: DATA-DOG/go-sqlmock

### Behavioral Tests
External BDD tests in [Insights Behavioral Spec](https://github.com/RedHatInsights/insights-behavioral-spec) via `insights_results_aggregator_tests.sh`

### Monitoring
- Prometheus metrics (configurable port)
- Sentry for error tracking
- CloudWatch (optional)
- Health endpoints: `/health`, `/ready`

## Pull Request Guidelines

### Before Creating a PR

Run `make before_commit` to verify tests, linting, coverage, and code quality.

### PR Requirements
- **Reviews**: Minimum 2 approvals from maintainers
- **Commit messages**: Clear, descriptive messages explaining the "why" rather than the "what"
  - No specific convention required
  - If related to a Jira task, optionally include the ticket ID: `[CCXDEV-12345] Description`
- **Documentation**: Update docs for API or behavior changes
- **Migrations**: Include database migrations if schema changes
- **Base branch**: `master` (main development branch)
- **Breaking changes**: Must be documented and communicated
- **WIP PRs**: Tag with `[WIP]` prefix in title to prevent accidental merging

### PR Checklist

Before pushing changes, ensure:

- All tests pass (`make test`)
- Integration tests pass (`make integration_tests`)
- Code passes linting (`make check`)
- Coverage has not decreased (`make coverage`)
- OpenAPI spec is valid (`make openapi-check`)
- License headers are present (`make license`)
- Documentation is updated (if API or behavior changes)
- Database migrations included (if schema changes)
- No merge conflicts with master branch

## Deployment Information

- **Deployment configs**: Located in `/deploy` directory
- **Tekton pipelines**: CI/CD definitions in `/.tekton`
- **Container**: Built via Dockerfile, published to registry
- **Configuration management**: Via app-interface (internal Red Hat)
- **Environments**: Ephemeral (testing), stage, production
- **Dashboards**: Grafana dashboards in `/dashboards` for monitoring

## Security Considerations
- **Never log sensitive data**: Organization IDs should be sanitized, no auth tokens, no PII
- **Database credentials**: Use environment variables or config files, never hardcode
- **Kafka credentials**: Configure in TOML or via environment variables, use SASL authentication when required
- **API authentication**: Validate all incoming requests
- **Input validation**: Sanitize and validate all user inputs
- **SQL injection**: Use parameterized queries, never string concatenation
- **RBAC**: Respect organization boundaries, validate user access

## Debugging Tips
- **Test coverage**: Generate HTML reports with `make cover` to identify untested code
- **Configuration check**: Use `./insights-results-aggregator print-config` to verify settings
- **Migration status**: Use `./insights-results-aggregator migration` to check database state
- **REST API tests**: Run specific test suites in `/tests/rest/` for endpoint validation

## External References
- [GitHub Pages Documentation](https://redhatinsights.github.io/insights-results-aggregator/)
- [GoDoc API Documentation](https://godoc.org/github.com/RedHatInsights/insights-results-aggregator)
- [Insights Behavioral Spec](https://github.com/RedHatInsights/insights-behavioral-spec)
- [Sarama Kafka Client](https://github.com/Shopify/sarama)
- [Zerolog Documentation](https://github.com/rs/zerolog)
