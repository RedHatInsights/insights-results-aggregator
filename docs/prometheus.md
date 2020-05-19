---
layout: page
nav_order: 7
---
# Prometheus API

It is possible to use `/api/v1/metrics` REST API endpoint to read all metrics exposed to Prometheus
or to any tool that is compatible with it.
Currently, the following metrics are exposed:

1. `api_endpoints_requests` the total number of requests per endpoint
1. `api_endpoints_response_time` API endpoints response time
1. `consumed_messages` the total number of messages consumed from Kafka
1. `feedback_on_rules` the total number of left feedback
1. `produced_messages` the total number of produced messages
1. `written_reports` the total number of reports written to the storage

Additionally it is possible to consume all metrics provided by Go runtime. There metrics start with
`go_` and `process_` prefixes.
