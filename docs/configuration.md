---
layout: page
nav_order: 2
---
# Configuration
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

Configuration is done by toml config, default one is `config.toml` in working directory,
but it can be overwritten by `INSIGHTS_RESULTS_AGGREGATOR_CONFIG_FILE` env var.

Also each key in config can be overwritten by corresponding env var. For example if you have config

```toml
[ocp_recommendations_storage]
db_driver = "postgres"
pg_username = "user"
pg_password = "password"
pg_host = "localhost"
pg_port = 5432
pg_db_name = "aggregator"
pg_params = ""
type = "sql"
```

and environment variables

```shell
INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__DB_DRIVER="postgres"
INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__PG_PASSWORD="your secret password"
```

the actual driver will be postgres with password "your secret password"

It's very useful for deploying docker containers and keeping some of your configuration
outside of main config file(like passwords).

### Clowder configuration

In Clowder environment, some configuration options are injected automatically.
Currently Kafka broker configuration is injected this side. To test this
behavior, it is possible to specify path to Clowder-related configuration file
via `ACG_CONFIG` environment variable:

```
export ACG_CONFIG="clowder_config.json"
```

## Broker configuration

Broker configuration is in section `[broker]` in config file

```toml
[broker]
addresses = "localhost:9092"
security_protocol = ""
cert_path = ""
sasl_mechanism = ""
sasl_username = ""
sasl_password = ""
topic = "topic"
timeout = "30s"
payload_tracker_topic = "payload-tracker-topic"
dead_letter_queue_topic = ""
service_name = "insights-results-aggregator"
group = "aggregator"
enabled = true
org_allowlist_file = ""
enable_org_allowlist = false
```

* `addresses` is a comma separated list of addresses of Kafka brokers; e.g kafka:9093,localhost:9092,kafka_2:9092
* `security_protocol` is a value for the `security.protocol` Kafka configuration. Defaults to ""
* `cert_path` is a path to a file containing an SSL certificate, only used if `secutiy_protocol` is properly set to `SSL`
* `sasl_mechanism` is the SASL authentication mechanism to use when `SASL_SSL` is set as `security_protocol`
* `sasl_username` is the SASL username to be used when `SASL_SSL` is set as `security_protocol`
* `sasl_password` is the SASL password to be used when `SASL_SSL` is set as `security_protocol`
* `topic` is a topic to consume messages from (DEFAULT: "")
* `timeout` is the time used as timeout for the Kafka client networking side. See notes above
* `payload_tracker_topic` is a topic to which messages for the Payload Tracker are published (see `producer` package) (DEFAULT: "")
* `dead_letter_queue_topic` is a topic where the non-processed messages will be sent in order to process them later.
* `service_name` is the name of this service as reported to the Payload Tracker (DEFAULT: "")
* `group` is a kafka group (DEFAULT: "")
* `enabled` is an option to turn broker on (DEFAULT: false)
* `org_allowlist_file`
* `enable_org_allowlist`

The offset is stored in the same kafka broker. If it turned off,
consuming will be started from the most recent message (DEFAULT: false)

Option names in env configuration:

* `addresses` - INSIGHTS_RESULTS_AGGREGATOR__BROKER__ADDRESSES
* `security_protocol` - INSIGHTS_RESULTS_AGGREGATOR__BROKER__SECURITY_PROTOCOL
* `cert_path` - INSIGHTS_RESULTS_AGGREGATOR__BROKER__CERT_PATH
* `sasl_mechanism` - INSIGHTS_RESULTS_AGGREGATOR__BROKER__SASL_MECHANISM
* `sasl_username` - INSIGHTS_RESULTS_AGGREGATOR__BROKER__SASL_USERNAME
* `sasl_password` - INSIGHTS_RESULTS_AGGREGATOR__BROKER__SASL_PASSWORD
* `topic` - INSIGHTS_RESULTS_AGGREGATOR__BROKER__TOPIC
* `timeout` - INSIGHTS_RESULTS_AGGREGATOR__BROKER__TIMEOUT
* `payload_tracker_topic` - INSIGHTS_RESULTS_AGGREGATOR__BROKER__PAYLOAD_TRACKER_TOPIC
* `dead_letter_queue_topic` - INSIGHTS_RESULTS_AGGREGATOR__BROKER__DEAD_LETTER_QUEUE_TOPIC
* `service_name` - INSIGHTS_RESULTS_AGGREGATOR__BROKER__SERVICE_NAME
* `group` - INSIGHTS_RESULTS_AGGREGATOR__BROKER__GROUP
* `enabled` - INSIGHTS_RESULTS_AGGREGATOR__BROKER__ENABLED
* `org_allowlist_file` - INSIGHTS_RESULTS_AGGREGATOR__BROKER__ORG_ALLOWLIST_FILE
* `enable_org_allowlist` - INSIGHTS_RESULTS_AGGREGATOR__BROKER__ENABLE_ORG_ALLOWLIST

