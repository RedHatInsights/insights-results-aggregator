---
layout: page
nav_order: 14
---
# Local setup on bare metal
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}



## Synopsis

Sometimes it is needed to be able to run Insights Results Aggregator, DB
Writer, PostgreSQL database, and Kafka instance on bare metal. Just on bare
metal it would be possible to run benchmarks or performance tests and get
reliable results. This document contains description how to setup all such
tools and services on brand new Fedora 35 Server system.



## Basic setup

Some preparation steps to make the system usable and prepared to perform
following steps.



### System info

All steps described in this document have been performed on brand new Fedora 35
Server installation:

```
$ cat /etc/fedora-release 

Fedora release 35 (Thirty Five)
```

Kernel version:

```
$ uname -a

Linux hpe-dl380pgen8-02-vm-15.hpe2.lab.eng.bos.redhat.com 5.16.15-201.fc35.x86_64 #1 SMP PREEMPT Thu Mar 17 05:45:13 UTC 2022 x86_64 x86_64 x86_64 GNU/Linux
```



### Common packages to be used

First we need to install several packages that are not included in default
Fedora Server system:

```
# dnf install git
# dnf install wget
# dnf install tar
```


## PostgreSQL installation and setup

Install local PostgreSQL in case you need to run benchmarks and performance
tests against local DB running on bare metal.



### Package installation

Install:

```
# dnf install postgresql-server
```

Check installation:

```
$ psql --version

psql (PostgreSQL) 13.4
```



### Database setup

Initialize database:

```
# /usr/bin/postgresql-setup --initdb

 * Initializing database in '/var/lib/pgsql/data'
 * Initialized, logs are in /var/lib/pgsql/initdb_postgresql.log
```

Setup how users and services will be logged into DB:

```
# vim /var/lib/pgsql/data/pg_hba.conf
```

Change to:

```
# "local" is for Unix domain socket connections only
# local   all             all                                     peer
local   all             all                                     password
# IPv4 local connections:
host    all             all             127.0.0.1/32            password
# IPv6 local connections:
host    all             all             ::1/128                 password
```



### Start DB service

Start the service:

```
# systemctl start postgresql

# systemctl status postgresql

● postgresql.service - PostgreSQL database server
     Loaded: loaded (/usr/lib/systemd/system/postgresql.service; disabled; vendor preset: disabled)
     Active: active (running) since Tue 2022-03-22 05:51:59 EDT; 2s ago
    Process: 31514 ExecStartPre=/usr/libexec/postgresql-check-db-dir postgresql (code=exited, status=0/SUCCESS)
   Main PID: 31516 (postmaster)
      Tasks: 8 (limit: 4661)
     Memory: 15.9M
        CPU: 34ms
     CGroup: /system.slice/postgresql.service
             ├─31516 /usr/bin/postmaster -D /var/lib/pgsql/data
             ├─31517 "postgres: logger " "" "" "" "" "" "" "" "" "" "" "" "" "" "" "" "" "" "" "" "" "" "" "" "" ""
             ├─31519 "postgres: checkpointer " "" "" "" "" "" "" "" "" "" "" "" "" "" "" "" "" "" "" ""
             ├─31520 "postgres: background writer " "" "" "" "" "" "" "" "" "" "" "" "" "" ""
             ├─31521 "postgres: walwriter " "" "" "" "" "" "" "" "" "" "" "" "" "" "" "" "" "" "" "" "" "" ""
             ├─31522 "postgres: autovacuum launcher " "" "" "" "" "" "" "" "" "" "" "" ""
             ├─31523 "postgres: stats collector " "" "" "" "" "" "" "" "" "" "" "" "" "" "" "" ""
             └─31524 "postgres: logical replication launcher " "" "" ""
```

Check if the service has been really started and look for any error message or warning:

```
Mar 22 05:51:59 hpe-dl380pgen8-02-vm-15.hpe2.lab.eng.bos.redhat.com systemd[1]: Starting PostgreSQL database server...
Mar 22 05:51:59 hpe-dl380pgen8-02-vm-15.hpe2.lab.eng.bos.redhat.com postmaster[31516]: 2022-03-22 05:51:59.877 EDT [31516] LOG:  redirecting log output to log>
Mar 22 05:51:59 hpe-dl380pgen8-02-vm-15.hpe2.lab.eng.bos.redhat.com postmaster[31516]: 2022-03-22 05:51:59.877 EDT [31516] HINT:  Future log output will appea>
Mar 22 05:51:59 hpe-dl380pgen8-02-vm-15.hpe2.lab.eng.bos.redhat.com systemd[1]: Started PostgreSQL database server.
```


