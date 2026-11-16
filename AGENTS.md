# TiDB Operator

Read [the development guide](docs/CONTRIBUTING.md) before changing code.

## Upgrade compatibility

An operator or CRD upgrade with unchanged user intent must preserve existing Pods.
Unintended workload restarts are release-blocking bugs. An intentional restart
requires an explicit requirement and a documented rollout plan.

For changes to API fields, CRD defaults, template comparison, revisions, or
configuration rendering:

- Trace old stored CRs through API-server defaulting, Group reconciliation,
  instance reconciliation, and existing Pod annotations. Annotation JSON does
  not receive CRD defaults.
- Compare effective behavior, not just serialized values. Normalize equivalent
  unset/default values on comparison copies; never mutate live desired state.
- Add regression tests using pre-upgrade objects and Pod templates. Verify that
  an upgrade preserves Pods, while real configuration changes still trigger
  their required update behavior. Observe the regression fail before fixing it.
- Check rendered configuration and revision changes as well as template equality.
  A metadata-only revision change must not accidentally recreate workloads.
- Inspect instance-level deletion and leader eviction paths. A Group's rollout
  budget does not automatically protect actions taken by instance controllers.
- Check recovery with eviction pending, Pod deletion in progress, and Pod
  recreation complete. Do not claim a deletion already issued can be cancelled.
- Report unit-test, API-server/upgrade-test, and live-cluster evidence separately.
  Passing unit tests alone does not establish a disruption-free upgrade.

## Code placement

Keep API types and schema declarations in `api/`. Put operator-side interpretation
and defaulting helpers in `pkg/apiutil/core/v1alpha1` (`coreutil`). Follow existing
caller nil-handling conventions and keep shared helpers behavior-preserving.
