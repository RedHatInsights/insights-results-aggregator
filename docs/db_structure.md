---
layout: page
nav_order: 4
---
# DB structure
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

## Table report

This table is used as a cache for reports consumed from broker. Size of this
table (i.e. number of records) scales linearly with the number of clusters,
because only latest report for given cluster is stored (it is guarantied by DB
constraints). That table has defined compound key `org_id+cluster`,
additionally `cluster` name needs to be unique across all organizations.

```sql
CREATE TABLE report (
    org_id          INTEGER NOT NULL,
    cluster         VARCHAR NOT NULL UNIQUE,
    report          VARCHAR NOT NULL,
    reported_at     TIMESTAMP,
    last_checked_at TIMESTAMP,
    PRIMARY KEY(org_id, cluster)
)
```

## Tables rule and rule_error_key

These tables represent the content for Insights rules to be displayed by OCM.
The table `rule` represents more general information about the rule, whereas the `rule_error_key`
contains information about the specific type of error which occurred. The combination of these two
create a unique rule.
Very trivialized example could be:

* rule "REQUIREMENTS_CHECK"
  * error_key "REQUIREMENTS_CHECK_LOW_MEMORY"
  * error_key "REQUIREMENTS_CHECK_MISSING_SYSTEM_PACKAGE"

```sql
CREATE TABLE rule (
    module        VARCHAR PRIMARY KEY,
    name          VARCHAR NOT NULL,
    summary       VARCHAR NOT NULL,
    reason        VARCHAR NOT NULL,
    resolution    VARCHAR NOT NULL,
    more_info     VARCHAR NOT NULL
)
```

```sql
CREATE TABLE rule_error_key (
    error_key     VARCHAR NOT NULL,
    rule_module   VARCHAR NOT NULL REFERENCES rule(module),
    condition     VARCHAR NOT NULL,
    description   VARCHAR NOT NULL,
    impact        INTEGER NOT NULL,
    likelihood    INTEGER NOT NULL,
    publish_date  TIMESTAMP NOT NULL,
    active        BOOLEAN NOT NULL,
    generic       VARCHAR NOT NULL,
    PRIMARY KEY(error_key, rule_module)
)
```

## Table cluster_rule_user_feedback

```sql
-- user_vote is user's vote,
-- 0 is none,
-- 1 is like,
-- -1 is dislike
CREATE TABLE cluster_rule_user_feedback (
    cluster_id VARCHAR NOT NULL,
    rule_id VARCHAR NOT NULL,
    user_id VARCHAR NOT NULL,
    message VARCHAR NOT NULL,
    user_vote SMALLINT NOT NULL,
    added_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,

    PRIMARY KEY(cluster_id, rule_id, user_id),
    FOREIGN KEY (cluster_id)
        REFERENCES report(cluster)
        ON DELETE CASCADE,
    FOREIGN KEY (rule_id)
        REFERENCES rule(module)
        ON DELETE CASCADE
)
```

## Table cluster_rule_toggle

```sql
CREATE TABLE cluster_rule_toggle (
    cluster_id VARCHAR NOT NULL,
    rule_id VARCHAR NOT NULL,
    user_id VARCHAR NOT NULL,
    disabled SMALLINT NOT NULL,
    disabled_at TIMESTAMP NULL,
    enabled_at TIMESTAMP NULL,
    updated_at TIMESTAMP NOT NULL,

    CHECK (disabled >= 0 AND disabled <= 1),

    PRIMARY KEY(cluster_id, rule_id, user_id)
)
```

## Table consumer_error

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
