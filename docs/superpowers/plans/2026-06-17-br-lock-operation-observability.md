# BR Lock Operation Observability Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 BR operation ID 和锁冲突 blocker 信息从 BR 子进程输出透传到 Backup/Restore CR status。

**Architecture:** BR 继续生成 operation ID。backup-manager 在 Backup/Restore Job Pod 内实时读取 BR stdout/stderr，解析 operation started 与 lock conflict 事件，并通过现有 status updater 写回 CR。CRD 增加一组共享语义的 status 字段：最近 10 条 BR operation，以及最近一次 lock blocker 诊断。

**Tech Stack:** Go, Kubernetes CRD/status, tidb-operator backup-manager, BR structured log, controller-gen/openapi-gen.

---

## Spec Source

- Design doc: `docs/design-proposals/2026-06-16-br-lock-operation-observability.md`
- Upstream TiDB PR to confirm during execution: `pingcap/tidb#69231`

## File Map

- Modify `pkg/apis/pingcap/v1alpha1/types.go`
  - Add shared BR observability status structs.
  - Add `brOperations` and `lockBlocker` to `BackupStatus` and `RestoreStatus`.
- Modify generated files after API changes:
  - `pkg/apis/pingcap/v1alpha1/zz_generated.deepcopy.go`
  - `pkg/apis/pingcap/v1alpha1/openapi_generated.go`
  - `manifests/crd/v1/pingcap.com_backups.yaml`
  - `manifests/crd/v1/pingcap.com_restores.yaml`
  - `manifests/crd.yaml`
- Modify `pkg/controller/backup_status_updater.go` and `pkg/controller/restore_status_updater.go`
  - Carry BR observability updates through `BackupUpdateStatus` and `RestoreUpdateStatus`.
  - Append/de-duplicate/cap `brOperations` at 10.
  - Set or clear `lockBlocker`.
- Modify tests:
  - `pkg/controller/backup_status_updater_test.go`
  - `pkg/controller/restore_status_updater_test.go`
- Create `cmd/backup-manager/app/brlog/`
  - Parser and observer for BR structured log lines.
  - Unit tests with fixtures copied from the confirmed upstream PR logs.
- Modify `cmd/backup-manager/app/util/util.go`
  - Replace block-at-end stderr consumption with reusable real-time line streaming helper.
- Modify BR execution paths:
  - `cmd/backup-manager/app/backup/backup.go`
  - `cmd/backup-manager/app/backup/manager.go`
  - `cmd/backup-manager/app/restore/restore.go`
  - `cmd/backup-manager/app/restore/manager.go`
- Add focused tests in `cmd/backup-manager/app/brlog/parser_test.go`,
  `cmd/backup-manager/app/util/util_test.go`, and the controller status updater tests.

---

### Task 1: Confirm BR Log Contract From Upstream PR

**Files:**
- Read: upstream PR `pingcap/tidb#69231`
- Modify: `docs/design-proposals/2026-06-16-br-lock-operation-observability.md`

This task is a hard prerequisite for implementation. The parser contract must be derived from the
current PR head and its actual BR log/error output, not from prior discussion, cached summaries, or
guessed field names.

- [x] **Step 1: Fetch PR metadata and diff**

Run:

```bash
gh pr view 69231 --repo pingcap/tidb --json title,headRefOid,baseRefName,files
gh pr diff 69231 --repo pingcap/tidb --name-only
gh pr diff 69231 --repo pingcap/tidb
```

Expected:

- The diff includes BR operation metadata changes in BR/objstore lock-related files.
- The diff contains concrete log message text or structured fields for operation started and lock conflict events.

- [x] **Step 2: Record the exact parser contract**

Add a short `## Execution Notes` section to `docs/design-proposals/2026-06-16-br-lock-operation-observability.md` with:

- target TiDB PR head commit SHA
- operation started event marker
- operation started fields required by operator
- lock conflict event marker
- lock conflict fields required by operator
- whether the fields are emitted on stdout, stderr, or both if the PR makes that clear
- at least one real operation-started log fixture copied from the PR diff or generated from the PR's
  test/log output
- at least one real lock-conflict log or failure-output fixture copied from the PR diff or generated
  from the PR's test/log output

