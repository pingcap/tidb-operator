# Replication Restore PR 2: backup-manager CLI + BR integration

> **Spec reference:** `docs/superpowers/specs/2026-04-23-replication-restore-design.md`, sections §3 (Status Write Ownership / Option B), §6 (BR CLI Surface), §7 (Module Layout — PR 2 rows).
>
> **Companion plan:** `docs/superpowers/plans/2026-04-23-replication-restore-pr1.md` (operator-side foundation; already implemented on this branch).

## Context

PR 1 landed the operator-side state machine, two-Job orchestration, and the `replicationRestoreStatusUpdater` no-op wrapper as a callable but **not yet activated** type. PR 1 also intentionally shipped a stub `buildPiTRBaseArgs` that produces a Job which is **functionally insufficient for backup-manager to actually run BR end-to-end** (wrong main-container image, no TLS volumes, no env vars, no init container that copies the BR binary).

PR 2 closes the remaining gap so a `Restore` with `replicationConfig != nil` actually runs to completion. Three concrete deficits get fixed:

1. **P1.1 — Job is a stub.** `buildPiTRBaseArgs` returns a 4-arg slice and `makeReplicationBRJob` puts `restore.Spec.ToolImage` as the main container image. The real Job needs to come from `restoreManager.makeRestoreJobWithMode` so it inherits the full PiTR setup (TiDBBackupManagerImage main container, BR-binary init container, TLS volumes, password/storage env, additional volumes, etc.). PR 2 wires `replicationHandler` to call `makeRestoreJobWithMode` and then post-processes the result via a small `applyReplicationPhase` helper that overrides Job name, adds replication labels, and appends `--replicationPhase=N`.

2. **P2.1 — Replication path skips `pm.Enable`.** Standard PiTR restores poll the TiKV `gc.ratio-threshold` ConfigMap until it has been overridden to `-1` (the override itself is written by `tikvMemberManager.applyPiTRConfigOverride` in the TiKV reconcile loop, which already covers replication restores because they satisfy `Spec.Mode == PiTR`). Today the replication interception at `restore_manager.go:300-302` returns *before* `pm.Enable` runs, so the gate is bypassed. PR 2 reorders `pm.Enable` to before the replication interception so both PiTR variants share the gate.

3. **§6 — backup-manager doesn't know about replication.** `cmd/backup-manager/app/cmd/restore.go` has no `--replicationPhase` flag, the `Options` struct has no field for it, the PiTR branch of `restoreData` doesn't append the three new BR flags, and `runRestore` always uses the real status updater (never the no-op wrapper Option B requires for phase-1). PR 2 adds these four pieces.

### Combined-PR decision (deviation from spec §PR Breakdown)

The spec splits PR 1 (operator) and PR 2 (backup-manager) into separate PRs because PR 2's BR flags depend on internal BR kernel readiness. As of 2026-04-29 the BR kernel exposes `--replication-storage-phase`, `--replication-status-sub-prefix`, and `--pitr-concurrency`, so this constraint is satisfied. Per maintainer decision the PR 2 work is folded into the existing `wip-ccr-test` branch and shipped as a single PR; this plan file keeps the "PR 2" name to remain readable against the spec's section numbering.

### Scope explicitly NOT in this PR

- **Job watch on the operator side (PR 1 review item P1.3).** Phase-2 `Phase=Running/Complete/Failed` writes come from backup-manager (Option B unwraps in phase-2, see §3); the controller observes those writes through the existing Restore informer. Phase-1 Job completion is detected on the next Restore resync (≤ 30 s default). This latency is acceptable for now; an explicit Job informer is a separate follow-up.
- **TidbCluster controller does not watch Restore CRs.** This makes the GC override write hit a delay between user creating the Restore and TiKV's reconcile firing. Existing standard PiTR has the same delay; replication does not make it worse. Follow-up.
- **Full §10 integration scenarios against a real BR.** The 9 scenarios listed in the spec's Test Plan need a live tikv + BR. PR 2 covers the must-have happy paths at handler-level (no real BR), per the maintainer's "先加必须的测试" decision; full e2e is deferred to manual verification post-merge.

## File-level scope