## Kafka installation and setup

Install local Kafka in case you need to run benchmarks and performance
tests against local message broker.



### Package installation

Install Java package:

```
# dnf install java
```

Check installation:

```
$ java -version
openjdk version "11.0.14.1" 2022-02-08
OpenJDK Runtime Environment 18.9 (build 11.0.14.1+1)
OpenJDK 64-Bit Server VM 18.9 (build 11.0.14.1+1, mixed mode, sharing)
```

Get Kafka package:

```
$ wget https://dlcdn.apache.org/kafka/3.1.0/kafka_2.12-3.1.0.tgz

--2022-03-22 10:13:01--  https://dlcdn.apache.org/kafka/3.1.0/kafka_2.12-3.1.0.tgz
Resolving dlcdn.apache.org (dlcdn.apache.org)... 2a04:4e42::644, 151.101.2.132
Connecting to dlcdn.apache.org (dlcdn.apache.org)|2a04:4e42::644|:443... failed: Network is unreachable.
Connecting to dlcdn.apache.org (dlcdn.apache.org)|151.101.2.132|:443... connected.
HTTP request sent, awaiting response... 200 OK
Length: 88217241 (84M) [application/x-gzip]
Saving to: ‘kafka_2.12-3.1.0.tgz’

kafka_2.12-3.1.0.tg 100%[===================>]  84.13M  93.7MB/s    in 0.9s

2022-03-22 10:13:02 (93.7 MB/s) - ‘kafka_2.12-3.1.0.tgz’ saved [88217241/88217241]
```



### Start Zookeeper and Kafka

Start the Zookeeper first:

```
$ cd kafka_2.12-3.1.0

$ bin/zookeeper-server-start.sh config/zookeeper.properties

[2022-03-22 10:34:33,179] INFO Reading configuration from: config/zookeeper.properties (org.apache.zookeeper.server.quorum.QuorumPeerConfig)
[2022-03-22 10:34:33,179] INFO Reading configuration from: config/zookeeper.properties (org.apache.zookeeper.server.quorum.QuorumPeerConfig)
[2022-03-22 10:34:33,190] INFO clientPortAddress is 0.0.0.0:2181 (org.apache.zookeeper.server.quorum.QuorumPeerConfig)
```

Then start Kafka broker itself:

```
$ cd kafka_2.12-3.1.0

$ bin/kafka-server-start.sh config/server.properties

[2022-03-22 10:36:47,534] INFO Registered kafka:type=kafka.Log4jController MBean (kafka.utils.Log4jControllerRegistration$)
[2022-03-22 10:36:48,109] INFO Setting -D jdk.tls.rejectClientInitiatedRenegotiation=true to disable client-initiated TLS renegotiation (org.apache.zookeeper.common.X509Util)
[2022-03-22 10:36:50,467] INFO [BrokerToControllerChannelManager broker=0 name=alterIsr]: Recorded new controller, from now on will use broker hpe-dl380pgen8-02-vm-15.hpe2.lab.eng.bos.redhat.com:9092 (id: 0 rack: null) (kafka.server.BrokerToControllerRequestThread)
[2022-03-22 10:36:50,511] INFO [BrokerToControllerChannelManager broker=0 name=forwarding]: Recorded new controller, from now on will use broker hpe-dl380pgen8-02-vm-15.hpe2.lab.eng.bos.redhat.com:9092 (id: 0 rack: null) (kafka.server.BrokerToControllerRequestThread)
```



## Kafkacat installation and setup

Install Kafkacat (now named Kcat) in order to be able to publish messages during benchmarking and testing.



### Installation

Install all required packages first:

```
# dnf install gcc-g++ cmake
# dnf install cyrus-sasl-devel zlib-devel libcurl-devel krb5-devel
```

Retrieve Kafkacat/Kcat sources:

```
$ git clone https://github.com/edenhill/kafkacat.git
$ cd kafkacat
```



### Build

Try to build Kafkacat/Kcat:

```
$ ./bootstrap.sh
```

Please note that sometimes it is needed to fix some "side" errors like
rebuilding `libyajl` by hands (it is located in `tmp-bootstrap/libyajl`
subdirectory.

Last check if binary has been produced can be made:

```
$ ./kcat -V

kcat - Apache Kafka producer and consumer tool
https://github.com/edenhill/kcat
Copyright (c) 2014-2021, Magnus Edenhill
Version 1.7.1-2-g338ae3 (JSON, Avro, Transactions, IncrementalAssign, JSONVerbatim, librdkafka 1.8.2 builtin.features=gzip,snappy,ssl,sasl,regex,lz4,sasl_gssapi,sasl_plain,sasl_scram,plugins,zstd,sasl_oauthbearer)
```



### Check connection to Kafka

Now check if Kafkacat/kcat works as expected. We can use local broker that's been started already:

```
$ ./kcat -L -b localhost:9092

Metadata for all topics (from broker -1: localhost:9092/bootstrap):
 1 brokers:
  broker 0 at hpe-dl380pgen8-02-vm-15.hpe2.lab.eng.bos.redhat.com:9092 (controller)
 0 topics:
```


## Insights Results Aggregator installation and setup

Now Insights Results Aggregator can be installed and setup.



### Dependend packages installation

Install Go:

```
# dnf install go
```

Check if Go was installed correctly:

```
$ go version

go version go1.16.15 linux/amd64
```



### Insights Results Aggregator setup

Clone the repository:

```
$ git clone https://github.com/RedHatInsights/insights-results-aggregator.git

Cloning into 'insights-results-aggregator'...
remote: Enumerating objects: 10761, done.
remote: Counting objects: 100% (786/786), done.
remote: Compressing objects: 100% (451/451), done.
remote: Total 10761 (delta 469), reused 527 (delta 326), pack-reused 9975
Receiving objects: 100% (10761/10761), 11.32 MiB | 21.07 MiB/s, done.
Resolving deltas: 100% (7198/7198), done.
```

Build the service:

```
$ cd insights-results-aggregator
$ make
```

Check if the service has been built correctly:

```
$ ./insights-results-aggregator --help

Clowder is not enabled, skipping init...
Clowder is disabled

Command '--help' not found

Aggregator service for insights results

Usage:

    ./insights-results-aggregator [command]

The commands are:

    <EMPTY>             starts aggregator
    start-service       starts aggregator
    help                prints help
    print-help          prints help
    print-config        prints current configuration set by files & env variables
    print-env           prints env variables
    print-version-info  prints version info
    migration           prints information about migrations (current, latest)
    migration <version> migrates database to the specified version
```



### Aggregator database setup

First, empty database needs to be prepared in PostgreSQL:

```
# psql -U postgres

Password for user postgres:
psql (13.4)
Type "help" for help.



postgres=# create database aggregator;
CREATE DATABASE
postgres=# \q
```

Check if migration have not happed (yet):

```
$ ./insights-results-aggregator migration

Clowder is not enabled, skipping init...
Clowder is disabled
{"level":"info","time":"2022-03-22T07:21:57-04:00","message":"Making connection to data storage, driver=postgresWithHooks datasource=postgresql://postgres:postgres@localhost:5432/aggregator?sslmode=disable"}
{"level":"debug","type":"SQL","time":"2022-03-22T07:21:57-04:00","message":"query `CREATE TABLE IF NOT EXISTS migration_info (version INTEGER NOT NULL);` with params `[]`\n"}
{"level":"debug","type":"SQL","time":"2022-03-22T07:21:57-04:00","message":"query `CREATE TABLE IF NOT EXISTS migration_info (version INTEGER NOT NULL);` with params `[]` took 2.017535ms\n"}
{"level":"debug","type":"SQL","time":"2022-03-22T07:21:57-04:00","message":"query `INSERT INTO migration_info (version) SELECT 0 WHERE NOT EXISTS (SELECT version FROM migration_info);` with params `[]`\n"}
{"level":"debug","type":"SQL","time":"2022-03-22T07:21:57-04:00","message":"query `INSERT INTO migration_info (version) SELECT 0 WHERE NOT EXISTS (SELECT version FROM migration_info);` with params `[]` took 696.923µs\n"}
{"level":"debug","type":"SQL","time":"2022-03-22T07:21:57-04:00","message":"query `SELECT COUNT(*) FROM migration_info;` with params `[]`\n"}
{"level":"debug","type":"SQL","time":"2022-03-22T07:21:57-04:00","message":"query `SELECT COUNT(*) FROM migration_info;` with params `[]` took 528.857µs\n"}
{"level":"debug","type":"SQL","time":"2022-03-22T07:21:57-04:00","message":"query `SELECT COUNT(*) FROM migration_info;` with params `[]`\n"}
{"level":"debug","type":"SQL","time":"2022-03-22T07:21:57-04:00","message":"query `SELECT COUNT(*) FROM migration_info;` with params `[]` took 342.38µs\n"}
{"level":"debug","type":"SQL","time":"2022-03-22T07:21:57-04:00","message":"query `SELECT version FROM migration_info;` with params `[]`\n"}
{"level":"debug","type":"SQL","time":"2022-03-22T07:21:57-04:00","message":"query `SELECT version FROM migration_info;` with params `[]` took 133.918µs\n"}
{"level":"info","time":"2022-03-22T07:21:57-04:00","message":"Current DB version: 0"}
{"level":"info","time":"2022-03-22T07:21:57-04:00","message":"Maximum available version: 22"}
{"level":"info","time":"2022-03-22T07:21:57-04:00","message":"Closing connection to data storage"}
```

Migrate to latest schema:

```
$ ./insights-results-aggregator migration latest

...
...
...
{"level":"debug","type":"SQL","time":"2022-03-22T07:22:18-04:00","message":"query `UPDATE migration_info SET version=$1;` with params `[22]`\n"}
{"level":"debug","type":"SQL","time":"2022-03-22T07:22:18-04:00","message":"query `UPDATE migration_info SET version=$1;` with params `[22]` took 362.156µs\n"}
{"level":"info","time":"2022-03-22T07:22:18-04:00","message":"Database version is now 22"}
{"level":"info","time":"2022-03-22T07:22:18-04:00","message":"Closing connection to data storage"}
```

Check the migration again:

```
$ ./insights-results-aggregator migration

Clowder is not enabled, skipping init...
Clowder is disabled
{"level":"info","time":"2022-03-22T07:22:32-04:00","message":"Making connection to data storage, driver=postgresWithHooks datasource=postgresql://postgres:postgres@localhost:5432/aggregator?sslmode=disable"}
{"level":"debug","type":"SQL","time":"2022-03-22T07:22:32-04:00","message":"query `CREATE TABLE IF NOT EXISTS migration_info (version INTEGER NOT NULL);` with params `[]`\n"}
{"level":"debug","type":"SQL","time":"2022-03-22T07:22:32-04:00","message":"query `CREATE TABLE IF NOT EXISTS migration_info (version INTEGER NOT NULL);` with params `[]` took 337.499µs\n"}
{"level":"debug","type":"SQL","time":"2022-03-22T07:22:32-04:00","message":"query `INSERT INTO migration_info (version) SELECT 0 WHERE NOT EXISTS (SELECT version FROM migration_info);` with params `[]`\n"}
{"level":"debug","type":"SQL","time":"2022-03-22T07:22:32-04:00","message":"query `INSERT INTO migration_info (version) SELECT 0 WHERE NOT EXISTS (SELECT version FROM migration_info);` with params `[]` took 478.954µs\n"}
{"level":"debug","type":"SQL","time":"2022-03-22T07:22:32-04:00","message":"query `SELECT COUNT(*) FROM migration_info;` with params `[]`\n"}
{"level":"debug","type":"SQL","time":"2022-03-22T07:22:32-04:00","message":"query `SELECT COUNT(*) FROM migration_info;` with params `[]` took 276.456µs\n"}
{"level":"debug","type":"SQL","time":"2022-03-22T07:22:32-04:00","message":"query `SELECT COUNT(*) FROM migration_info;` with params `[]`\n"}
{"level":"debug","type":"SQL","time":"2022-03-22T07:22:32-04:00","message":"query `SELECT COUNT(*) FROM migration_info;` with params `[]` took 148.287µs\n"}
{"level":"debug","type":"SQL","time":"2022-03-22T07:22:32-04:00","message":"query `SELECT version FROM migration_info;` with params `[]`\n"}
{"level":"debug","type":"SQL","time":"2022-03-22T07:22:32-04:00","message":"query `SELECT version FROM migration_info;` with params `[]` took 135.369µs\n"}
{"level":"info","time":"2022-03-22T07:22:32-04:00","message":"Current DB version: 22"}
{"level":"info","time":"2022-03-22T07:22:32-04:00","message":"Maximum available version: 22"}
{"level":"info","time":"2022-03-22T07:22:32-04:00","message":"Closing connection to data storage"}
```

Check the DB content (just for sure):

```
$ psql -U postgres -h localhost

Password for user postgres:
psql (13.4)
Type "help" for help.



postgres=# \c aggregator
You are now connected to database "aggregator" as user "postgres".
aggregator=# \dt
                       List of relations
 Schema |                Name                | Type  |  Owner
--------+------------------------------------+-------+----------
 public | advisor_ratings                    | table | postgres
 public | cluster_rule_toggle                | table | postgres
 public | cluster_rule_user_feedback         | table | postgres
 public | cluster_user_rule_disable_feedback | table | postgres
 public | consumer_error                     | table | postgres
 public | migration_info                     | table | postgres
 public | recommendation                     | table | postgres
 public | report                             | table | postgres
 public | rule_disable                       | table | postgres
 public | rule_hit                           | table | postgres
(10 rows)
```


### Run DB writer

Last step is to run the DB writer:

```
$ ./insights-results-aggregator 

Clowder is not enabled, skipping init...
Clowder is disabled
{"level":"info","type":"init","time":"2022-03-24T07:59:27-04:00","message":"Version: 0.5"}
{"level":"info","type":"init","time":"2022-03-24T07:59:27-04:00","message":"Build time: Tue Mar 22 05:29:03 AM EDT 2022"}
{"level":"info","type":"init","time":"2022-03-24T07:59:27-04:00","message":"Branch: master"}
{"level":"info","type":"init","time":"2022-03-24T07:59:27-04:00","message":"Commit: 35270ba93d9a9b91bfbe74b5e74764e8d57b5294"}
{"level":"info","type":"init","time":"2022-03-24T07:59:27-04:00","message":"Utils version:v1.23.3"}
{"level":"info","time":"2022-03-24T07:59:27-04:00","message":"Making connection to data storage, driver=postgresWithHooks datasource=postgresql://postgres:postgres@localhost:5432/aggregator?sslmode=disable"}
{"level":"debug","type":"SQL","time":"2022-03-24T07:59:27-04:00","message":"query `SELECT COUNT(*) FROM migration_info;` with params `[]`\n"}
{"level":"debug","type":"SQL","time":"2022-03-24T07:59:27-04:00","message":"query `SELECT COUNT(*) FROM migration_info;` with params `[]` took 1.484884ms\n"}
{"level":"debug","type":"SQL","time":"2022-03-24T07:59:27-04:00","message":"query `SELECT version FROM migration_info;` with params `[]`\n"}
{"level":"debug","type":"SQL","time":"2022-03-24T07:59:27-04:00","message":"query `SELECT version FROM migration_info;` with params `[]` took 215.947µs\n"}
{"level":"debug","type":"SQL","time":"2022-03-24T07:59:27-04:00","message":"query `SELECT cluster, last_checked_at FROM report;` with params `[]`\n"}
{"level":"debug","type":"SQL","time":"2022-03-24T07:59:27-04:00","message":"query `SELECT cluster, last_checked_at FROM report;` with params `[]` took 789.466µs\n"}
{"level":"info","time":"2022-03-24T07:59:27-04:00","message":"PrintRuleToggles start"}
{"level":"debug","type":"SQL","time":"2022-03-24T07:59:27-04:00","message":"query `\n\tSELECT\n\t\trule_id,\n\t\tcount(*)\n\tFROM\n\t\tcluster_rule_toggle\n\tGROUP BY\n\t\trule_id\n\t` with params `[]`\n"}
{"level":"debug","type":"SQL","time":"2022-03-24T07:59:27-04:00","message":"query `\n\tSELECT\n\t\trule_id,\n\t\tcount(*)\n\tFROM\n\t\tcluster_rule_toggle\n\tGROUP BY\n\t\trule_id\n\t` with params `[]` took 1.13213ms\n"}
{"level":"info","time":"2022-03-24T07:59:27-04:00","message":"PrintRuleDisableFeedbacks start"}
{"level":"debug","type":"SQL","time":"2022-03-24T07:59:27-04:00","message":"query `\n\tSELECT\n\t\trule_id,\n\t\tcount(*)\n\tFROM\n\t\tcluster_user_rule_disable_feedback\n\tGROUP BY\n\t\trule_id\n\t` with params `[]`\n"}
{"level":"debug","type":"SQL","time":"2022-03-24T07:59:27-04:00","message":"query `\n\tSELECT\n\t\trule_id,\n\t\tcount(*)\n\tFROM\n\t\tcluster_user_rule_disable_feedback\n\tGROUP BY\n\t\trule_id\n\t` with params `[]` took 557.768µs\n"}
{"level":"info","time":"2022-03-24T07:59:27-04:00","message":"Closing connection to data storage"}
{"level":"info","time":"2022-03-24T07:59:27-04:00","message":"Making connection to data storage, driver=postgresWithHooks datasource=postgresql://postgres:postgres@localhost:5432/aggregator?sslmode=disable"}
{"level":"info","time":"2022-03-24T07:59:27-04:00","message":"Making connection to data storage, driver=postgresWithHooks datasource=postgresql://postgres:postgres@localhost:5432/aggregator?sslmode=disable"}
{"level":"info","package":"sarama","time":"2022-03-24T07:59:27-04:00","message":"Initializing new client"}
{"level":"info","package":"sarama","time":"2022-03-24T07:59:27-04:00","message":"ClientID is the default of 'sarama', you should consider setting it to something application-specific."}
{"level":"info","package":"sarama","time":"2022-03-24T07:59:27-04:00","message":"ClientID is the default of 'sarama', you should consider setting it to something application-specific."}
{"level":"info","package":"sarama","time":"2022-03-24T07:59:27-04:00","message":"client/metadata fetching metadata for all topics from broker localhost:9092\n"}
{"level":"debug","type":"SQL","time":"2022-03-24T07:59:27-04:00","message":"query `SELECT COUNT(*) FROM migration_info;` with params `[]`\n"}
{"level":"info","package":"sarama","time":"2022-03-24T07:59:27-04:00","message":"Connected to broker at localhost:9092 (unregistered)\n"}
{"level":"debug","type":"SQL","time":"2022-03-24T07:59:27-04:00","message":"query `SELECT COUNT(*) FROM migration_info;` with params `[]` took 3.053161ms\n"}
{"level":"debug","type":"SQL","time":"2022-03-24T07:59:27-04:00","message":"query `SELECT version FROM migration_info;` with params `[]`\n"}
{"level":"debug","type":"SQL","time":"2022-03-24T07:59:27-04:00","message":"query `SELECT version FROM migration_info;` with params `[]` took 177.899µs\n"}
{"level":"info","time":"2022-03-24T07:59:27-04:00","message":"Starting HTTP server at ':8080'"}
{"level":"info","time":"2022-03-24T07:59:27-04:00","message":"Initializing HTTP server at ':8080'"}
{"level":"info","package":"sarama","time":"2022-03-24T07:59:27-04:00","message":"client/brokers registered new broker #0 at hpe-dl380pgen8-02-vm-15.hpe2.lab.eng.bos.redhat.com:9092"}
{"level":"info","package":"sarama","time":"2022-03-24T07:59:27-04:00","message":"Successfully initialized new client"}
{"level":"info","package":"sarama","time":"2022-03-24T07:59:27-04:00","message":"Initializing new client"}
{"level":"info","package":"sarama","time":"2022-03-24T07:59:27-04:00","message":"ClientID is the default of 'sarama', you should consider setting it to something application-specific."}
{"level":"info","package":"sarama","time":"2022-03-24T07:59:27-04:00","message":"ClientID is the default of 'sarama', you should consider setting it to something application-specific."}
{"level":"info","package":"sarama","time":"2022-03-24T07:59:27-04:00","message":"client/metadata fetching metadata for all topics from broker localhost:9092\n"}
{"level":"info","package":"sarama","time":"2022-03-24T07:59:27-04:00","message":"Connected to broker at localhost:9092 (unregistered)\n"}
{"level":"info","package":"sarama","time":"2022-03-24T07:59:27-04:00","message":"client/brokers registered new broker #0 at hpe-dl380pgen8-02-vm-15.hpe2.lab.eng.bos.redhat.com:9092"}
{"level":"info","package":"sarama","time":"2022-03-24T07:59:27-04:00","message":"Successfully initialized new client"}
{"level":"info","package":"sarama","time":"2022-03-24T07:59:27-04:00","message":"Initializing new client"}
{"level":"info","package":"sarama","time":"2022-03-24T07:59:27-04:00","message":"ClientID is the default of 'sarama', you should consider setting it to something application-specific."}
{"level":"info","package":"sarama","time":"2022-03-24T07:59:27-04:00","message":"ClientID is the default of 'sarama', you should consider setting it to something application-specific."}
{"level":"info","package":"sarama","time":"2022-03-24T07:59:27-04:00","message":"client/metadata fetching metadata for all topics from broker localhost:9092\n"}
{"level":"info","package":"sarama","time":"2022-03-24T07:59:27-04:00","message":"Connected to broker at localhost:9092 (unregistered)\n"}
{"level":"info","package":"sarama","time":"2022-03-24T07:59:27-04:00","message":"client/brokers registered new broker #0 at hpe-dl380pgen8-02-vm-15.hpe2.lab.eng.bos.redhat.com:9092"}
{"level":"info","package":"sarama","time":"2022-03-24T07:59:27-04:00","message":"Successfully initialized new client"}
{"level":"info","time":"2022-03-24T07:59:27-04:00","message":"waiting for consumer to become ready"}
{"level":"info","package":"sarama","time":"2022-03-24T07:59:27-04:00","message":"client/metadata fetching metadata for [ccx.ocp.results] from broker localhost:9092\n"}
{"level":"info","package":"sarama","time":"2022-03-24T07:59:27-04:00","message":"client/coordinator requesting coordinator for consumergroup aggregator from localhost:9092\n"}
{"level":"info","package":"sarama","time":"2022-03-24T07:59:27-04:00","message":"client/coordinator coordinator for consumergroup aggregator is #0 (hpe-dl380pgen8-02-vm-15.hpe2.lab.eng.bos.redhat.com:9092)\n"}
{"level":"info","package":"sarama","time":"2022-03-24T07:59:27-04:00","message":"ClientID is the default of 'sarama', you should consider setting it to something application-specific."}
{"level":"info","package":"sarama","time":"2022-03-24T07:59:27-04:00","message":"Connected to broker at hpe-dl380pgen8-02-vm-15.hpe2.lab.eng.bos.redhat.com:9092 (registered as #0)\n"}
{"level":"info","package":"sarama","time":"2022-03-24T07:59:27-04:00","message":"client/coordinator requesting coordinator for consumergroup aggregator from localhost:9092\n"}
{"level":"info","package":"sarama","time":"2022-03-24T07:59:27-04:00","message":"client/coordinator coordinator for consumergroup aggregator is #0 (hpe-dl380pgen8-02-vm-15.hpe2.lab.eng.bos.redhat.com:9092)\n"}
{"level":"info","time":"2022-03-24T07:59:27-04:00","message":"new session has been setup"}
{"level":"info","time":"2022-03-24T07:59:27-04:00","message":"finished waiting for consumer to become ready"}
{"level":"info","time":"2022-03-24T07:59:27-04:00","message":"started serving consumer"}
{"level":"info","package":"sarama","time":"2022-03-24T07:59:27-04:00","message":"consumer/broker/0 added subscription to ccx.ocp.results/0\n"}
{"level":"info","offset":18,"time":"2022-03-24T07:59:27-04:00","message":"starting messages loop"}
{"level":"debug","type":"SQL","time":"2022-03-24T07:59:27-04:00","message":"query `SELECT COALESCE(MAX(kafka_offset), 0) FROM report;` with params `[]`\n"}
{"level":"debug","type":"SQL","time":"2022-03-24T07:59:27-04:00","message":"query `SELECT COALESCE(MAX(kafka_offset), 0) FROM report;` with params `[]` took 1.635786ms\n"}
```



## Conclusion

Now all services should be configured correctly and it is possible to run
benchmarks and performance tests against any of such service (especially by
producing data to Kafka topic consumed by Insights Aggregator DB Writer).