Expected:

- The design doc still says final CRD/log field names are confirmed at execution time.
- It now contains the concrete evidence used by the parser implementation.
- Parser constants and unit-test fixtures in later tasks map directly to this recorded evidence.

- [x] **Step 3: Stop if BR does not expose both required events**

If the PR does not expose operation started and lock conflict metadata in BR stdout/stderr structured
logs, stop implementation and update the design doc with the missing upstream contract. Do not invent
parser fields in operator.

Task 1 result: `pingcap/tidb#69231` head `955eb2b94928a5920fcca0a53308dbf2ba957e7e`
exposes `BR operation started` and lock-conflict metadata through `pkg/objstore.LockConflictLogFields`.
Execution notes in the design doc record the exact markers and fields. BR defaults to
`--log-format=text`, so in-scope backup-manager BR commands must force JSON log output before the
JSON parser can be reliable.

---

### Task 2: Add API Status Types

**Files:**
- Modify: `pkg/apis/pingcap/v1alpha1/types.go`
- Generated by Task 8: deepcopy, OpenAPI, CRDs

- [ ] **Step 1: Add shared status structs**

Add the following types near `BackupStatus` / `RestoreStatus` definitions:

```go
// BROperation records one observed BR process execution for this CR.
type BROperation struct {
	OperationID string `json:"operationID,omitempty"`
	StartedAt   *metav1.Time `json:"startedAt,omitempty"`
	Command     string `json:"command,omitempty"`
	ObservedAt  metav1.Time `json:"observedAt,omitempty"`
}

// BRLockBlocker records the most recent BR external storage lock blocker diagnosis.
type BRLockBlocker struct {
	LockPath            string `json:"lockPath,omitempty"`
	RemoteOperationID   string `json:"remoteOperationID,omitempty"`
	RemoteStartedAt     *metav1.Time `json:"remoteStartedAt,omitempty"`
	ResourceType        string `json:"resourceType,omitempty"`
	ObservedAt          metav1.Time `json:"observedAt,omitempty"`
}
```

Keep the CRD status schema intentionally small. Do not expose parser inputs or convenience
debug fields such as raw hint strings, log source, condition message text, pod name, job name,
restore ID, local failed operation ID, or restore/backup phase in this first phase. The status
fields must be enough to answer two questions only:

- which BR operations this CR recently started
- which remote lock owner currently blocks or most recently blocked this CR

- [ ] **Step 2: Add fields to BackupStatus**

Add to `BackupStatus`:

```go
// BROperations records recently observed BR process executions owned by this Backup.
// +nullable
BROperations []BROperation `json:"brOperations,omitempty"`
// LockBlocker records the most recent BR external storage lock blocker diagnosis.
// +nullable
LockBlocker *BRLockBlocker `json:"lockBlocker,omitempty"`
```

- [ ] **Step 3: Add fields to RestoreStatus**

Add the same fields to `RestoreStatus`:

```go
// BROperations records recently observed BR process executions owned by this Restore.
// +nullable
BROperations []BROperation `json:"brOperations,omitempty"`
// LockBlocker records the most recent BR external storage lock blocker diagnosis.
// +nullable
LockBlocker *BRLockBlocker `json:"lockBlocker,omitempty"`
```

- [ ] **Step 4: Run API generation**

Run:

```bash
./hack/update-crd.sh
./hack/update-openapi-spec.sh
go generate ./pkg/apis/pingcap/v1alpha1
```

Expected:

- `zz_generated.deepcopy.go` contains deepcopy methods for the new structs.
- backup and restore CRDs include `status.brOperations` and `status.lockBlocker`.
- No CompactBackup CRD field is generated in this phase.

---

### Task 3: Add Status Updater Semantics

**Files:**
- Modify: `pkg/controller/backup_status_updater.go`
- Modify: `pkg/controller/restore_status_updater.go`
- Test: `pkg/controller/backup_status_updater_test.go`
- Test: `pkg/controller/restore_status_updater_test.go`

- [ ] **Step 1: Extend update status structs**

Add to `BackupUpdateStatus` and `RestoreUpdateStatus`:

```go
// BROperation appends or refreshes one observed BR operation.
BROperation *v1alpha1.BROperation
// LockBlocker sets the latest lock blocker diagnosis when non-nil.
LockBlocker *v1alpha1.BRLockBlocker
// ClearLockBlocker clears stale lock blocker diagnosis when true.
ClearLockBlocker *bool
```

- [ ] **Step 2: Add shared update helpers**

Add helper functions in `pkg/controller/backup_status_updater.go` or a new small file `pkg/controller/br_observability_status.go`:

```go
const maxBROperations = 10

func updateBROperations(existing []v1alpha1.BROperation, observed *v1alpha1.BROperation) ([]v1alpha1.BROperation, bool) {
	if observed == nil || observed.OperationID == "" {
		return existing, false
	}
	next := make([]v1alpha1.BROperation, 0, len(existing)+1)
	updated := false
	next = append(next, *observed)
	for _, item := range existing {
		if item.OperationID == observed.OperationID {
			updated = true
			continue
		}
		next = append(next, item)
	}
	if len(next) > maxBROperations {
		next = next[:maxBROperations]
		updated = true
	}
	if !updated && len(next) == len(existing)+1 {
		return next, true
	}
	return next, !reflect.DeepEqual(existing, next)
}

func updateBRLockBlocker(existing *v1alpha1.BRLockBlocker, blocker *v1alpha1.BRLockBlocker, clear *bool) (*v1alpha1.BRLockBlocker, bool) {
	if clear != nil && *clear {
		if existing == nil {
			return nil, false
		}
		return nil, true
	}
	if blocker == nil {
		return existing, false
	}
	if reflect.DeepEqual(existing, blocker) {
		return existing, false
	}
	copied := blocker.DeepCopy()
	return copied, true
}
```

If this file is new, include imports for `reflect` and the local `v1alpha1` package.

- [ ] **Step 3: Wire helpers into Backup status update**

In `updateBackupStatus`, after existing scalar/progress/backoff updates, apply:

```go
if operations, updated := updateBROperations(status.BROperations, newStatus.BROperation); updated {
	status.BROperations = operations
	isUpdate = true
}
if blocker, updated := updateBRLockBlocker(status.LockBlocker, newStatus.LockBlocker, newStatus.ClearLockBlocker); updated {
	status.LockBlocker = blocker
	isUpdate = true
}
```

Important: the existing retry branch assigns `isUpdate = updateBackoffRetryStatus(...)`. Do not put the
BR observability helper calls before that assignment unless the retry branch is first changed to OR its
result into `isUpdate`; otherwise an observability-only update can be accidentally lost.

- [ ] **Step 4: Wire helpers into Restore status update**

In `updateRestoreStatus`, apply the same helper calls to `status.BROperations` and `status.LockBlocker`.

- [ ] **Step 5: Add Backup updater tests**

Add tests covering:

- nil observability update changes nothing
- new operation is prepended
- duplicate operation ID refreshes the existing entry instead of appending
- list is capped at 10
- blocker is set
- blocker is cleared when `ClearLockBlocker=true`

Run:

```bash
go test ./pkg/controller -run 'TestUpdateBackupStatus|TestUpdateBROperations|TestUpdateBRLockBlocker'
```

Expected: PASS.

- [ ] **Step 6: Add Restore updater tests**

Mirror the operation and blocker tests for `updateRestoreStatus`.

Run:

```bash
go test ./pkg/controller -run 'TestUpdateRestoreStatus|TestUpdateBROperations|TestUpdateBRLockBlocker'
```

Expected: PASS.

---

### Task 4: Add BR Structured Log Parser

**Files:**
- Create: `cmd/backup-manager/app/brlog/parser.go`
- Create: `cmd/backup-manager/app/brlog/parser_test.go`

- [ ] **Step 1: Add parser types**

Create a parser package with:

```go
package brlog

import "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"

type EventType string

const (
	EventNone             EventType = ""
	EventOperationStarted EventType = "operation-started"
	EventLockConflict     EventType = "lock-conflict"
)

type Event struct {
	Type        EventType
	Operation   *v1alpha1.BROperation
	LockBlocker *v1alpha1.BRLockBlocker
}
```

- [ ] **Step 2: Implement JSON structured log parsing**

