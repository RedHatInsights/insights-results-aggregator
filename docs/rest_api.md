---
layout: page
nav_order: 6
---
# REST API

Aggregator service provides information about its REST API schema via endpoint `api/v1/openapi.json`.
OpenAPI 3.0 is used to describe the schema; it can be read by human and consumed by computers.

For example, if aggregator is started locally, it is possible to read schema based on OpenAPI 3.0
specification by using the following command:

```shell
curl localhost:8080/api/v1/openapi.json
```

Please note that OpenAPI schema is accessible w/o the need to provide authorization tokens.

## Accessing results

### Settings for localhost

```
export ADDRESS=localhost:8080/api/v1
```

### Basic endpoints

#### List of clusters associated with the specified organization ID

```
/organizations/{orgId}/clusters
```

##### Usage:

```
curl -k -v $ADDRESS/organizations/{orgId}/clusters
```

#### Report for the given organization and cluster

```
/organizations/{orgId}/clusters/{clusterId}/users/{userId}/report
```

##### Usage:

```
curl -k -v $ADDRESS/organizations/{orgId}/clusters/{clusterId}/users/{userId}/report
```

#### Latest reports for the given list of clusters

##### Using `GET` method

```
/organizations/{orgId}/clusters/{clusterList}/reports
```

##### Usage:

```
curl -k -v $ADDRESS/organizations/{orgId}/clusters/{cluster1}/report
curl -k -v $ADDRESS/organizations/{orgId}/clusters/{cluster1},{cluster2}/report
curl -k -v $ADDRESS/organizations/{orgId}/clusters/{cluster1},{cluster2},{cluster3}/report
```

Plase note that the total URL length is limited, usually to 2000 or 2048 characters.

##### Using `POST` method

```
/organizations/{orgId}/clusters/reports
```

##### Usage:

```
curl -k -v $ADDRESS/organizations/{orgId}/clusters/reports -d @cluster_list.json
```

##### Format of the payload:

```json
{
        "clusters" : [
                "34c3ecc5-624a-49a5-bab8-4fdc5e51a266",
                "74ae54aa-6577-4e80-85e7-697cb646ff37",
                "a7467445-8d6a-43cc-b82c-7007664bdf69",
                "ee7d2bf4-8933-4a3a-8634-3328fe806e08"
        ]
}
```

#### Latest rule report for the given organization, cluster, user and rule ids

```
/organizations/{orgId}/clusters/{clusterId}/users/{userId}/rules/{ruleId}
```

#### List of clusters for a given rule selector (plugin_name|error_key) within the specified organization ID

```
/rules/{rule_selector}/organizations/{org_id}/users/{user_id}/clusters_detail
```

##### Usage:

```
curl -k -v $ADDRESS/rules/{rule_selector}/organizations/{org_id}/users/{user_id}/clusters_detail
```

Plase note that user ID is expected, but is only used for improving logging.
