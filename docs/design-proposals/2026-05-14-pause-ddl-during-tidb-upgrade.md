# Pause user DDL during TiDB version upgrade

## Summary

TiDB Operator currently upgrades TiDB by reconciling `spec.version` into a Kubernetes StatefulSet rolling update. This replaces TiDB pods safely at the Kubernetes level, but it does not coordinate TiDB's smooth-upgrade mode, so users still need to avoid user-initiated DDL during the upgrade window.

This proposal adds TiDB smooth-upgrade orchestration for tidb-operator v1 classic TiDB clusters. For TiDB version pairs that require the HTTP switch, TiDB Operator will call `POST /upgrade/start` before the first TiDB pod is rolled and `POST /upgrade/finish` after every TiDB pod is upgraded and healthy. The active pause state is persisted in controller-owned `TidbCluster` annotations so the controller can recover after restarts and always finish a pause window that it started.

## Motivation

TiDB smooth upgrade is designed to reduce DDL-related risk during TiDB binary upgrades. The HTTP API flips TiDB global cluster upgrade state; the DDL subsystem reacts to that state by pausing non-system/user DDL while allowing system DB DDL, then resuming paused user jobs on finish. TiUP already uses this mechanism during `tiup cluster upgrade`, but TiDB Operator does not. In Kubernetes deployments, especially managed environments, users expect TiDB Operator to provide a comparable upgrade experience without requiring a separate manual script to pause and resume user DDL.

### Goals

- Pause user DDL automatically during eligible TiDB Server version upgrades managed by TiDB Operator.
- Call TiDB's bodyless HTTP API `POST /upgrade/start` before TiDB rolling upgrade starts.
- Call `POST /upgrade/finish` after all TiDB pods are upgraded and healthy.
- Apply only to TiDB version upgrades, not scale, config, resource, restart, failover, or volume operations.
- Apply only to tidb-operator v1 classic TiDB kernel clusters.
- Persist the active pause state so operator restart does not lose the obligation to call `/upgrade/finish`.
- Preserve existing rolling-upgrade behavior for unsupported or no-switch-needed version pairs.

### Non-Goals

- Supporting Premium/keyspace-specific behavior. tidb-operator v1 supports classic TiDB kernel only.
- Pausing DDL for TiKV, PD, TiFlash, TiCDC, TiProxy, Pump, TiCI, scale operations, or config-only rollouts.
- Preventing all risky user operations listed in TiDB smooth-upgrade limitations. TiDB enforces its own smooth-upgrade mode semantics after `/upgrade/start`.
- Introducing a new user-facing CRD field for this behavior.
- Using `POST /upgrade/show` as a required reconcile gate. It can be useful during rollout to confirm consistency or for diagnostics, but start/finish and persisted controller state are sufficient for reconciliation.

## Proposal

### User Stories

#### Story 1: supported switch-controlled TiDB upgrade

A user runs a TiDB cluster at `v7.4.x` and updates `spec.version` to a later supported version. TiDB Operator detects that this is a TiDB image version upgrade and that the source/target pair requires smooth-upgrade HTTP switching. Before updating the TiDB StatefulSet in a way that can replace a pod, the operator sends:

```bash
curl -X POST http://{TiDBIP}:10080/upgrade/start
```

TiDB returns:

```text
"success!"
```

The operator then persists controller-owned annotations on the `TidbCluster` and proceeds with the existing TiDB rolling upgrade. After all TiDB pods are upgraded, available, and healthy, the operator sends:

```bash
curl -X POST http://{TiDBIP}:10080/upgrade/finish
```

After finish succeeds, the operator removes the annotations.

#### Story 2: unsupported smooth-upgrade version pair

A user upgrades from `v7.3.0` to `v7.4.0`. TiDB's smooth-upgrade matrix marks this pair as unsupported. TiDB Operator does not call `/upgrade/start` or `/upgrade/finish`; it proceeds with the existing rolling-upgrade behavior and emits a warning event indicating that TiDB smooth upgrade is not available for the source/target pair.

#### Story 3: operator restart during upgrade

TiDB Operator calls `/upgrade/start` successfully and then restarts while TiDB pods are still rolling. When reconciliation resumes, the operator reads `tidb.pingcap.com/smooth-upgrade-ddl-paused: "true"` from the `TidbCluster`, skips duplicate start, waits for TiDB upgrade completion, calls `/upgrade/finish`, and removes the annotations after finish succeeds.

### Risks and Mitigations