| File | Change | Approx. lines |
|------|--------|---------------|
| `cmd/backup-manager/app/restore/restore.go` | Add `Options.ReplicationPhase int` field; add `replicationStatusSubPrefix` / `replicationPiTRConcurrency` constants at file top; add new package-private helper `replicationBRFlags(phase int) []string`; one-line edit inside `restoreData`'s PiTR branch to call the helper. **Existing `restoreData` body otherwise unchanged.** | +40, −0 |
| `cmd/backup-manager/app/cmd/restore.go` | Register `--replicationPhase` flag (cobra `IntVar`); add new helper `validateReplicationPhase` and call it on entry to `runRestore`; add new helper `selectRestoreStatusUpdater` and use it to swap between no-op wrapper (phase 1) and real updater (otherwise). **Two-line edit at the existing `statusUpdater :=` site.** | +25, −1 |
| `pkg/backup/restore/replication_handler.go` | Replace stub `buildPiTRBaseArgs` body with a call to an injected job builder; add new `applyReplicationPhase` post-processor; add `buildJob` field + constructor parameter on `replicationHandler` (struct introduced in PR 1, so adding a field is permitted). **`buildPiTRBaseArgs` deleted entirely after the migration.** | +50, −40 |
| `pkg/backup/restore/restore_manager.go` | Reorder three blocks at L296-323 so `pm.Enable(tc)` runs before the replication interception (P2.1); update the comment block above the interception to reflect the new order. Two-step `NewRestoreManager` so it can pass `rm.makeRestoreJobWithMode` into `newReplicationHandler`. **No changes inside `makeRestoreJobWithMode` itself.** | +10, −8 |
| `pkg/backup/restore/replication_handler_test.go` | Update test fixtures so `newTestReplicationHandler` injects a fake job builder; add unit tests for `applyReplicationPhase`; update three existing PR-1 tests (`TestMakeReplicationBRJob_*`) so their assertions track the new "post-process a fake-builder Job" pattern; add four handler-level scenario tests (Task 7) | +180, −60 |
| `cmd/backup-manager/app/restore/restore_test.go` *(new file)* | Unit tests for `Options.ReplicationPhase` default and `replicationBRFlags` table-driven over `{0, 1, 2}` | +60 |
| `cmd/backup-manager/app/cmd/restore_test.go` *(new file)* | Unit tests for `validateReplicationPhase` (range check) and `selectRestoreStatusUpdater` (returns no-op iff phase==1) | +50 |

Net diff target: **≈ +415, −110**, distributed across 7 files.

