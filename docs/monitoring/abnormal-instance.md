# Instance condition alerts

`tidb_operator_abnormal_instance` exposes the last recorded `Ready` and `Synced`
conditions. Its labels are `namespace`, `cluster`, `component`, `group`, `instance`,
`condition`, and `reason`. The value is `1` for `False`, and `0` for `True`,
`Unknown`, or an absent condition. `reason` is copied verbatim from the condition;
an absent condition has an empty reason. Condition messages are not labels.

Observation does not read Cluster state or suppress reasons. Suspending or
suspended instances report `Ready=False` with `reason="Suspending"` or
`reason="Suspended"`, so their Ready metric is also `1`. Alert rules decide whether
these states should page an operator. Synced failures should remain visible even
while a cluster is suspended.

[abnormal-instance.rules.yaml](abnormal-instance.rules.yaml) is a Prometheus rule
file with two example alerts. It excludes the expected suspend reasons only for
Ready, and keeps all Synced failures. Adjust the example ten-minute duration and
severity for your environment, then install it in your Prometheus rule setup.
This repository does not automatically deploy these rules.

The examples use `max without (reason)` so a change between abnormal reasons does
not change the alert's label set and restart its `for` timer. Other labels remain
part of the alert identity. See the Prometheus documentation for
[alerting rules](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/)
and [aggregation operators](https://prometheus.io/docs/prometheus/latest/querying/operators/).

Only the current reason series is exported for each instance condition. A reason
change removes the old series, and instance finalization or a subsequent NotFound
reconcile removes all its reason series. Prometheus retains historical samples
according to its retention policy; removing an exported series does not erase
history. Reason changes can increase historical series cardinality, so reasons
should remain a bounded vocabulary rather than contain dynamic error messages.

When adding this label to an existing installation, update alerts alongside the
operator rollout. Existing unfiltered `metric == 1` rules will include expected
suspend states; dashboards and recording rules may also need to aggregate away
`reason`. A rolling operator upgrade can temporarily expose both old and new
label schemas; the example aggregation handles both, but old operator replicas
still report their previous semantics until upgraded.

If a resume reconcile fails before the normal Ready task runs, the last suspend
reason remains recorded. The metric faithfully exposes that stale condition, and
the example Ready alert still excludes it. Monitor reconciliation failures as
well; these rules do not guarantee detection of a resume failure before conditions
are refreshed.

Validate the examples and their suspend/resume and reason-change cases with:

```sh
promtool check rules docs/monitoring/abnormal-instance.rules.yaml
promtool test rules docs/monitoring/abnormal-instance.rules.test.yaml
```
