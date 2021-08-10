---
layout: page
nav_order: 13
---
# Local setup
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

There is a `docker-compose` configuration that provisions a minimal stack of Insight Platform and
a postgres database.
You can download it here <https://gitlab.cee.redhat.com/insights-qe/iqe-ccx-plugin/blob/master/docker-compose.yml>

## Prerequisites

* edit localhost line in your `/etc/hosts`:  `127.0.0.1       localhost kafka minio`
* `ingress` image should present on your machine. You can build it locally from this repo
<https://github.com/RedHatInsights/insights-ingress-go>
* `ccx-data-pipeline` installed. You can doing it by cloning the [repository](https://github.com/RedHatInsights/ccx-data-pipeline) and running `pip install -r requirements.txt`. Make sure you are using the appropiate version of Python (look at `ccx-data-pipeline` README) and a virtualenv (optional but recommended).

## Usage

1. Start the stack `podman-compose up` or `docker-compose up`
2. Wait until kafka will be up.
3. Start `ccx-data-pipeline`: `python3 -m insights_messaging config-devel.yaml`
4. Build `insights-results-aggregator`: `make build`
5. Start `insights-results-aggregator`: `INSIGHTS_RESULTS_AGGREGATOR_CONFIG_FILE=config-devel.toml ./insights-results-aggregator`

Stop Minimal Insights Platform stack `podman-compose down` or `docker-compose down`

In order to upload an insights archive, you can use `curl`:

```shell
curl -k -vvvv -F "upload=@/path/to/your/archive.zip;type=application/vnd.redhat.testareno.archive+zip" http://localhost:3000/api/ingress/v1/upload -H "x-rh-identity: eyJpZGVudGl0eSI6IHsiYWNjb3VudF9udW1iZXIiOiAiMDAwMDAwMSIsICJpbnRlcm5hbCI6IHsib3JnX2lkIjogIjEifX19Cg=="
```

or you can use integration tests suite. More details are [here](https://gitlab.cee.redhat.com/insights-qe/iqe-ccx-plugin).

## Troubleshooting

* If you find an error in the last step because no `migration_info` table was found in the DB, run
```
INSIGHTS_RESULTS_AGGREGATOR_CONFIG_FILE=config-devel.toml ./insights-results-aggregator migrate latest
```

* If the binary cannot find the `config-devel.toml` even if the relative path is right, use an absolute one instead i.e `/path/to/config-devel.toml`.

* Make sure to read the contents of `config-devel.toml` as it has all the configuration about the connections to the containers and the API prefix.


## Kafka producer

It is possible to use the script `produce_insights_results` from `utils` to produce several Insights
results into Kafka topic. Its dependency is Kafkacat that needs to be installed on the same machine.
You can find installation instructions [on this page](https://github.com/edenhill/kafkacat).