- **TiDB handler/session errors return HTTP 400**: preserve response body/status in operator errors and requeue without advancing rollout on start failure. `/upgrade/start` can legitimately take almost 10 seconds while waiting for DDL owner sync, so use a 30-second timeout. Duplicate start/finish HTTP 200 responses are success and should not block recovery after client timeout or operator restart.
- **Source version is not available from `spec.version` after the user updates the CR**: derive the source version from the old TiDB StatefulSet image tag and the target version from the new TiDB StatefulSet image tag.
- **Operator crash after `/upgrade/start`**: persist the pause window in annotations immediately after start succeeds.
- **Annotation update conflict**: annotation updates modify the main `TidbCluster` object. Mitigation: use retry-on-conflict and keep annotation changes small and deterministic.
- **User manually removes controller annotations**: document these annotations as controller-owned. Emit warnings when metadata is missing or inconsistent.
- **Custom image tags cannot be parsed as TiDB versions**: treat them as unsupported for smooth-upgrade API calls and preserve existing rolling-upgrade behavior.

## Design Details

### Existing upgrade flow

The current TiDB flow already has a natural gate for smooth-upgrade start:

- TiDB member reconciliation builds a desired StatefulSet and calls the TiDB upgrader when the pod template differs or TiDB is already in `UpgradePhase`.
- The TiDB upgrader blocks TiDB upgrade while other components are upgrading/scaling or TiDB is scaling.
- The TiDB upgrader then advances StatefulSet rolling partitions one pod at a time.

The smooth-upgrade start call should happen after existing blockers pass and before the first StatefulSet update/partition change can replace a TiDB pod. Finish should happen after status sync proves the TiDB rolling upgrade has completed.

### TiDB HTTP API

The classic TiDB smooth-upgrade API is wired at `pkg/server/http_status.go` to `pkg/server/handler/upgrade_handler.go`:

```text
POST /upgrade/{op}
```

Supported operations include:

- `start`
- `finish`
- `show`

For this feature, TiDB Operator uses only:

```text
POST /upgrade/start
POST /upgrade/finish
```

The calls are bodyless. TiDB returns HTTP 400 for non-POST requests and for handler/session errors. Successful and idempotent responses are HTTP 200 with one of these bodies:

```text
"success!"
"It's a duplicated operation and the cluster is already in upgrading state."
"It's a duplicated operation and the cluster is already in normal state."
```

Implementation should treat these HTTP 200 bodies as success and return errors that include HTTP status and response body for non-200 or unexpected responses. A 200 response from `/upgrade/start` is the point where it is safe to start replacing TiDB pods: TiDB waits up to 10 seconds for the DDL owner to observe and sync the cluster into upgrading state before returning success. Set the client HTTP timeout above 10 seconds; use 30 seconds to allow network/proxy overhead.

The existing shared `http.Client` in `defaultTiDBControl` uses a 5-second timeout and must not be used for `StartUpgrade`. `StartUpgrade` must construct a dedicated `http.Client` with a 30-second timeout (or use a per-request `context.WithTimeout`) for this call only. `FinishUpgrade` can use the shared client since TiDB returns immediately for that operation.

### Version matrix

TiDB Operator should classify TiDB version upgrades into three categories.

#### Unsupported

- Source `< v7.1.0` to any target.
- Source `v7.1.0`, `v7.1.1`, `v7.2.0`, or `v7.3.0` to target `>= v7.4.0`.
- Any version pair that cannot be parsed as semantic TiDB versions.
- Any pair not matched by the auto-supported or HTTP switch-controlled categories below (catch-all default).

For unsupported pairs, do not call `/upgrade/start` or `/upgrade/finish`. Keep existing rolling-upgrade behavior and emit a warning event.

#### Auto-supported, no HTTP switch needed

- `v7.1.0` to `v7.1.1`, `v7.2.0`, or `v7.3.0`.
- `v7.1.1` to `v7.2.0` or `v7.3.0`.
- `v7.2.0` to `v7.3.0`.

For these pairs, do not call `/upgrade/start` or `/upgrade/finish`; smooth upgrade is automatic in TiDB.

#### HTTP switch-controlled

- Source in `[v7.1.2, v7.2.0)` to target in `[v7.1.2, v7.2.0)`.
- Source in `[v7.1.2, v7.2.0)` or `>= v7.4.0` to target `>= v7.4.0`.

Only these pairs require TiDB Operator to call `/upgrade/start` and `/upgrade/finish`.

### Detecting version upgrades

Add shared helper logic, for example in `pkg/manager/member/smooth_upgrade.go`:

