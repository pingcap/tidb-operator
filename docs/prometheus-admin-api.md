# Prometheus admin API

TidbMonitor no longer enables the Prometheus TSDB admin API by default. Normal
queries, scraping, alert evaluation, Grafana access and rule/configuration reload
remain available. The independent `--web.enable-lifecycle` option remains enabled
because the reloaders use it.

The admin API was originally enabled for exporting TSDB data (see
[PR #360](https://github.com/pingcap/tidb-operator/pull/360)). To retain this
capability, merge the following into the existing TidbMonitor configuration,
keeping any other `commandOptions` entries:

```yaml
spec:
  prometheus:
    config:
      commandOptions:
        - --web.enable-admin-api
```

This enables all TSDB admin endpoints, including snapshots and data deletion; it
does not add authentication. Restrict access to trusted administrators using your
network and authentication controls. A ClusterIP Service alone is not an access
control boundary, and port-forwarding does not prevent other clients from
accessing a reachable Service or Pod address. See the
[Prometheus admin API documentation](https://prometheus.io/docs/prometheus/latest/querying/api/#tsdb-admin-apis).

## Upgrade behavior

If existing monitoring-data export or backup scripts use the snapshot API,
complete the Operator/controller-manager upgrade first, then add the explicit
option above. Without the option, those calls stop working after the monitoring
Pods update. This change does not affect TiDB database backup/restore jobs.

**Do not add the option while an older Operator is still reconciling monitors.**
Older Operators already add the flag unconditionally, so this produces duplicate
flags; Prometheus v2.27.1 rejects them and cannot start. Waiting until all
controller-manager instances run the new version avoids this failure. The safe
upgrade-then-opt-in sequence can roll monitoring Pods twice and temporarily
interrupt TSDB export. To retain the opt-in on the first updated monitoring Pod,
coordinate the controller upgrade and CR change while monitor reconciliation is
paused, so an old controller never applies the explicit option. The monitoring
Pod still rolls as described below.

Before rolling the Operator back to an older version, remove the explicit admin
API option; otherwise the old controller generates duplicate flags again.

Changes to the generated Prometheus command, including this Operator upgrade
when the option is absent, roll the affected TidbMonitor StatefulSet Pods.
Changing `commandOptions` also changes the Pod template. All containers in each
updated monitoring Pod restart, and its initializers run again; TiDB, PD and TiKV
Pods are not restarted by this change. A single monitoring replica temporarily
stops collecting data, evaluating alerts and serving queries. Existing PVC data
is retained, but Prometheus may take time to replay its WAL. Ephemeral storage
does not provide data persistence across Pod replacement.
