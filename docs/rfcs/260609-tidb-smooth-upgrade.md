# TiDB Smooth Upgrade

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: switch-controlled TiDB upgrade](#story-1-switch-controlled-tidb-upgrade)
    - [Story 2: unsupported smooth-upgrade version pair](#story-2-unsupported-smooth-upgrade-version-pair)
    - [Story 3: operator restart during upgrade](#story-3-operator-restart-during-upgrade)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Current v2 Upgrade Flow](#current-v2-upgrade-flow)
  - [TiDB HTTP API](#tidb-http-api)
  - [Version Matrix](#version-matrix)
    - [Unsupported](#unsupported)
    - [Auto-supported, no HTTP switch needed](#auto-supported-no-http-switch-needed)
    - [HTTP switch-controlled](#http-switch-controlled)
  - [Detecting TiDB Version Upgrades](#detecting-tidb-version-upgrades)
  - [Persisted Controller State](#persisted-controller-state)
  - [Start Gate](#start-gate)
  - [Finish Gate](#finish-gate)
  - [API Client Changes](#api-client-changes)
  - [Events and Conditions](#events-and-conditions)
  - [Test Plan](#test-plan)
    - [Unit Tests](#unit-tests)
    - [E2E Tests](#e2e-tests)
  - [Feature Gate](#feature-gate)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Store active state only in status conditions](#store-active-state-only-in-status-conditions)
  - [Call TiDB smooth-upgrade APIs for every TiDB version change](#call-tidb-smooth-upgrade-apis-for-every-tidb-version-change)
  - [Implement the gate in each TiDB instance controller](#implement-the-gate-in-each-tidb-instance-controller)
  - [Use Upgrade Show as the reconciliation source of truth](#use-upgrade-show-as-the-reconciliation-source-of-truth)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a release*.

- [ ] (R) This design doc has been discussed and approved
- [ ] (R) Test plan is in place
  - [ ] (R) e2e tests in kind
- [ ] (R) Graduation criteria is in place if required
- [ ] (R) User-facing documentation has been created in [pingcap/docs-tidb-operator]

## Summary

TiDB Operator v2 upgrades TiDB by reconciling a `TiDBGroup` template version into one `TiDB` instance CR per TiDB server. The existing updater coordinates Kubernetes-level rolling changes, but it does not coordinate TiDB smooth-upgrade mode. Users still need to avoid user DDL manually during some TiDB binary upgrade windows.

This RFC adds smooth-upgrade orchestration for TiDB Operator v2. For TiDB version pairs that require the TiDB HTTP switch, the `TiDBGroup` controller calls `POST /upgrade/start` before it creates, updates, or deletes any `TiDB` instance as part of the version rollout. After every managed TiDB instance reaches the target revision, target version, and Ready state, the controller calls `POST /upgrade/finish`. The active pause window is recorded in controller-owned `TiDBGroup` annotations so the controller can recover after restarts and always finish a pause window that it started.

## Motivation

TiDB smooth upgrade reduces DDL-related risk during TiDB binary upgrades. The TiDB HTTP API flips the cluster upgrade state. TiDB DDL then pauses non-system user DDL while allowing internal/system DDL, and resumes paused user jobs after finish. TiUP already uses this mechanism during `tiup cluster upgrade`; TiDB Operator should provide the same automated behavior for Kubernetes-managed TiDB clusters.

### Goals

- Pause user DDL automatically during eligible TiDB Server version upgrades managed by `TiDBGroup`.
- Call TiDB's bodyless `POST /upgrade/start` before the v2 updater performs the first version rollout action.
- Call `POST /upgrade/finish` after all `TiDB` instances in the group are updated and ready.
- Apply only to TiDB version upgrades, not scale-only, config-only, resource-only, metadata-only, suspend, adoption, or restart-only changes.
- Persist active pause state on `TiDBGroup` so operator restart does not lose the obligation to call `/upgrade/finish`.
- Preserve the existing rollout behavior for unsupported and no-switch-needed version pairs.
- Keep this behavior internal to the controller with no new required user-facing CRD field.

### Non-Goals

- Pausing DDL for PD, TiKV, TiFlash, TiCDC, TiProxy, TSO, Scheduling, ResourceManager, DM, Router, or BR components.
- Preventing every risky user operation described in TiDB smooth-upgrade limitations. TiDB enforces its own smooth-upgrade semantics after `/upgrade/start`.
- Changing v2's component upgrade ordering policy.
- Supporting downgrades with TiDB smooth-upgrade mode.
- Requiring `POST /upgrade/show` as a normal reconcile gate.

## Proposal

Add a smooth-upgrade gate to the `TiDBGroup` controller's updater task:

1. Detect whether the current reconciliation is a TiDB version upgrade.
2. Classify the source and target version pair.
3. For switch-controlled pairs, call `/upgrade/start` and persist an active pause annotation before the updater mutates any `TiDB` instance for the rollout.
4. Let the existing updater perform the rolling change.
5. On later reconciliations, when all managed `TiDB` instances are updated and ready, call `/upgrade/finish` and clear the annotations.

### User Stories

#### Story 1: switch-controlled TiDB upgrade

A user runs a `TiDBGroup` at `v7.5.0` and updates `spec.template.spec.version` to `v7.5.3`. The controller detects an eligible switch-controlled version upgrade. Before the updater changes any `TiDB` instance CR, the controller chooses a healthy TiDB endpoint and sends:

```bash
curl -X POST http://{TiDB}:10080/upgrade/start
```

After TiDB returns HTTP 200, the controller records smooth-upgrade annotations on the `TiDBGroup` and proceeds with the existing rolling update. When every managed `TiDB` instance is on the target revision/version and Ready, the controller sends:

```bash
curl -X POST http://{TiDB}:10080/upgrade/finish
```

After finish succeeds, the controller removes the annotations.

#### Story 2: unsupported smooth-upgrade version pair

A user upgrades from `v6.5.10` to `v8.1.0`. The version matrix marks this pair unsupported. The controller does not call `/upgrade/start` or `/upgrade/finish`, emits a warning event, and keeps the existing updater behavior.

#### Story 3: operator restart during upgrade

The controller calls `/upgrade/start`, persists annotations, and then restarts while TiDB instances are still rolling. When reconciliation resumes, the controller reads `core.pingcap.com/smooth-upgrade-ddl-paused: "true"` from the `TiDBGroup`, skips duplicate start, waits for rollout completion, calls `/upgrade/finish`, and clears the annotations.

### Risks and Mitigations

- **TiDB returns handler/session errors as HTTP 400**: keep HTTP status and response body in returned errors. Start failure must return retry and must not allow the updater to mutate any versioned `TiDB` instance.
- **`/upgrade/start` can take about 10 seconds**: use a 30-second client timeout for start. TiDB waits for the DDL owner to sync upgrading state before returning success, so a 200 start response is the safe point to begin rollout.
- **Operator crashes after `/upgrade/start` but before annotation persistence**: retrying start is acceptable because TiDB returns a successful duplicate-upgrading response. The controller should call start before patching annotations, then patch annotations before running the updater.
- **Annotation update conflicts**: patch only the smooth-upgrade annotation keys with retry-on-conflict semantics. Do not rewrite user annotations.
- **Manual annotation removal**: document the annotations as controller-owned. If annotations are missing, the controller cannot infer a previous successful start from Kubernetes state alone; users should not edit these keys.
- **Stale or mismatched annotations**: if active annotations refer to a different source/target pair, call `/upgrade/finish`, clear the annotations only after finish succeeds, emit a warning event, and requeue before starting the current upgrade.
- **Stalled rollout holds DDL pause**: keep annotations and do not call finish until the group is healthy and up to date. Users must fix the underlying pod/image/config issue to resume DDL safely.

## Design Details

### Current v2 Upgrade Flow

`pkg/controllers/tidbgroup/tasks/TaskUpdater` is the natural boundary for smooth-upgrade start:

- It checks whether a version upgrade is needed by comparing `dbg.Spec.Template.Spec.Version` with `dbg.Status.Version`.
- It blocks TiDB upgrades until the configured cluster upgrade policy allows TiDB to upgrade after dependent components.
- It waits for ready-but-not-available instances to satisfy `minReadySeconds`.
- It computes the desired group revision.
- It calls `pkg/updater` to create, update, or delete `TiDB` instance CRs according to surge, unavailable, topology, adoption, and restart rules.

The finish gate should also live in the `TiDBGroup` controller because only the group controller has the complete group-level view of desired replicas, revisions, and all managed `TiDB` instances. The instance controller should continue to own per-instance health, pod reconciliation, and TiDB status API calls used for ordinary readiness.

### TiDB HTTP API

TiDB exposes classic smooth-upgrade endpoints:

```text
POST /upgrade/start
POST /upgrade/finish
POST /upgrade/show
```

This feature uses only:

```text
POST /upgrade/start
POST /upgrade/finish
```

The calls are bodyless. TiDB returns HTTP 400 for non-POST requests and handler/session errors. Successful and idempotent responses are HTTP 200 with JSON string bodies such as:

```text
"success!"
"It's a duplicated operation and the cluster is already in upgrading state."
"It's a duplicated operation and the cluster is already in normal state."
```

The v2 client must decode JSON string bodies before matching responses. It should treat success and duplicate-success bodies as success, and return errors containing HTTP status, response body, operation, and URL for non-200 or unexpected responses.

### Version Matrix

The controller classifies TiDB version upgrades into three categories. Any pair where `target <= source` is unsupported and must never call smooth-upgrade APIs.

#### Unsupported

- Source `< v7.1.0` to any target.
- Source `v7.1.0`, `v7.1.1`, `v7.2.0`, or `v7.3.0` to target `>= v7.4.0`.
- Any version pair that cannot be parsed as semantic TiDB versions.
- Any downgrade or same-version pair.
- Any pair not matched by the auto-supported or HTTP switch-controlled categories below.

For unsupported pairs, do not call `/upgrade/start` or `/upgrade/finish`. Keep existing rollout behavior and emit a warning event.

#### Auto-supported, no HTTP switch needed

- `v7.1.0` to `v7.1.1`, `v7.2.0`, or `v7.3.0`.
- `v7.1.1` to `v7.2.0` or `v7.3.0`.
- `v7.2.0` to `v7.3.0`.

For these pairs, do not call `/upgrade/start` or `/upgrade/finish`; smooth upgrade is automatic in TiDB.

#### HTTP switch-controlled

- Source in `[v7.1.2, v7.2.0)` to target in `[v7.1.2, v7.2.0)`.
- Source in `[v7.1.2, v7.2.0)` or `>= v7.4.0` to target `>= v7.4.0`.

Only these pairs require TiDB Operator v2 to call `/upgrade/start` and `/upgrade/finish`.

### Detecting TiDB Version Upgrades

In v2, the desired target is `dbg.Spec.Template.Spec.Version`. The source version should come from observed group state, not from the desired spec after the user edits it.

Use this order:

1. If `dbg.Status.Version` is non-empty, use it as the source version.
2. If status version is empty, derive source candidates from managed `TiDB` instances that do not have the update revision and use their `spec.version` only when all outdated candidates agree.
3. If the source cannot be determined or the versions cannot be parsed, classify the pair as unsupported for smooth-upgrade API calls.

This logic must return "not a version upgrade" when the version is unchanged even if the group revision changes for config, resource, label, annotation, feature, or scheduling changes.

### Persisted Controller State

Use controller-owned annotations on the `TiDBGroup` object:

```text
core.pingcap.com/smooth-upgrade-ddl-paused: "true"
core.pingcap.com/smooth-upgrade-source-version: "<source>"
core.pingcap.com/smooth-upgrade-target-version: "<target>"
core.pingcap.com/smooth-upgrade-started-at: "<RFC3339 timestamp>"
```

Rules:

1. Set annotations only after `/upgrade/start` succeeds.
2. Treat `smooth-upgrade-ddl-paused=true` as meaning `/upgrade/start` has succeeded and `/upgrade/finish` is still owed.
3. Do not call `/upgrade/start` again while active annotations match the current source/target pair.
4. Remove all smooth-upgrade annotations only after `/upgrade/finish` succeeds.
5. If active annotations survive operator restart, continue waiting for rollout completion and then call finish.
6. If active annotations conflict with the current source/target pair, call `/upgrade/finish`, clear annotations after finish succeeds, emit a warning, and requeue.
7. Do not store this state in `TiDBGroup.spec.template.metadata.annotations`; it must not be propagated to `TiDB` instances.

Status conditions may be added for visibility, but annotations are the durable workflow state.

### Start Gate

Add `ensureSmoothUpgradeStarted(ctx, state, client)` before `updater.New(...).Build().Do(ctx)` in `TaskUpdater`.

The function should:

1. Return immediately when the group does not need a version upgrade.
2. Determine `source` and `target`.
3. Classify the version pair.
4. For auto-supported pairs, proceed without HTTP calls.
5. For unsupported pairs, emit `SmoothUpgradeUnsupported` and proceed without HTTP calls.
6. If active annotations match the current source/target pair, proceed without duplicate start.
7. If active annotations are stale or mismatched, call finish, clear annotations, and return retry so the next reconcile starts from clean state.
8. Select one healthy `TiDB` instance endpoint.
9. Call `POST /upgrade/start` with a 30-second timeout.
10. If start fails or times out, return retry/fail before invoking the updater.
11. If start succeeds, patch the four annotations and continue to the updater.

Healthy endpoint selection should prefer deterministic ordering, for example by sorted `TiDB` name. It must use only instances that are Ready and have a reachable status endpoint.

### Finish Gate

Add `maybeFinishSmoothUpgrade(ctx, state, client)` after the updater and before final status persistence in the `TiDBGroup` runner. It can also be called inside `TaskUpdater` after a no-op updater result, but it must run only after the state has enough information to prove group completion.

The function should:

1. Return immediately when the pause annotation is not active.
2. Verify the group has no pending version rollout:
   - `dbg.Status.ObservedGeneration == dbg.Generation`
   - `dbg.Status.Version == target`
   - desired replicas equal status replicas
   - ready replicas, updated replicas, and current replicas all equal desired replicas
   - update revision equals current revision
   - every managed `TiDB` instance is Ready
   - every managed `TiDB` instance has `spec.version == target`
3. If the group is not complete, return wait/retry and keep annotations.
4. Select one healthy `TiDB` endpoint.
5. Call `POST /upgrade/finish`.
6. If finish fails, return retry/fail and keep annotations.
7. If finish succeeds, delete all smooth-upgrade annotations.

The finish gate should tolerate TiDB's duplicate-normal response as success. This is required for recovery from client timeouts and stale annotations.

### API Client Changes

Extend `pkg/tidbapi/v1.TiDBClient`:

```go
type TiDBClient interface {
    GetHealth(ctx context.Context) (bool, error)
    GetInfo(ctx context.Context) (*ServerInfo, error)
    SetServerLabels(ctx context.Context, labels map[string]string) error
    GetPoolStatus(ctx context.Context) (*PoolStatus, error)
    Activate(ctx context.Context, keyspace string) error
    StartUpgrade(ctx context.Context) error
    FinishUpgrade(ctx context.Context) error
}
```

Implementation notes:

- Add path constants for `upgrade/start` and `upgrade/finish`.
- Keep the call body empty.
- Decode TiDB's JSON string response before comparing with known success bodies.
- Use 30 seconds for `StartUpgrade`. This can be a dedicated client, a per-call timeout, or a small helper that preserves the same TLS transport behavior as `NewTiDBClient`.
- `FinishUpgrade` may use the normal TiDB request timeout, but accepting the same 30-second path is also acceptable.

The `TiDBGroup` controller needs a small helper to construct a `TiDBClient` for a selected `TiDB` instance, reusing the TLS setup already used by `pkg/controllers/tidb/tasks/TaskContextInfoFromPDAndTiDB`.

### Events and Conditions

Emit Kubernetes events on the `TiDBGroup`:

- `SmoothUpgradeStarted`
- `SmoothUpgradeFinished`
- `SmoothUpgradeStartFailed`
- `SmoothUpgradeFinishFailed`
- `SmoothUpgradeUnsupported`
- `SmoothUpgradeSkipped`
- `SmoothUpgradeRecovered`

Optionally add a `TiDBGroup` condition such as `SmoothUpgradePaused` for visibility. It must be derived from annotations and current reconciliation state, not used as the source of truth.

### Test Plan

#### Unit Tests

- `pkg/tidbapi/v1/client_test.go`
  - `StartUpgrade` sends bodyless `POST /upgrade/start`.
  - `StartUpgrade` accepts `"success!"` and duplicate-upgrading HTTP 200 responses.
  - `FinishUpgrade` sends bodyless `POST /upgrade/finish`.
  - `FinishUpgrade` accepts `"success!"` and duplicate-normal HTTP 200 responses.
  - HTTP 400, 5xx, client timeout, and unexpected response bodies return useful errors.
  - JSON string response bodies are decoded before matching.

- `pkg/controllers/tidbgroup/tasks/smooth_upgrade_test.go`
  - Classify every row in the version matrix.
  - Downgrades and same-version changes are unsupported.
  - Invalid versions are unsupported.
  - Config/resource/metadata-only revisions are not version upgrades.
  - Read, write, clear, and patch only the four smooth-upgrade annotations.
  - Active matching annotations skip duplicate start.
  - Stale or mismatched annotations call finish, clear annotations, and requeue.
  - Source version detection uses `status.version` first and falls back to outdated instances only when unambiguous.

- `pkg/controllers/tidbgroup/tasks/updater_test.go`
  - Switch-controlled version upgrade calls start before the updater mutates any `TiDB` instance.
  - Start failure blocks rollout.
  - Annotation patch failure blocks rollout and retries safely.
  - Auto-supported and unsupported pairs skip start.
  - Finish runs only after all group status and instance readiness checks pass.
  - Finish failure keeps annotations.
  - Finish success removes annotations.

#### E2E Tests

- Deploy a switch-controlled source version, for example `v7.5.0`, with at least two TiDB replicas.
- Upgrade to a supported target, for example `v7.5.3`.
- Verify smooth-upgrade annotations appear before rollout completes.
- Verify source and target annotation values.
- Verify annotations are removed after all TiDB instances are upgraded and ready.
- Restart the operator after start and before finish; verify finish still happens and annotations are cleared.
- Upgrade an unsupported pair, for example `v6.5.10` to the latest supported image; verify no smooth-upgrade annotations are set and rollout still completes.

### Feature Gate

No feature gate. This behavior is internal, applies only to supported switch-controlled TiDB version upgrades, and preserves existing behavior for every unsupported or unrelated rollout.

## Drawbacks

- The `TiDBGroup` controller must update metadata in addition to status and instance reconciliation, which adds conflict handling.
- A stuck TiDB rollout keeps user DDL paused until the rollout is fixed and finish succeeds.
- Unsupported pairs still rely on users avoiding DDL manually; the operator can only warn and preserve the previous behavior.
- Endpoint selection and TiDB API calls add another dependency to the group updater path.

## Alternatives

### Store active state only in status conditions

Status is observational and can be recalculated. After `/upgrade/start` succeeds, the controller owns a future `/upgrade/finish` obligation. Annotations on the main object are a better durable workflow marker.

### Call TiDB smooth-upgrade APIs for every TiDB version change

This is simpler but incorrect. Unsupported pairs could fail upgrades that currently work, and auto-supported pairs do not need HTTP switching.

### Implement the gate in each TiDB instance controller

Each instance controller sees only one `TiDB` instance and cannot reliably decide the group-level first rollout action or final group completion. The start and finish decisions belong in `TiDBGroup`.

### Use Upgrade Show as the reconciliation source of truth

`/upgrade/show` can help diagnostics, but live TiDB state alone does not record whether this operator started the pause window or which source/target pair it owns. Controller-owned annotations provide deterministic recovery. `/upgrade/show` can be added later as an optional defensive check.
