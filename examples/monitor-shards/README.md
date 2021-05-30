# Shards and Replicas

If single prometheus can't hold current targets metrics,user can reshard targets on multiple prometheus servers.
Shards use prometheus `modulus` configuration to implement,which take of the hash of the source label values,split scrape targets based on the number of shards.

Prometheus operator will create  number of `shards` multiplied by `replicas` pods.

Note that scaling down shards will not reshard data onto remaining instances,it must be manually moved. Increasing shards will not reshard data either but it will continue to be available from the same instances. 
To query globally use Thanos sidecar and Thanos querier or remote write data to a central location. Sharding is done on the content of the `__address__` target meta-label.

## Install Example

Install TiDB:

```bash
kubectl apply -f tidb-cluster.yaml -n ${namespace}
```

Wait for Pods ready:

```bash
watch kubectl -n ${namespace} get pod
```

Install TidbMonitor with two shards :

```bash
kubectl apply -f tidb-monitor.yaml -n ${namespace}
```

Wait for Pods ready:

```bash
watch kubectl -n ${namespace} get pod
```

We will that the scrape targets are distributed on the two shards.

And we must use thanos sidecar to query globally.