### About `timeout` definition

The `timeout` configuration should be an string that can be parsed by the
function [`time.ParseDuration`](https://golang.org/pkg/time/#ParseDuration) from
Golang standard library.

This timeout will be applied as the configuration for dial, read and write
timeouts of the Sarama Kafka library.

## Server configuration

Server configuration is in section `[server]` in config file.

```toml
[server]
address = ":8080"
api_prefix = "/api/v1/"
api_spec_file = "openapi.json"
debug = true
auth = true
auth_type = "xrh"
maximum_feedback_message_length = 255
```

* `address` is host and port which server should listen to
* `api_prefix` is prefix for RestAPI path
* `api_spec_file` is the location of a required OpenAPI specifications file
* `debug` is developer mode that enables some special API endpoints not used on production. In
production, `false` is used every time.
* `auth` turns on or turns authentication. Please note that this option can be set to `false` only
in devel environment. In production, `true` is used every time.
* `auth_type` set type of auth, it means which header to use for auth `x-rh-identity` or
`Authorization`. Can be used only with `auth = true`. Possible options: `jwt`, `xrh`
* `maximum_feedback_message_length` is a maximum possible length of a string for user's feedback

Please note that if `auth` configuration option is turned off, not all REST API endpoints will be
usable. Whole REST API schema is satisfied only for `auth = true`.

## CloudWatch configuration

CloudWatch configuration is in section `[cloudwatch]` in config file

```toml
[cloudwatch]
aws_access_id = "a key id"
aws_secret_key = "tshhhh it is a secret"
aws_session_token = ""
aws_region = "us-east-1"
log_group = "platform-dev"
stream_name = "insights-results-aggregator"
debug = false
```

* `aws_access_id` is an aws access id
* `aws_secret_key` is an aws secret key
* `aws_session_token` is an aws session token
* `aws_region` is an aws region
* `log_group` is a log group for aws logging
* `stream_name` is a stream name for aws logging. If you're deploying multiple pods,
you can add `$HOSTNAME` to the stream name so that they aren't writing to the same stream at once
* `debug` is an option to enable debug output of cloudwatch logging

## Storage configuration

Two storage backends can be configured separately:

* Storage for OCP recommendations
* Storage for DVO recommendations

For each storage, specific section in configuration file is used:

```toml
[ocp_recommendations_storage]
db_driver = "postgres"
pg_username = "user"
pg_password = "password"
pg_host = "localhost"
pg_port = 5432
pg_db_name = "aggregator"
pg_params = ""
log_sql_queries = true
type = "sql"

[dvo_recommendations_storage]
db_driver = "postgres"
pg_username = "user"
pg_password = "password"
pg_host = "localhost"
pg_port = 5432
pg_db_name = "aggregator"
pg_params = ""
log_sql_queries = true
type = "sql"
```

Actually used storage backend is selected by the following configuration option:

```toml
[storage_backend]
use = "ocp_recommendations"
```

By default OCP recommendations storage is selected if no backend is configured.

## Redis configuration

Redis configuration is in section `[redis]` in config file

```toml
[redis]
database = 0
endpoint = "localhost:6379"
password = ""
timeout_seconds = 30
```

* Please note that Redis databases are numbered from 0 to 15 and that default value is 0
* Also please note that Redis database will be used only if `type=redis`


## Metrics configuration

Metrics configuration is in section `[metrics]` in config file

```toml
[metrics]
namespace = "mynamespace"
```

* `namespace` if defined, it is used as `Namespace` argument when creating all
  the Prometheus metrics exposed by this service.
