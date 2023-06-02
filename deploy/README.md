# Deployment

## Testing the local version of the cache-writer in ephemeral

1. Install `bonfire`
```
pip install crc-bonfire
```

2. Log into https://console-openshift-console.apps.c-rh-c-eph.8p0c.p1.openshiftapps.com/k8s/cluster/projects

```
oc login --token=${TOKEN} --server=https://api.c-rh-c-eph.8p0c.p1.openshiftapps.com:6443
```

3. Reserve a namespace
```
NAMESPACE=$(bonfire namespace reserve)
```

4. Deploy the cache-writer and Redis workloads
```
bonfire deploy -c deploy/test-cache-writer.yaml -n $NAMESPACE ccx-data-pipeline
```

5. Test that you can read and write from Redis

Spin up a container:

```
oc --namespace $NAMESPACE run test -i --rm \
    --image=quay.io/edge-infrastructure/redis:6.2.7-debian-10-r23 \
    sh
```

And run:
```
export REDISCLI_AUTH="rNOp(B^!Y1tRpGL50w_6rAv~"
redis-cli -h ccx-redis -p 6379 ping; echo $?
redis-cli -h ccx-redis -p 6379 SET mykey "Hello\nWorld";
redis-cli -h ccx-redis -p 6379 GET mykey;
```

You can also check that the metrics are exposed:
```
curl ccx-redis-metrics:9121/metrics
```

Don't worry if you can't see the command prompt. Just write and execute commands.
Then exit with CTRL+D.

6. TODO: Test the cache-writer

7. Delete the namespace
```
bonfire namespace release $NAMESPACE 
```
