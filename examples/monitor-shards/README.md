# Shards and Replicas

If a single Prometheus can't hold the current target metrics, the user can shard the targets on multiple Prometheus servers.
Shards use the Prometheus `modulus` configuration, which takes the hash of the `__address__` source label values, and splits the scrape targets based on the number of shards.

Note that decreasing shards will not reshard data onto the remaining instances, the data must be manually moved. Increasing shards will not reshard data either but it will continue to be available from the original instances. 

It’s not recommended to configure `spec.prometheus.ingress` and `spec.grafana` in the TidbMonitor CR when using multiple shards. And we recommend using Thanos to query the metrics globally.

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
