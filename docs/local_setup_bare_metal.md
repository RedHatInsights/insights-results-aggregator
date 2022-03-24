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


