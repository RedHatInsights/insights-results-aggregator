# Insights Results Aggregator

[![GoDoc](https://godoc.org/github.com/RedHatInsights/insights-results-aggregator?status.svg)](https://godoc.org/github.com/RedHatInsights/insights-results-aggregator) [![Go Report Card](https://goreportcard.com/badge/github.com/RedHatInsights/insights-results-aggregator)](https://goreportcard.com/report/github.com/RedHatInsights/insights-results-aggregator) [![Build Status](https://travis-ci.org/RedHatInsights/insights-results-aggregator.svg?branch=master)](https://travis-ci.org/RedHatInsights/insights-results-aggregator) [![codecov](https://codecov.io/gh/RedHatInsights/insights-results-aggregator/branch/master/graph/badge.svg)](https://codecov.io/gh/RedHatInsights/insights-results-aggregator)

Aggregator service for insights results

## Description

## Architecture

### Whole data flow

![data_flow](./doc/customer-facing-services-architecture.png)

1. Event about new data from insights operator is consumed from Kafka. That event contains (among other things) URL to S3 Bucket
2. Insights operator data is read from S3 Bucket and insigts rules are applied to that data
3. Results (basically organization ID + cluster name + insights results JSON) are stored back into Kafka, but into different topic
4. That results are consumed by Insights rules aggregator service that caches them
5. The service provides such data via REST API to other tools, like OpenShift Cluster Manager web UI, OpenShift console, etc.

## Utilities

### produce_insights_results

This shell script can be used to produce several Insights results into Kafka topic. Its dependency is Kafkacat that needs to be installed on the same machine.

## Documentation for developers

All packages developed in this project have documentation available on [GoDoc server](https://godoc.org/):

* [entry point to the service](https://godoc.org/github.com/RedHatInsights/insights-results-aggregator)
* [package `broker`](https://godoc.org/github.com/RedHatInsights/insights-results-aggregator/broker)
* [package `consumer`](https://godoc.org/github.com/RedHatInsights/insights-results-aggregator/consumer)
* [package `producer`](https://godoc.org/github.com/RedHatInsights/insights-results-aggregator/producer)
* [package `server`](https://godoc.org/github.com/RedHatInsights/insights-results-aggregator/server)
* [package `storage`](https://godoc.org/github.com/RedHatInsights/insights-results-aggregator/storage)
* [package `types`](https://godoc.org/github.com/RedHatInsights/insights-results-aggregator/types)

## Database

By default aggregator uses SQLite3 DB, but also it has support of PostgreSQL. For starting PostgreSQL exist script in folder `local_storage`:
```Bash
./dockerize_postgres.sh
```

For establish connection to PostgreSQL, the following configuration options needs to be changed in `storage` section of `config.toml`:

```
[storage]
driver = "postgres"
datasource = "postgres://postgres:postgres@localhost/controller?sslmode=disable"
```

## Local setup

### Kafka broker

This service depends on Kafka broker. It can be installed and configured locally. Please follow these steps to configure Kafka:

1. Download the stable Kafka version from [this link](https://www.apache.org/dyn/closer.cgi?path=/kafka/2.4.0/kafka_2.12-2.4.0.tgz)
1. Uncompress downloaded tarball: `tar -xzf kafka_2.12-2.4.0.tgz`
1. Change current directory to a newly created directory: `cd kafka_2.12-2.4.0`
1. Start Zookeeper service: `bin/zookeeper-server-start.sh config/zookeeper.properties`
1. Start Kafka broker: `bin/kafka-server-start.sh config/server.properties`

### Kafka producer

It is possible to use the script `produce_insights_results` from `utils` to produce several Insights results into Kafka topic. Its dependency is Kafkacat that needs to be installed on the same machine. You can find installation instructions [on this page](https://github.com/edenhill/kafkacat).

## Testing

### Unit tests:

`make test`

### All integration tests:

`make integration_tests`

#### Only rest api tests

`make rest_api_tests`

#### Only metrics tests

`make metrics_tests`