```go
type smoothUpgradeSupport string

const (
    smoothUpgradeUnsupported      smoothUpgradeSupport = "Unsupported"
    smoothUpgradeAutoSupported    smoothUpgradeSupport = "AutoSupportedNoop"
    smoothUpgradeSwitchControlled smoothUpgradeSupport = "SwitchControlled"
)
```

The helper should:

1. Compare only the main TiDB container image tag in the old and new StatefulSet templates.
2. Return `not a version upgrade` when the tag is unchanged, even if other parts of the pod template changed.
3. Classify the source and target versions using the matrix above.
4. Treat invalid/custom tags as unsupported for smooth-upgrade API calls.

This prevents smooth-upgrade API calls for scale-only, config-only, resource-only, annotation-only, restart, failover, and volume-change reconciliations.

### Persisted controller state

Use controller-owned annotations on the `TidbCluster` as the source of truth for an active pause window:

```text
tidb.pingcap.com/smooth-upgrade-ddl-paused: "true"
tidb.pingcap.com/smooth-upgrade-source-version: "<source>"
tidb.pingcap.com/smooth-upgrade-target-version: "<target>"
tidb.pingcap.com/smooth-upgrade-started-at: "<RFC3339 timestamp>"
```

Rules:

1. Set annotations only after `/upgrade/start` succeeds.
2. Treat `smooth-upgrade-ddl-paused=true` as meaning `/upgrade/start` has succeeded and `/upgrade/finish` is still owed.
3. Do not call `/upgrade/start` again while the annotation is active.
4. Remove all smooth-upgrade annotations only after `/upgrade/finish` succeeds.
5. If annotations are present after operator restart, continue waiting for TiDB upgrade completion and then call finish.
6. If annotation metadata conflicts with the current source/target pair, emit a warning, call `/upgrade/finish` (which is idempotent and safe to call even if TiDB is already in normal state), and remove annotations after finish succeeds. Do not silently drop the annotation without calling finish.

Status conditions may be added for visibility, but must not be the source of truth. Events and logs should be emitted for start, finish, skip, and failure cases.

### Reconcile flow

#### Start gate

When TiDB upgrader is about to start an eligible TiDB version rolling upgrade:

1. Run existing blockers first: do not start while PD/TiKV/TiFlash/Pump are upgrading/scaling or TiDB is scaling.
2. Detect whether old/new TiDB StatefulSet images represent a version upgrade.
3. Classify the source/target pair.
4. If the pair is not switch-controlled, skip HTTP API calls and proceed with existing behavior.
5. If the pause annotation is already active, skip duplicate start and proceed with rolling upgrade.
6. Select one healthy TiDB endpoint, preferably the lowest healthy ordinal.
7. Call `POST /upgrade/start` with a 30-second client timeout.
8. If start fails or the client times out, do not advance the rolling update; return an error/requeue. Retry is safe because duplicate start returns HTTP 200.
9. If start returns HTTP 200 success or duplicate-upgrading, persist the annotations and proceed with existing rolling-upgrade logic; the DDL owner has synced the upgrading state before the first-success response returns.

#### Finish gate

Call `maybefinishSmoothUpgrade(tc)` at the end of `tidbMemberManager.Sync()`, after `syncTidbClusterStatus` has run. This ensures the phase and StatefulSet status fields reflect the current state before the check.

1. Check whether the pause annotation is active.
2. If not active, do nothing.
3. If active, wait until `tc.Status.TiDB.StatefulSet.UpdatedReplicas == tc.Status.TiDB.StatefulSet.Replicas` and `tc.Status.TiDB.Phase == NormalPhase`. Use these StatefulSet status fields rather than pod-level labels to avoid races between pod label updates and StatefulSet controller reconciliation.
4. Select one healthy TiDB endpoint.
5. Call `POST /upgrade/finish`.
6. If finish fails, keep annotations and requeue.
7. If finish succeeds, remove all smooth-upgrade annotations.

### Controller API changes

Extend `TiDBControlInterface`:

```go
type TiDBControlInterface interface {
    GetHealth(tc *v1alpha1.TidbCluster, ordinal int32) (bool, error)
    GetInfo(tc *v1alpha1.TidbCluster, ordinal int32) (*DBInfo, error)
    SetServerLabels(tc *v1alpha1.TidbCluster, ordinal int32, labels map[string]string) error
    StartUpgrade(tc *v1alpha1.TidbCluster, ordinal int32) error
    FinishUpgrade(tc *v1alpha1.TidbCluster, ordinal int32) error
}
```

