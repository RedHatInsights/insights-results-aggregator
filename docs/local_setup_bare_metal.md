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



### Common to be used

First we need to install several packages that are not included in default
Fedora Server system:

```
# dnf install git
# dnf install wget
# dnf install tar
```


