---
layout: page
nav_order: 3
---

# Clowder configuration

As the rest of the services deployed in the Console Red Hat platform, the
Insights Results Aggregator DB Writer should update its configuration
using the relevant values extracted from the Clowder configuration file.

For a general overview about Clowder configuration file and how it is used
in the external data pipeline, please refer to
[Clowder configuration](https://ccx.pages.redhat.com/ccx-docs/customer/clowder.html)
in CCX Docs.

## How can we deal with the Clowder configuration file?

As the service is implemented in Golang, we take advantage of using
[app-common-go](https://github.com/RedHatInsights/app-common-go/).
It provides to the service all the needed values for configuring Kafka
access and the topics name mapping.

The `conf` package is taking care of reading the data structures provided
by `app-common-go` library and merge Clowder configuration with the service
configuration.

# Insights Results Aggregator specific relevant values

This service is running in 3 different modes in the platform:

- DB Writer: the service connects to Kafka to receive messages in a
  specific topic and write the results into a SQL database.
- Cache Writer: the service connects to Kafka to receive messages in a
  specific topic and write the results into Redis.
- Results Aggregator: expose the database stored data into several API
  endpoints.

For this reason, the relevant values for both types of instances are a bit
different:

- DB Writer needs to update its Kafka access configuration and its DB
  access configuration in order to work.
- Cache Writer needs to update its Kafka access configuration and its DB
  access configuration in order to work.
- Results Aggregator just need to update its DB access configuration.