Implement `ParseLine(line string) Event` using `encoding/json` into `map[string]interface{}`. It must:

- ignore non-JSON lines
- match the operation started marker recorded in Task 1 `Execution Notes`
- match the lock conflict marker recorded in Task 1 `Execution Notes`
- use the `message` field as the canonical PingCAP log JSON message key, with `msg` and `Message`
  accepted only as compatibility fallbacks
- parse RFC3339 timestamps into `metav1.Time`
- parse `operation_started_at` from lock metadata `hint` strings when present and use it as
  `BRLockBlocker.RemoteStartedAt`
- ignore `restore_id` in the lock metadata hint for now; it is useful context for some PiTR cases,
  but is not part of the first-phase CRD status schema
- keep raw hints, raw log messages, source labels, and local failed operation IDs out of the
  returned status structs
- leave missing optional fields empty
- never return a synthetic operation ID

Use constants for the markers and field keys so Task 1 evidence maps to one place in code. Do not add
or rename parser fields unless they are present in the current `pingcap/tidb#69231` evidence recorded
by Task 1 and included in the minimal status structs from Task 2.

- [ ] **Step 3: Add parser tests from PR fixtures**

Copy one operation started log line and one lock conflict log line from Task 1 into test fixtures. Add tests with these exact names:

- `TestParseLineOperationStarted`
- `TestParseLineLockConflict`
- `TestParseLineIgnoresNonJSON`
- `TestParseLineIgnoresUnknownJSON`

Run:

```bash
go test ./cmd/backup-manager/app/brlog
```

Expected: PASS.

---

### Task 5: Stream BR stdout/stderr In Real Time

**Files:**
- Modify: `cmd/backup-manager/app/util/util.go`
- Test: `cmd/backup-manager/app/util/util_test.go`

- [ ] **Step 1: Add a reusable line streaming helper**

Add a helper that reads one stream line-by-line and calls a callback for every line:

```go
func ReadLinesToChannel(reader io.Reader, lineCh chan<- string, errCh chan<- error) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		lineCh <- scanner.Text()
	}
	if err := scanner.Err(); err != nil {
		errCh <- err
		return
	}
	errCh <- nil
}
```

Keep `ReadAllStdErrToChannel` until all old call sites are migrated or remove it if unused after Task 6.

- [ ] **Step 2: Add utility tests**

Add tests that pass a `strings.NewReader("a\nb\n")` and assert both lines arrive before the nil error.

Run:

```bash
go test ./cmd/backup-manager/app/util -run TestReadLinesToChannel
```

Expected: PASS.

---

### Task 6: Wire Parser Into Backup BR Execution

**Files:**
- Modify: `cmd/backup-manager/app/backup/manager.go`
- Modify: `cmd/backup-manager/app/backup/backup.go`
- Test: `cmd/backup-manager/app/brlog/parser_test.go`
- Test: `pkg/controller/backup_status_updater_test.go`

- [ ] **Step 1: Pass status updater to log-truncate BR execution**

Change `doTruncateLogBackup` to accept `statusUpdater controller.BackupConditionUpdaterInterface`, and call it from `truncateLogBackup`:

```go
backupErr := bm.doTruncateLogBackup(ctx, backup, bm.StatusUpdater)
```

Keep other log subcommands unchanged unless Task 1 proves they expose the same BR lock conflict metadata.

Ensure the log-truncate BR command emits terminal JSON logs by appending `--log-format=json` to the
BR args used by this path. The Job already sets `BR_LOG_TO_TERM`; do not add local log-file parsing.

- [ ] **Step 2: Add a BR log observer for Backup**

In `backup.go`, build an observer before running BR:

- on `EventOperationStarted`, call `statusUpdater.Update(backup, nil, &controller.BackupUpdateStatus{BROperation: event.Operation})`
- on `EventLockConflict`, store `event.LockBlocker` in memory for final failure handling
- set `ObservedAt` before updating status if the parser did not already set it
- do not add backup phase, job name, pod name, log source, raw hint, raw message, or local failed
  operation ID metadata to the status update

- [ ] **Step 3: Replace stdout-only loop with stdout/stderr streaming**

Update `brCommandRunWithLogCallback` or introduce `brCommandRunWithObserver` so both stdout and stderr lines:

- are logged to klog
- contribute `[ERROR]` lines to `errMsg`
- invoke the existing volume snapshot callback when provided
- invoke the new BR observability observer when provided

After `cmd.Wait()`:

- if BR succeeds, call `statusUpdater.Update(backup, nil, &controller.BackupUpdateStatus{ClearLockBlocker: ptr.To(true)})`
- if BR fails and a lock blocker candidate was observed, call `statusUpdater.Update(backup, nil, &controller.BackupUpdateStatus{LockBlocker: candidate})`
- if BR fails without a lock blocker candidate, call `statusUpdater.Update(backup, nil, &controller.BackupUpdateStatus{ClearLockBlocker: ptr.To(true)})`
- return the original BR error message

- [ ] **Step 4: Run focused Backup tests**

Run:

```bash
go test ./cmd/backup-manager/app/backup ./pkg/controller ./cmd/backup-manager/app/brlog
```

Expected: PASS.

---

### Task 7: Wire Parser Into Restore BR Execution

**Files:**
- Modify: `cmd/backup-manager/app/restore/restore.go`
- Modify: `cmd/backup-manager/app/restore/manager.go`
- Test: `cmd/backup-manager/app/brlog/parser_test.go`
- Test: `pkg/controller/restore_status_updater_test.go`

- [ ] **Step 1: Add a BR log observer for Restore**

In `restoreData`, create the same observer shape as Backup:

- operation started writes `RestoreUpdateStatus.BROperation`
- lock conflict is held as the current execution's candidate blocker
- success clears stale blocker
- lock-conflict failure writes latest blocker
- non-lock failure clears stale blocker

Ensure in-scope restore/log-restore BR commands emit terminal JSON logs by appending
`--log-format=json` to the BR args used by this path. The Job already sets `BR_LOG_TO_TERM`; do not
add local log-file parsing.

- [ ] **Step 2: Preserve existing restore progress behavior**

Ensure the stdout parser still calls:

```go
ro.updateProgressAccordingToBrLog(line, restore, statusUpdater)
ro.updateResolvedTSForCSB(line, restore, progressStep, statusUpdater)
```

Do not call progress parsing for stderr unless the old code already did so.

- [ ] **Step 3: Respect replication phase status ownership**

The phase-1 replication restore path uses `NewReplicationRestoreStatusUpdater()` and must remain no-op for backup-manager status writes. Phase-2 log-restore uses the real updater and should record BR operations/blockers.

- [ ] **Step 4: Run focused Restore tests**

Run:

```bash
go test ./cmd/backup-manager/app/restore ./cmd/backup-manager/app/cmd ./pkg/controller ./cmd/backup-manager/app/brlog
```

Expected: PASS.

---

### Task 8: Regenerate And Verify API Artifacts

**Files:**
- Generated CRDs/OpenAPI/deepcopy files

- [ ] **Step 1: Run generation**

Run:

```bash
go generate ./pkg/apis/pingcap/v1alpha1
./hack/update-crd.sh
./hack/update-openapi-spec.sh
```

Expected:

- Generated files are updated deterministically.
- Backup/Restore CRDs include the new status fields.
- CompactBackup CRD is unchanged for this feature.

- [ ] **Step 2: Run verification**

Run:

```bash
./hack/verify-crd.sh
./hack/verify-openapi-spec.sh
go test ./pkg/controller ./cmd/backup-manager/app/brlog ./cmd/backup-manager/app/util ./cmd/backup-manager/app/backup ./cmd/backup-manager/app/restore ./cmd/backup-manager/app/cmd
```

Expected: PASS.

---

### Task 9: Final Review Checklist

- [ ] Confirm no controller-manager `pods/log` RBAC was added.
- [ ] Confirm no remote lock JSON read path was added.
- [ ] Confirm `CompactBackup` status was not changed.
- [ ] Confirm `brOperations` caps at 10 and de-duplicates by operation ID.
- [ ] Confirm `lockBlocker` is latest-only and clears on success/non-lock failure.
- [ ] Confirm status update failures only log and do not replace BR execution result.
- [ ] Confirm the design doc still matches implementation after Task 1 field-contract evidence.