> **Refactoring constraint:** existing functions (`restoreData`, `makeRestoreJobWithMode`, `runRestore`'s control flow outside the noted single edit) are not touched. Only single-line edits at well-defined points; new logic lives in new helpers. New helpers and PR-1-introduced types (`replicationHandler`) are extracted / extended freely.

## Glossary

- **`ReplicationPhase`** — int field on `Options`. Sentinel value 0 means "this is not a replication restore"; 1 = snapshot phase; 2 = log phase. Range-validated at backup-manager startup.
- **No-op wrapper / Option B** — `replicationRestoreStatusUpdater` (defined in `pkg/controller/restore_status_updater.go`, landed in PR 1). Returns nil from `Update` so backup-manager's status writes during phase-1 are silently dropped. Activated only when `ReplicationPhase == 1`.
- **`applyReplicationPhase`** — new helper in `pkg/backup/restore/replication_handler.go`. Takes a Job built by `makeRestoreJobWithMode` and mutates it: rewrites Job name, adds `ReplicationStepLabelKey` and `RestoreUIDLabelKey` to job and pod labels, appends `--replicationPhase=N` to the main container args.
- **`buildJob`** — function-typed dependency injected into `replicationHandler` at construction. Production wiring passes `rm.makeRestoreJobWithMode`; tests pass a fake that returns a pre-built Job with assertable shape.
- **PiTR-mode common gate** — `pm.Enable(tc)` polling the TiKV ConfigMap. Reorder makes it apply to both standard and replication PiTR.

## Task 1: Add `Options.ReplicationPhase` field

Smallest unit; no dependencies.

- [ ] **Step 1: Write a failing test for the field default**

```go
// cmd/backup-manager/app/restore/restore_test.go (new file)
package restore

import "testing"

func TestOptions_ReplicationPhase_DefaultsToZero(t *testing.T) {
    opts := Options{}
    if opts.ReplicationPhase != 0 {
        t.Fatalf("expected ReplicationPhase default 0, got %d", opts.ReplicationPhase)
    }
}
```

- [ ] **Step 2: Run — expect compile failure** (`Options.ReplicationPhase` not defined yet).

```bash
go test ./cmd/backup-manager/app/restore/...
```

- [ ] **Step 3: Add the field**

In `cmd/backup-manager/app/restore/restore.go` extend the existing `Options` struct:

```go
type Options struct {
    backupUtil.GenericOptions
    Prepare bool
    TargetAZ string
    UseFSR bool
    Abort bool

    // ReplicationPhase indicates which phase of a two-phase replication
    // restore this backup-manager invocation handles. Valid values:
    //   1 — snapshot-restore phase (status writes suppressed by Option B
    //       no-op wrapper; controller is sole writer of status.Phase)
    //   2 — log-restore phase (backup-manager directly writes Phase /
    //       Running / Complete / Failed; no wrapper)
    // The zero value (0) means this is NOT a replication restore — the
    // CLI flag was not passed and standard PiTR / snapshot semantics
    // apply. See spec §3 (Option B) and §6 (BR CLI surface).
    ReplicationPhase int
}
```

- [ ] **Step 4: Run — expect pass**

```bash
go test ./cmd/backup-manager/app/restore/...
```

- [ ] **Step 5: Commit**

Commit message:
```
feat(replication-restore): add Options.ReplicationPhase field

Sentinel 0 means "not a replication restore" (CLI flag unset);
non-zero values are validated in Task 2 when the flag is registered.
```

## Task 2: Register `--replicationPhase` flag and validate

Builds on Task 1. After this task, the field has a real CLI source and runtime validation.

- [ ] **Step 1: Write a failing test for the flag binding and validation**

In `cmd/backup-manager/app/cmd/restore_test.go` (new file) — use cobra's `Flags().Set()` to drive the bound variable:

```go
func TestNewRestoreCommand_BindsReplicationPhase(t *testing.T) {
    cmd := NewRestoreCommand()
    if err := cmd.Flags().Set("replicationPhase", "2"); err != nil {
        t.Fatalf("setting --replicationPhase=2: %v", err)
    }
    // The bound Options.ReplicationPhase is internal to NewRestoreCommand's
    // closure; test by parsing through cmd flagset directly.
    got, err := cmd.Flags().GetInt("replicationPhase")
    if err != nil { t.Fatal(err) }
    if got != 2 { t.Fatalf("want 2, got %d", got) }
}

func TestRunRestore_RejectsInvalidReplicationPhase(t *testing.T) {
    // Direct call into validation helper (extracted in Step 3 below)
    cases := []struct{
        in   int
        want bool // true = expect error
    }{ {-1, true}, {0, false}, {1, false}, {2, false}, {3, true}, {99, true} }
    for _, c := range cases {
        err := validateReplicationPhase(c.in)
        if (err != nil) != c.want {
            t.Fatalf("phase=%d, expected error=%v, got %v", c.in, c.want, err)
        }
    }
}
```

- [ ] **Step 2: Run — expect failures**

The `Flags().GetInt` call returns "flag accessed but not defined"; `validateReplicationPhase` doesn't exist yet.

```bash
go test ./cmd/backup-manager/app/cmd/...
```

- [ ] **Step 3: Implement flag binding and validation**

In `cmd/backup-manager/app/cmd/restore.go`, inside `NewRestoreCommand()` after the existing flag definitions:

```go
cmd.Flags().IntVar(&ro.ReplicationPhase, "replicationPhase", 0,
    "Replication restore phase: 1 = snapshot, 2 = log. "+
        "Omit (or 0) for standard PiTR / snapshot.")
```

Add `validateReplicationPhase` as a package-level helper in the same file:

```go
// validateReplicationPhase ensures the CLI flag value is in {0, 1, 2}.
// 0 is the sentinel for "flag not passed / not a replication restore";
// 1 and 2 are the two replication phases. Anything else is operator
// or user error.
func validateReplicationPhase(p int) error {
    switch p {
    case 0, 1, 2:
        return nil
    default:
        return fmt.Errorf(
            "invalid --replicationPhase=%d (must be 1 or 2; omit for standard restore)",
            p,
        )
    }
}
```

In `runRestore` near the entry, before constructing the informer factory:

```go
if err := validateReplicationPhase(restoreOpts.ReplicationPhase); err != nil {
    return err
}
```

- [ ] **Step 4: Run — expect pass**

```bash
go test ./cmd/backup-manager/app/cmd/...
```

- [ ] **Step 5: Commit**

Commit message:
```
feat(replication-restore): register --replicationPhase flag with validation

Sentinel 0 = "not a replication restore" (cobra default when flag is
absent). Reject {-N, ≥3} early in runRestore so the operator gets a
clean error instead of silently mishandling the value downstream.
```

## Task 3: Conditional status-updater wrapper swap (Option B activation)

Activates the no-op wrapper from PR 1 when `ReplicationPhase == 1`. Phase-2 keeps the real updater.

- [ ] **Step 1: Write a failing test**

The wrapper-vs-real selection is small enough that a direct unit test is fine. Add to `cmd/backup-manager/app/cmd/restore_test.go`:

```go
func TestRunRestore_StatusUpdaterSelection(t *testing.T) {
    cases := []struct {
        phase     int
        wantNoOp  bool // true = expect *replicationRestoreStatusUpdater
    }{
        {0, false},
        {1, true},
        {2, false},
    }
    for _, c := range cases {
        got := selectRestoreStatusUpdater(restore.Options{ReplicationPhase: c.phase}, /* fakeReal */ nil)
        _, isNoOp := got.(*controller.replicationRestoreStatusUpdater)
        if isNoOp != c.wantNoOp {
            t.Fatalf("phase=%d: wantNoOp=%v, gotNoOp=%v", c.phase, c.wantNoOp, isNoOp)
        }
    }
}
```

> **Note:** Since `replicationRestoreStatusUpdater` is unexported, the test casts the interface to a known no-op type. If unexported access is awkward, use `controller.IsReplicationRestoreNoOp(updater)` — a tiny exported helper added in PR 1's package — or call `Update` and assert it returns nil without side effect.

- [ ] **Step 2: Run — expect fail**

```bash
go test ./cmd/backup-manager/app/cmd/...
```

- [ ] **Step 3: Extract `selectRestoreStatusUpdater` and rewrite `runRestore`**

```go
// In cmd/backup-manager/app/cmd/restore.go
func selectRestoreStatusUpdater(
    opts restore.Options,
    real controller.RestoreConditionUpdaterInterface,
) controller.RestoreConditionUpdaterInterface {
    if opts.ReplicationPhase == 1 {
        // Phase-1 Option B: suppress backup-manager status writes so the
        // controller remains the sole writer of Phase=SnapshotRestore and
        // condition markers. Phase-2 falls through to the real updater.
        return controller.NewReplicationRestoreStatusUpdater()
    }
    return real
}
```

Replace `cmd/backup-manager/app/cmd/restore.go:72`:

```go
// before
statusUpdater := controller.NewRealRestoreConditionUpdater(cli, restoreInformer.Lister(), recorder)

// after
realStatusUpdater := controller.NewRealRestoreConditionUpdater(cli, restoreInformer.Lister(), recorder)
statusUpdater := selectRestoreStatusUpdater(restoreOpts, realStatusUpdater)
```

- [ ] **Step 4: Run — expect pass**

```bash
go test ./cmd/backup-manager/app/cmd/...
```

- [ ] **Step 5: Commit**

Commit message:
```
feat(replication-restore): activate no-op status updater for phase 1

Symmetric to compact.go's ShardedCompactStatusUpdater wrap, but driven
by the CLI flag (not CR.Spec) because the same Restore CR runs through
backup-manager twice with different phases.
```

## Task 4: Append BR replication flags + define constants

The PiTR branch in `restoreData` already builds the BR command line for `--restored-ts` and the full-backup-storage args. Replication adds three more.

- [ ] **Step 1: Write a failing test for the new helper**

The test targets a new package-private helper `replicationBRFlags(phase int) []string`. The existing `restoreData` is **not** refactored; only a single new line is added inside it to call this helper.

`cmd/backup-manager/app/restore/restore_test.go`:

```go
func TestReplicationBRFlags(t *testing.T) {
    cases := []struct {
        phase int
        want  []string
    }{
        {phase: 0, want: nil},
        {phase: 1, want: []string{
            "--replication-storage-phase=1",
            "--replication-status-sub-prefix=ccr",
            "--pitr-concurrency=1024",
        }},
        {phase: 2, want: []string{
            "--replication-storage-phase=2",
            "--replication-status-sub-prefix=ccr",
            "--pitr-concurrency=1024",
        }},
    }
    for _, c := range cases {
        got := replicationBRFlags(c.phase)
        if !reflect.DeepEqual(got, c.want) {
            t.Fatalf("phase=%d: got %v, want %v", c.phase, got, c.want)
        }
    }
}
```

- [ ] **Step 2: Run — expect failure**

`replicationBRFlags` doesn't exist yet.

- [ ] **Step 3: Define constants and the new helper; minimal edit inside `restoreData`**

At the top of `cmd/backup-manager/app/restore/restore.go`, after the imports:

```go
const (
    // replicationStatusSubPrefix is the fixed sub-prefix BR uses to lay
    // out replication status under the log-backup storage. Hardcoded
    // here (not exposed via API) per spec §6: this is a BR call detail
    // and shouldn't leak into the CRD.
    replicationStatusSubPrefix = "ccr"

    // replicationPiTRConcurrency is the parallelism BR uses when
    // applying compacted log files. Spec §6.
    replicationPiTRConcurrency = 1024
)

// replicationBRFlags returns the BR command-line flags that are
// specific to replication restore phases. Returns nil for phase 0
// (= not a replication restore), in which case the caller appends
// nothing. Spec §6.
func replicationBRFlags(phase int) []string {
    if phase == 0 {
        return nil
    }
    return []string{
        fmt.Sprintf("--replication-storage-phase=%d", phase),
        fmt.Sprintf("--replication-status-sub-prefix=%s", replicationStatusSubPrefix),
        fmt.Sprintf("--pitr-concurrency=%d", replicationPiTRConcurrency),
    }
}
```

Inside `restoreData`'s `case string(v1alpha1.RestoreModePiTR):` branch, after the existing PiTR arg building (`--restored-ts`, `--full-backup-storage`, etc.) and before `restoreType = "point"`, add **one** line:

```go
args = append(args, replicationBRFlags(ro.ReplicationPhase)...)
```

`append(slice, nil...)` is a no-op when phase is 0, so standard PiTR is unaffected without any conditional in `restoreData` itself.

> **Constraint reminder:** `restoreData` (existing ~200-line function) is otherwise untouched. The diff inside `restoreData` is exactly one append line. All replication-specific arg construction lives in the new helper.

- [ ] **Step 4: Run — expect pass**

```bash
go test ./cmd/backup-manager/app/restore/...
```

- [ ] **Step 5: Commit**

Commit message:
```
feat(replication-restore): append BR replication flags in PiTR branch

Three BR flags are now appended when --replicationPhase > 0:
--replication-storage-phase, --replication-status-sub-prefix=ccr,
--pitr-concurrency=1024. Constants live in restore.go (BR call
details; not exposed via API). Arg construction is extracted to a
testable buildBRArgs helper.
```

## Task 5: Reorder `pm.Enable` before replication interception (P2.1)

Operator-side; independent of Tasks 1–4. Closes the GC-gate gap for replication restores.

- [ ] **Step 1: Write a failing test for the gate**

In `pkg/backup/restore/replication_handler_test.go` (or a sibling test file in the same package — placement should match how existing `pm.Enable`-related tests are organized; if standard PiTR doesn't have a unit test for this gate, add the test to `replication_handler_test.go`):

```go
func TestSyncRestoreJob_ReplicationRestore_GatedByPMEnable(t *testing.T) {
    // Build a TidbCluster ConfigMap that does NOT yet have
    // gc.ratio-threshold = -1 (override hasn't propagated).
    // Build a Restore with mode=PiTR, ReplicationConfig != nil,
    // status.Phase="" (just created).
    //
    // Call rm.syncRestoreJob(restore) once.
    //
    // Assert: returns RequeueError (from pm.Enable polling).
    // Assert: replicationHandler.Sync was NOT called (no Job
    //         created, no Phase=SnapshotRestore written).
    //
    // Then update the ConfigMap to set ratio-threshold = -1.
    // Call rm.syncRestoreJob(restore) again.
    // Assert: replicationHandler.Sync WAS called this time
    //         (Phase=SnapshotRestore should appear via the
    //         status updater hook).
}
```

- [ ] **Step 2: Run — expect failure** (or, in the current code, expect the test to pass *too easily* because today the replication branch returns before `pm.Enable` runs, so it would create a Job even with no ConfigMap override). Either failure mode validates the test is meaningful.

- [ ] **Step 3: Reorder the three blocks in `restore_manager.go:296–323`**

Move the `if Mode == PiTR { pm.Enable(tc) }` block to run **before** the replication interception. The replication interception then sits between `pm.Enable` and the single-Job lookup; the single-Job lookup runs only on the standard path. Update the comment block that currently says "All logic below this line assumes a single-Job Restore" to remain accurate after the reorder (it still applies to the lines below the interception, which is now further down).

After the change, the structure is:

```go
// pm.Enable is the GC-disabled gate shared by all PiTR restores
// (standard PiTR and replication restore alike). Run it before the
// replication interception so replication restores also wait for the
// TiKV ConfigMap override to propagate before launching any BR Job.
if restore.Spec.Mode == v1alpha1.RestoreModePiTR {
    if err := pm.Enable(tc); err != nil {
        if controller.IsRequeueError(err) {
            return err
        }
        return fmt.Errorf("restore %s/%s enable pitr failed, err: %v", ns, name, err)
    }
}

// Replication restore: delegate to dedicated handler that owns the
// two-Job state machine and coordinates with CompactBackup. All logic
// below this line assumes a single-Job Restore (GetRestoreJobName etc.)
// which does not apply to replication restore.
if restore.Spec.Mode == v1alpha1.RestoreModePiTR && restore.Spec.ReplicationConfig != nil {
    return rm.replicationHandler.Sync(restore)
}

restoreJobName := restore.GetRestoreJobName()
// ... single-Job idempotency lookup (unchanged) ...
```

- [ ] **Step 4: Run — expect pass**

```bash
go test ./pkg/backup/restore/...
```

- [ ] **Step 5: Commit**

Commit message:
```
fix(replication-restore): apply pm.Enable gate to replication path

Reorder so pm.Enable runs before the replication interception. Both
PiTR variants (standard PiTR and replication restore) now wait for
TiKV's gc.ratio-threshold = -1 override to propagate via ConfigMap
before any BR Job is launched.

The override itself is written by tikvMemberManager.applyPiTRConfigOverride
in the TiKV reconcile loop, which already covers replication restores
because they satisfy Spec.Mode == PiTR. No TiKV-side change is needed.
```

## Task 6: Inject Job builder + extract `applyReplicationPhase` helper (P1.1)

Replace the stub `buildPiTRBaseArgs` so phase-1 / phase-2 Jobs are real, runnable BR Jobs.

- [ ] **Step 1: Write failing tests**

Three test groups in `pkg/backup/restore/replication_handler_test.go`:

```go
// 1. applyReplicationPhase mutations
func TestApplyReplicationPhase_RewritesJobName(t *testing.T) {
    in := newFakeRestoreJob("dr-restore", "r1-uid")
    out := applyReplicationPhase(in, fixtureRestore(), label.ReplicationStepSnapshotRestoreVal)
    if got := out.Name; got != "dr-restore-snapshot-restore" {
        t.Fatalf("Job name = %q, want %q", got, "dr-restore-snapshot-restore")
    }
}

func TestApplyReplicationPhase_AddsReplicationLabels(t *testing.T) {
    // Verify ReplicationStepLabelKey and RestoreUIDLabelKey are added
    // to BOTH job.Labels and job.Spec.Template.Labels (pod labels).
}

func TestApplyReplicationPhase_AppendsReplicationPhaseArg(t *testing.T) {
    // Verify the main container's args has --replicationPhase=1 (or 2)
    // appended at the end. Existing args are preserved.
}

// 2. makeReplicationBRJob composes the builder + helper
func TestMakeReplicationBRJob_DelegatesToBuilder(t *testing.T) {
    var calledWith *v1alpha1.Restore
    fakeBuilder := func(r *v1alpha1.Restore, isPrune bool) (*batchv1.Job, string, error) {
        calledWith = r
        return newFakeRestoreJob("dr-restore", "r1-uid"), "", nil
    }
    h := newTestReplicationHandlerWithBuilder(fakeBuilder)
    job, err := h.makeReplicationBRJob(fixtureRestore(), label.ReplicationStepLogRestoreVal)
    if err != nil { t.Fatal(err) }
    if calledWith == nil { t.Fatal("builder was not called") }
    // Verify the post-processing happened (name override + label + arg).
    if job.Name != "dr-restore-log-restore" {
        t.Fatalf("Job name = %q, want %q", job.Name, "dr-restore-log-restore")
    }
}

// 3. Existing tests that asserted stub Job shape need to be updated
// to use the fake builder fixture. Specifically:
// - TestMakeReplicationBRJob_SnapshotRestore_HasCorrectLabelsAndArgs
// - TestMakeReplicationBRJob_LogRestore_HasCorrectPhaseArg
// - TestMakeReplicationBRJob_StampsRestoreUIDLabel
// These tests previously verified labels/args directly on the stub Job;
// now they verify post-processing on top of a fake-builder Job that
// supplies a different starting shape.
```

- [ ] **Step 2: Run — expect failures**

`applyReplicationPhase` doesn't exist; `newTestReplicationHandlerWithBuilder` doesn't exist; the existing PR-1 tests reference the stub `buildPiTRBaseArgs` flow that's being replaced.

- [ ] **Step 3: Implement**

In `pkg/backup/restore/replication_handler.go`:

(a) Add a function-typed dependency to the handler struct:

```go
type jobBuilderFunc func(restore *v1alpha1.Restore, isPruneJob bool) (*batchv1.Job, string, error)

type replicationHandler struct {
    deps          *controller.Dependencies
    statusUpdater controller.RestoreConditionUpdaterInterface
    recorder      record.EventRecorder
    buildJob      jobBuilderFunc // injected; production = rm.makeRestoreJobWithMode
}

func newReplicationHandler(
    deps *controller.Dependencies,
    statusUpdater controller.RestoreConditionUpdaterInterface,
    recorder record.EventRecorder,
    buildJob jobBuilderFunc,
) *replicationHandler {
    return &replicationHandler{deps: deps, statusUpdater: statusUpdater, recorder: recorder, buildJob: buildJob}
}
```

(b) Replace `makeReplicationBRJob`:

```go
// makeReplicationBRJob produces a BR Job for the given replication step
// by delegating to the standard PiTR Job builder and post-processing
// the result with replication-specific overrides.
//
// The Job inherits everything makeRestoreJobWithMode produces — main
// container image (TiDBBackupManagerImage), BR-binary init container
// + br-bin shared volume, TLS volume mounts, password / storage env,
// resource requirements, additional volumes, owner reference. Three
// things are overridden here:
//   1. Job name: <restore.Name>-<step> instead of <restore.Name>-restore
//   2. ReplicationStepLabelKey + RestoreUIDLabelKey on Job and Pod
//   3. --replicationPhase=N appended to main container args
func (h *replicationHandler) makeReplicationBRJob(restore *v1alpha1.Restore, step string) (*batchv1.Job, error) {
    job, _, err := h.buildJob(restore, false /* isPruneJob */)
    if err != nil {
        return nil, err
    }
    return applyReplicationPhase(job, restore, step), nil
}

// applyReplicationPhase mutates a Job built by makeRestoreJobWithMode
// to make it suitable for one phase of a replication restore.
//
// Mutations:
//   - Job name set to <restore.Name>-<step>
//   - ReplicationStepLabelKey and RestoreUIDLabelKey added to both
//     job.Labels and pod template Labels
//   - --replicationPhase=1 (snapshot) or =2 (log) appended to the
//     primary container's Args
func applyReplicationPhase(job *batchv1.Job, restore *v1alpha1.Restore, step string) *batchv1.Job {
    job.Name = fmt.Sprintf("%s-%s", restore.Name, step)

    if job.Labels == nil {
        job.Labels = map[string]string{}
    }
    job.Labels[label.ReplicationStepLabelKey] = step
    job.Labels[label.RestoreUIDLabelKey] = string(restore.UID)

    if job.Spec.Template.Labels == nil {
        job.Spec.Template.Labels = map[string]string{}
    }
    job.Spec.Template.Labels[label.ReplicationStepLabelKey] = step
    job.Spec.Template.Labels[label.RestoreUIDLabelKey] = string(restore.UID)

    phase := 1
    if step == label.ReplicationStepLogRestoreVal {
        phase = 2
    }
    if len(job.Spec.Template.Spec.Containers) > 0 {
        job.Spec.Template.Spec.Containers[0].Args = append(
            job.Spec.Template.Spec.Containers[0].Args,
            fmt.Sprintf("--replicationPhase=%d", phase),
        )
    }
    return job
}
```

(c) Delete `buildPiTRBaseArgs` entirely (the documenting comment block can stay as historical context in the commit message but the function body goes).

(d) In `pkg/backup/restore/restore_manager.go`, update the constructor call to pass the production builder:

```go
// before
replicationHandler: newReplicationHandler(deps, statusUpdater, deps.Recorder),

// after
rm := &restoreManager{...}
rm.replicationHandler = newReplicationHandler(
    deps, statusUpdater, deps.Recorder,
    rm.makeRestoreJobWithMode, // production builder
)
```

> **Note on initialization order:** the constructor needs `rm` to exist before it can reference `rm.makeRestoreJobWithMode`. The existing `NewRestoreManager` (or its equivalent) needs a tiny rewrite to two-step the construction. Verify the existing tests still pass after this rewrite.

(e) Update test fixture helper (in `replication_handler_test.go`) so existing tests inject a fake builder:

```go
func newTestReplicationHandler(...) *replicationHandler {
    return newReplicationHandlerWithBuilder(..., defaultFakeBuilder())
}

func newTestReplicationHandlerWithBuilder(..., builder jobBuilderFunc) *replicationHandler {
    return newReplicationHandler(..., builder)
}

func defaultFakeBuilder() jobBuilderFunc {
    return func(r *v1alpha1.Restore, isPrune bool) (*batchv1.Job, string, error) {
        return newFakeRestoreJob(r.Name, string(r.UID)), "", nil
    }
}

func newFakeRestoreJob(name, uid string) *batchv1.Job {
    // Construct a minimal *batchv1.Job that resembles makeRestoreJobWithMode's
    // output: name = <name>-restore, one container with some args, owner ref,
    // RestoreLabelKey set. Tests assert on what applyReplicationPhase mutates,
    // so the fake doesn't need to be exhaustive.
}
```

- [ ] **Step 4: Run — expect pass**

```bash
go test ./pkg/backup/restore/... -count=1
```

- [ ] **Step 5: Commit**

Commit message:
```
feat(replication-restore): wire real BR Job via makeRestoreJobWithMode

Replace the stub buildPiTRBaseArgs with an injected job builder
(production = rm.makeRestoreJobWithMode) plus a small
applyReplicationPhase post-processor. Phase-1 and phase-2 Jobs now
inherit the full PiTR Job setup: TiDBBackupManagerImage main
container, BR-binary init container with br-bin shared volume, TLS
volume mounts, password / storage env, owner reference. Replication
adds three mutations on top: Job name override, replication-step +
restore-UID labels, --replicationPhase arg.
```

## Task 7: Required handler-level scenario tests

Cover the must-have happy paths and the wrapper non-regression. Full §10 e2e against a real BR is out of scope.

- [ ] **Step 1: Write failing scenario tests**

Add to `pkg/backup/restore/replication_handler_test.go`:

```go
// Scenario 1: phase-1 success + CompactBackup Complete → phase-2 Job created.
// Verifies the gate logic combined with the new real-Job wiring:
// - first Sync: Phase=SnapshotRestore + Step="snapshot-restore" written
// - second Sync: phase-1 Job created (with TiDBBackupManagerImage main
//   container, --replicationPhase=1 arg, replication labels)
// - mark phase-1 Job Complete + CompactBackup Complete
// - third Sync: SnapshotRestored marker written
// - fourth Sync: CompactSettled marker written (or in same Sync via top block)
// - fifth Sync: Phase=LogRestore + Step="log-restore" written
// - sixth Sync: phase-2 Job created (with --replicationPhase=2)
func TestSync_HappyPath_PhaseOneSuccessCompactComplete_TransitionsToPhaseTwo(t *testing.T) {
    // ... setup ...
}

// Scenario 2: phase-1 success + CompactBackup Failed → phase-2 still launches.
// The CompactSettled reason becomes "ShardsPartialFailed" but the gate
// still passes; BR-side handles the fallback to uncompacted log.
func TestSync_HappyPath_CompactFailed_StillLaunchesPhaseTwo(t *testing.T) {
    // ... setup ...
}

// Scenario 3: phase-1 Job Failed → Restore terminal Failed.
// ReplicationStep is preserved (so triage can tell phase-1 vs phase-2).
func TestSync_HappyPath_PhaseOneJobFailed_FailsRestoreAndPreservesStep(t *testing.T) {
    // ... setup ...
}

// Scenario 4: wrapper non-regression. Simulates backup-manager calling
// statusUpdater.Update during phase-1 (where the no-op wrapper is in
// effect): verify no Restore mutation observed.
func TestReplicationRestoreStatusUpdater_DropsAllWrites_NoRestoreMutation(t *testing.T) {
    // Already covered for the wrapper itself in PR 1. This adds a
    // composition-level assertion: passing the wrapper through
    // selectRestoreStatusUpdater and Update'ing it from the fake
    // backup-manager codepath leaves the Restore CR unchanged.
}
```

- [ ] **Step 2: Run — expect failures**

The scenario test bodies don't exist yet.

- [ ] **Step 3: Implement the test bodies**

Use the existing fixtures from PR 1 (`newReplicationRestoreFixture`, `newReplicationJobNoCondition`, etc.). For each scenario:

- Build a `replicationHandler` with the fake builder and a `FakeRestoreConditionUpdater` that records writes.
- Drive the state machine via repeated `Sync` calls + intermediate fixture mutations (mark Job Complete, mark CompactBackup terminal, etc.).
- Assert the final `(Phase, ReplicationStep, Conditions[])` triple plus the Job-creation count and image / args of any created Job.

- [ ] **Step 4: Run — expect pass**

```bash
go test ./pkg/backup/restore/... -run "TestSync_HappyPath\|TestReplicationRestoreStatusUpdater_DropsAllWrites" -count=1
```

- [ ] **Step 5: Commit**

Commit message:
```
test(replication-restore): handler-level scenario tests for happy paths

Three end-to-end scenarios at handler granularity (no real BR):
- phase-1 success + CompactBackup Complete → phase-2 launches
- phase-1 success + CompactBackup Failed → phase-2 still launches
  (BR-side falls back to uncompacted log)
- phase-1 Job Failed → Restore Failed with ReplicationStep preserved

Plus a composition-level non-regression for the no-op wrapper.

Full §10 integration scenarios against a real BR cluster are deferred.
```

## Final verification

- [ ] **Full build + tests**

```bash
go build ./...
go vet ./...
go test ./... 2>&1 | tail -40
```

Expected: all packages build, all tests pass.

- [ ] **Scan for placeholder code**

```bash
grep -nE "TODO|FIXME|XXX" \
    cmd/backup-manager/app/restore/restore.go \
    cmd/backup-manager/app/cmd/restore.go \
    pkg/backup/restore/replication_handler.go \
    pkg/backup/restore/restore_manager.go
```

Expected: only `context.TODO()` (false positive). `buildPiTRBaseArgs` and its long deferral comment must be gone after Task 6.

- [ ] **Cross-check spec coverage**

Walk through spec §3 (Status Write Ownership) and §6 (BR CLI Surface) and confirm each line is realized:

- §3: phase-1 wrapper activated when `ReplicationPhase == 1` ✓ (Task 3)
- §3: phase-2 uses real updater ✓ (default branch in Task 3)
- §6: `--replicationPhase` flag added ✓ (Task 2)
- §6: `--replication-storage-phase=<1|2>` appended ✓ (Task 4)
- §6: `--replication-status-sub-prefix=ccr` appended ✓ (Task 4)
- §6: `--pitr-concurrency=1024` appended ✓ (Task 4)
- §6: constants placed in backup-manager (not types.go) ✓ (Task 4)

- [ ] **PR body update**

The branch is shipping as a single combined PR (PR 1 + PR 2). Update the PR description to reflect:

- One paragraph on the user-visible feature (replication restore = `mode=PiTR + replicationConfig`)
- Bullet list of file groups touched (operator state machine, label, status updater wrapper, handler, controller wire-up, CRD types — these are PR 1; backup-manager CLI flag + BR args + wrapper activation + Job builder rewire + pm.Enable reorder — these are PR 2)
- Test plan: list the new unit + handler-level scenario tests; note that full §10 e2e is deferred
- Compatibility statement: existing PiTR / snapshot / volume-snapshot users unaffected (`replicationConfig == nil` falls through to original code path; `--replicationPhase` defaults to 0)
- Spec link: `docs/superpowers/specs/2026-04-23-replication-restore-design.md`

- [ ] **Push branch**

```bash
git push -u origin wip-ccr-test
```

Open the PR via `gh pr create` against `master` with the body assembled above.

## Self-review checklist

1. **Spec coverage**

   - §3 (Option B) — Tasks 2 + 3 ✓
   - §4.1 (interception point) — Task 5 (reorder) ✓
   - §6 (BR CLI surface) — Tasks 1, 2, 4 ✓
   - §7 (module layout) — file mapping in "File-level scope" ✓
   - §8 (backward compatibility) — `ReplicationPhase == 0` sentinel preserves existing behavior ✓
   - §10 (test plan) — partial: unit + handler-level ✓; e2e against real BR deferred ✗

2. **PR 1 review items addressed**

   - **P1.1** Job uses backup-manager image — Task 6 ✓
   - **P1.2** `--replicationPhase` flag registered — Task 2 ✓
   - **P1.3** Job watch — explicitly deferred per maintainer decision ✗
   - **P2.1** `pm.Enable` for replication — Task 5 ✓
   - **P2.2** storage location compare — already landed in PR 1 (commit `c0eb6c05c`) ✓

3. **Type / naming consistency**

   - `ReplicationPhase` (Go field) ↔ `--replicationPhase` (CLI flag) — camelCase per existing convention (`--restoreName`, `--pitrRestoredTs`).
   - `--replication-storage-phase` etc. — kebab-case (BR CLI convention).
   - Constants `replicationStatusSubPrefix` / `replicationPiTRConcurrency` lowercase package-private (BR call detail; not exported).

4. **No leftover stubs or TODOs after Task 6**

   - `buildPiTRBaseArgs` removed ✓
   - "PR 2 will complete the integration" comments removed ✓

5. **Commit hygiene**

   - One logical change per commit (Tasks 1–7 produce 7 commits, plus any documentation updates).
   - No Claude attribution footers (per repo conventions).
   - Each commit message describes *why*, not just *what*.

## Notes / deferred follow-ups

- **Job watch on Restore controller** (P1.3): not in this PR; phase-1 Job completion is detected on the next Restore CR resync (≤ 30 s). If business signals a tighter SLA, add a Job informer in a follow-up.
- **TidbCluster doesn't watch Restore CRs**: minor latency between Restore creation and TiKV reconcile firing the GC override write. Standard PiTR has the same behavior; replication doesn't make it worse. Follow-up only if profiling shows this is a real bottleneck.
- **Full §10 integration scenarios**: 9 e2e scenarios require a live tikv + BR. Will be exercised manually against a staging cluster after merge.
- **`compactIsTerminal` unknown-state warning** (PR 1 review I3): already landed in commit `2ed25e209`. Listed here for completeness.
