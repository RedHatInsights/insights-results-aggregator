---
layout: page
nav_order: 7
---
# Prometheus API

It is possible to use `/api/v1/metrics` REST API endpoint to read all metrics exposed to Prometheus
or to any tool that is compatible with it.
Currently, the following metrics are exposed:

1. `consumed_messages` the total number of messages consumed from Kafka
1. `consuming_errors` the total number of errors during consuming messages from Kafka
1. `successful_messages_processing_time` the time to process successfully message
1. `failed_messages_processing_time` the time to process message fail
1. `last_checked_timestamp_lag_minutes` shows how slow we get messages from clusters
1. `produced_messages` the total number of produced messages
1. `written_reports` the total number of reports written to the storage
1. `feedback_on_rules` the total number of left feedback
1. `sql_queries_counter` the total number of SQL queries
1. `sql_queries_durations` the SQL queries durations

Additionally it is possible to consume all metrics provided by Go runtime. There metrics start with
`go_` and `process_` prefixes.

## API related metrics

There are a set of metrics provieded by `insights-operator-utils` library, all
of them related with the API usage. These are the API metrics exposed:

1. `api_endpoints_requests` the total number of requests per endpoint
1. `api_endpoints_response_time` API endpoints response time
1. `api_endpoints_status_codes` a counter of the HTTP status code responses
   returned back by the service

## Metrics namespace

As explained in the [configuration](./configuration) section of this
documentation, a namespace can be provided in order to act as a prefix to the
metric name. If no namespace is provided in the configuration, the metrics will
be exposed as described in this documentation.
