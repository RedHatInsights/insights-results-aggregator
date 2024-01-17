---
layout: page
nav_order: 5
---
# DB structure
{: .no_toc }

### Note

Please look [at detailed schema
description](https://redhatinsights.github.io/insights-results-aggregator/db-description/)
for more details about tables, indexes, and keys.

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

## List of tables

```
 Schema |                Name                | Type
--------+------------------------------------+------
 public | cluster_rule_toggle                | table
 public | cluster_rule_user_feedback         | table
 public | cluster_user_rule_disable_feedback | table
 public | consumer_error                     | table
 public | migration_info                     | table
 public | recommendation                     | table
 public | report                             | table
 public | rule_hit                           | table
 public | advisor_ratings                    | table
```

## Table `report`

This table is used as a cache for reports consumed from broker. Size of this
table (i.e. number of records) scales linearly with the number of clusters,
because only latest valid report for given cluster is stored (it is guarantied
by DB constraints). That table has defined compound key `org_id+cluster`,
additionally `cluster` name needs to be unique across all organizations.
Additionally `kafka_offset` is used to speedup consuming messages from Kafka
topic in case the offset is lost due to issues in Kafka, Kafka library, or
the service itself (messages with lower offset are skipped):

```sql
CREATE TABLE report (
    org_id          INTEGER NOT NULL,
    cluster         VARCHAR NOT NULL UNIQUE,
    report          VARCHAR NOT NULL,
    reported_at     TIMESTAMP,
    last_checked_at TIMESTAMP,
    kafka_offset    BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY(org_id, cluster)
)
```

We consider a report as valid if it includes all the required fields described
in the [agreed-upon report structure](
https://redhatinsights.github.io/insights-data-schemas/external-pipeline/ccx_data_pipeline.html#format-of-the-report-node).

If any of those fields is missing, we interpret it as a malformed report, most-
probably due to an error when the Insights Core engine processed the archive. In
that situation, the processing of the archive is aborted without storing any new
information in the databases. Therefore, it is important to understand the
difference between:
- an **empty** report, which is stored in the databases, as it indicates that any
previously found issue in the cluster has been resolved or is no longer happening.
- a **malformed** report, which can be empty but is missing required attributes,
and is not stored in the database as there is no guarantee that it represents the
latest state of the cluster.

To learn more about Insights Core processing, please refer to the [Red Hat Insights Core](
https://insights-core.readthedocs.io/en/latest/intro.html#id1) documentation.


## Table `rule_hit`

This table represents the content for Insights rules to be displayed by OCM.

```sql
CREATE TABLE rule_hit (
    org_id         INTEGER NOT NULL,
    cluster_id     VARCHAR NOT NULL,
    rule_fqdn      VARCHAR NOT NULL,
    error_key      VARCHAR NOT NULL,
    template_data  VARCHAR NOT NULL,
    PRIMARY KEY(cluster_id, org_id, rule_fqdn, error_key)
)
```

## Table `cluster_rule_user_feedback`

```sql
-- user_vote is user's vote,
-- 0 is none,
-- 1 is like,
-- -1 is dislike
CREATE TABLE cluster_rule_user_feedback (
    cluster_id  VARCHAR NOT NULL,
    rule_id     VARCHAR NOT NULL,
    error_key   VARCHAR NOT NULL,
    user_id     VARCHAR NOT NULL,
    message     VARCHAR NOT NULL,
    user_vote   SMALLINT NOT NULL,
    added_at    TIMESTAMP NOT NULL,
    updated_at  TIMESTAMP NOT NULL,

    PRIMARY KEY(cluster_id, rule_id, user_id, error_key),
    CONSTRAINT fk_cluster_rule_feedback_report_cluster
    FOREIGN KEY (cluster_id)
        REFERENCES report(cluster)
        ON DELETE CASCADE,
    CONSTRAINT fk_cluster_rule_feedback_rule_module
    FOREIGN KEY (rule_id)
        REFERENCES rule(module)
        ON DELETE CASCADE
)
```

## Table `cluster_rule_toggle`

```sql
CREATE TABLE cluster_rule_toggle (
    cluster_id  VARCHAR NOT NULL,
    rule_id     VARCHAR NOT NULL,
    error_key   VARCHAR NOT NULL,
    user_id     VARCHAR NOT NULL,
    disabled    SMALLINT NOT NULL,
    disabled_at TIMESTAMP NULL,
    enabled_at  TIMESTAMP NULL,
    updated_at  TIMESTAMP NOT NULL,

    disabled_check SMALLINT CHECK (disabled >= 0 AND disabled <= 1),

    PRIMARY KEY(cluster_id, rule_id, user_id)
)
```

## Table `cluster_user_rule_disable_feedback`

Feedback provided by user while disabling the rule on UI.

```sql
CREATE TABLE cluster_user_rule_disable_feedback (
    cluster_id  VARCHAR NOT NULL,
    user_id     VARCHAR NOT NULL,
    rule_id     VARCHAR NOT NULL,
    error_key   VARCHAR NOT NULL,
    message     VARCHAR NOT NULL,
    added_at    TIMESTAMP NOT NULL,
    updated_at  TIMESTAMP NOT NULL,

    PRIMARY KEY(cluster_id, user_id, rule_id, error_key)
)
```

## Table `recommendation`

All recommendations per organization and cluster.

```sql
CREATE TABLE recommendations (
    org_id      INTEGER NOT NULL,
    cluster_id  VARCHAR NOT NULL,
    rule_fqdn   VARCHAR NOT NULL,
    error_key   VARCHAR NOT NULL,
    rule_id     VARCHAR NOT NULL,
    created_at  TIMESTAMP WITHOUT TIME ZONE,

    PRIMARY KEY(org_id, cluster_id, rule_fqdn, error_key)
)
```

## Table `advisor_ratings`

Cluster independent ratings of a recommendation, per user

```sql
CREATE TABLE advisor_ratings (
    user_id VARCHAR NOT NULL,
    org_id VARCHAR NOT NULL,
    rule_fqdn VARCHAR NOT NULL,
    error_key VARCHAR NOT NULL,
    rated_at TIMESTAMP,
    last_updated_at TIMESTAMP,
    rating SMALLINT,
    rule_id VARCHAR NOT NULL,
    PRIMARY KEY(user_id, org_id, rule_fqdn, error_key)
)
```

## Table `consumer_error`

Errors that happen while processing a message consumed from Kafka are logged into this table. This
allows easier debugging of various issues, especially those related to unexpected input data format.

```sql
CREATE TABLE consumer_error (
    topic           VARCHAR NOT NULL,
    partition       INTEGER NOT NULL,
    topic_offset    INTEGER NOT NULL,
    key             VARCHAR,
    produced_at     TIMESTAMP NOT NULL,
    consumed_at     TIMESTAMP NOT NULL,
    message         VARCHAR,
    error           VARCHAR NOT NULL,

    PRIMARY KEY(topic, partition, topic_offset)
)
```

## Table `migration_info`

This table contains just one record with DB version value.

```sql
CREATE TABLE migration_info (
    version         VARCHAR NOT NULL
)
```



## Schema description

DB schema description can be generated by `generate_db_schema_doc.sh` script.
Output is written into directory `docs/db-description/`. Its content can be
viewed [at this
address](https://redhatinsights.github.io/insights-results-aggregator/db-description/).
