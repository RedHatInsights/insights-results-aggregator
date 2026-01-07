---
layout: page
---
# Database retention policy
{: .no_toc }



## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}



## List of tables

All tables that are stored in external data pipeline database (OCP Recommendations):

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

## Retention policy

This is the required retention policy that is watched by
`insights-results-aggregator-cleaner` tool:

```
              Table name            | Retention period
------------------------------------+-----------------
 cluster_rule_toggle                | inf.
 cluster_rule_user_feedback         | inf.
 cluster_user_rule_disable_feedback | inf.
 consumer_error                     | manual investigation
 migration_info                     | inf.
 recommendation                     | 2 months
 report                             | 2 months
 rule_hit                           | 2 months
 advisor_ratings                    | inf.
```

## Tables without retention policy

```
              Table name            | Retention period
------------------------------------+-----------------
 migration_info                     | inf.
```

This table contains just one record with DB version value. Does not have (and
should not) to be cleaned up.

```
              Table name            | Retention period
------------------------------------+-----------------
 cluster_rule_toggle                | inf.
 cluster_rule_user_feedback         | inf.
 cluster_user_rule_disable_feedback | inf.
 advisor_ratings                    | inf.
```

These tables contain feedback provided by users (feedback provided by user
while disabling the rule on UI, cluster independent ratings of a
recommendation, per user) and might be used for statistical purposes. Sizes of
such tables are very small compared with other tables and does not casuse any
bottleneck in the external data pipeline. Thus no retention policy needs to be
followed there.



## Tables with retention policy

```
              Table name            | Retention period
------------------------------------+-----------------
 recommendation                     | 2 months
 report                             | 2 months
 rule_hit                           | 2 months
```

All three tables listed above should contain data for "live" (i.e. connected)
clusters. And these tables are critical component of the whole external data
pipeline and might be the bottleneck for the system. So it is desirable to
cleanup old disconnected clusters. Retention policy set to 2 months is set to
keep some historical data for clusters disconnected in recent days.

```
              Table name            | Retention period
------------------------------------+-----------------
 consumer_error                     | manual investigation
```

That table contains errors that happen while processing a message consumed from
Kafka are logged into this table. This allows easier debugging of various
issues, especially those related to unexpected input data format.

Content of this table needs to be investigated manually and then cleaned by the
compound key (topic+partition+topic_offset).
