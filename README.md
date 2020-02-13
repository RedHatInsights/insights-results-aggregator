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
