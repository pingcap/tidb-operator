# Record Conditions in TidbCluster Status

## Motivation

We can record some important conditions in status struct. It’s convenient for
users to know the status of tidb clusters via `kubectl get tc`, e.g.

- Are all components up and healthy?
- If the tidb cluster is not ready, which problem do we have now?

## Proposal

A new condition to indicate the tidb cluster is ready or not

```go
// TidbClusterReady indicates that the tidb cluster is ready or not.
// This is defined as:
// - All statefulsets are up to date (currentRevision == updateRevision).
// - All PD members are healthy.
// - All TiDB pods are healthy.
// - All TiKV stores are up.
// - All TiFlash stores are up.
TidbClusterReady TidbClusterConditionType = "Ready"
```

Add "READY" and "MESSAGE" columns in `additionalPrinterColumns`:

```yaml
- JSONPath: .status.conditions[?(@.type=="Ready")].status
    name: Ready
    type: string
- JSONPath: .status.conditions[?(@.type=="Ready")].message
    name: Status
    priority: 1
    type: string
```

The status message is the detailed description of `Ready` condition. We set
`priority` to 1, then it's printed only if `-o wide` is provided.

## TODO

### Emit event on condition change

e.g. tidb cluster becomes unhealthy 

### More useful conditions

- Upgrading
- Scaling