`defaultTiDBControl` should reuse the existing TiDB HTTP client and base URL logic, except `StartUpgrade` must use a dedicated 30-second-timeout client as noted above. `FakeTiDBControl` should record the ordinal passed to each start/finish call (not just a call count) so unit tests can assert that the lowest healthy ordinal was selected, and support injected errors per method.

### Events

Suggested event reasons:

- `SmoothUpgradeStarted`
- `SmoothUpgradeFinished`
- `SmoothUpgradeStartFailed`
- `SmoothUpgradeFinishFailed`
- `SmoothUpgradeUnsupported`
- `SmoothUpgradeSkipped`

### Test Plan

#### Unit tests

- `pkg/controller/tidb_control_test.go`
  - `StartUpgrade` sends bodyless `POST /upgrade/start`.
  - `StartUpgrade` accepts `"success!"` and the duplicate-upgrading HTTP 200 body.
  - `FinishUpgrade` sends bodyless `POST /upgrade/finish`.
  - `FinishUpgrade` accepts `"success!"` and the duplicate-normal HTTP 200 body.
  - HTTP 400/5xx, client timeout, and unexpected response bodies return useful errors.
  - `StartUpgrade` uses a timeout greater than TiDB server-side 10-second owner-sync timeout; recommended 30 seconds.
  - TLS-enabled clusters reuse existing TiDB HTTP client behavior.
- `pkg/manager/member/smooth_upgrade_test.go`
  - Detect version upgrade by image tag change only.
  - Ignore config/resource/annotation-only template changes.
  - Classify every row in the TiDB version matrix.
  - Treat invalid/custom tags as unsupported.
  - Read/write/clear smooth-upgrade annotations.
  - Detect active, stale, and mismatched annotation states.
- `pkg/manager/member/tidb_upgrader_test.go`
  - Call start before first partition decrement for switch-controlled version upgrades.
  - Start failure blocks StatefulSet rollout.
  - Active annotation avoids duplicate start.
  - Auto-supported and unsupported pairs skip start.
- `pkg/manager/member/tidb_member_manager_test.go`
  - Call finish only after all TiDB pods are updated and healthy.
  - Finish failure requeues and keeps annotations.
  - Finish success removes annotations.
  - Scale-only and config-only changes do not call start or finish.

#### Integration or manual tests

1. Deploy a switch-controlled source version, for example `v7.4.x` or later.
2. Start a long-running user DDL workload.
3. Update `spec.version` to a supported target.
4. Verify `/upgrade/start` is called before the first TiDB pod replacement.
5. Verify the smooth-upgrade annotations are present during rolling upgrade.
6. Verify `/upgrade/finish` is called after all TiDB pods are upgraded.
7. Verify annotations are removed after finish succeeds.
8. Restart the operator after start and before finish; verify finish still happens.

## Drawbacks

- The controller must update `TidbCluster` annotations in addition to status and StatefulSet reconciliation, which introduces conflict/retry handling.
- If TiDB's repeated start/finish behavior is not fully idempotent, the controller must be careful to avoid duplicate start and to classify finish responses safely.
- Unsupported version pairs still rely on users avoiding DDL manually; the operator can only warn and preserve existing behavior.
- A stalled upgrade (pod crashloop, image pull failure, etc.) holds the DDL pause indefinitely. The operator does not call `/upgrade/finish` until all pods are on the new revision and healthy, so user DDL remains blocked for the full duration of any stuck upgrade. Users must resolve the underlying pod issue to unblock DDL.

## Alternatives

### Use status conditions as the source of truth

Status conditions are useful for visibility, but they are observations and can be recalculated or overwritten by status-sync paths. The active pause window is controller workflow state: after `/upgrade/start` succeeds, the controller owes a future `/upgrade/finish`. Annotations are a better fit for this durable lock-like state.

### Always call `/upgrade/start` for every TiDB version upgrade

This is simpler but incorrect for unsupported versions and unnecessary for auto-supported versions. It could fail upgrades that currently work and would not follow TiDB's documented version matrix.

### Let users call the API manually

This preserves current behavior but does not provide a TiUP-like operator experience. It is error-prone because users must coordinate API calls precisely with Kubernetes rolling updates.

### Use `/upgrade/show` as the reconciliation source of truth

`/upgrade/show` can help diagnostics, but relying on live TiDB state alone does not solve operator restart and in-flight workflow tracking as cleanly as controller-owned annotations. It can be added later as a defensive check if needed.
