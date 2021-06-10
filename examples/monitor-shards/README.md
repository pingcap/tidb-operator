# Shards and Replicas

If a single Prometheus can't hold the current target metrics, the user can shard the targets on multiple Prometheus servers.
Shards use the Prometheus `modulus` configuration, which takes the hash of the `__address__` source label values, and split the scrape targets based on the number of shards.

Note that scaling down shards will not reshard data onto the remaining instances, the data must be manually moved. Increasing shards will not reshard data neither but it will continue to be available from the same instances. 
To query globally, use Thanos sidecar and Thanos querier or write data to a remote central location.

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

We will see that the scrape targets are distributed on the two shards.

And we must use Thanos sidecar to query globally.





