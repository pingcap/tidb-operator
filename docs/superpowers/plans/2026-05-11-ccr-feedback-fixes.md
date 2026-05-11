# CCR Feedback Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Address verified CCR test feedback for sharded compact and replication restore flags without changing non-sharded compact behavior.

**Architecture:** Compact changes are scoped to `CompactBackup.spec.mode == "sharded"` only. Non-sharded compact must keep requiring `endTs` and must not receive the new compact tuning/checkpoint flags. Restore replication flag changes remain gated by `Options.ReplicationPhase > 0`, so standard restore behavior is unchanged.

**Tech Stack:** Go, Cobra backup-manager commands, TiDB Operator CRD controllers, table-driven unit tests.

---

## Scope

Implement:

- Sharded compact gets `--cal-shift-ts`.
- Sharded compact gets `--physical-file-cache-capacity=150G`.
- Sharded compact may omit `spec.endTs`; omitted `endTs` maps to `--crr-checkpoint-prefix <storage prefix>`.
- Non-sharded compact still rejects empty `spec.endTs`.
- Non-sharded compact does not get `--cal-shift-ts`, `--physical-file-cache-capacity`, or `--crr-checkpoint-prefix`.
- Compact `--shard` passed to tikv-ctl uses numerator `1..N`; Kubernetes `JOB_COMPLETION_INDEX` remains validated as `0..N-1`.
- Replication restore BR flags include `--retain-latest-mvcc-version`.
- Replication restore status sub-prefix changes from `ccr` to `crr-checkpoint`.

Do not implement:

- Default machine type `r8g.8xlarge` / default `-N=64`. `CompactBackup.spec.concurrency` remains the user-controlled source for `-N`.

---

## File-Level Scope

| File | Responsibility |
|------|----------------|
| `cmd/backup-manager/app/compact/manager.go` | Build tikv-ctl compact args; scope new compact flags to sharded mode. |
| `cmd/backup-manager/app/compact/manager_sharded_test.go` | Verify compact args for default, sharded, and sharded checkpoint modes. |
| `cmd/backup-manager/app/compact/options/options.go` | Parse/validate compact options; allow unset `UntilTS` only when sharded. |
| `cmd/backup-manager/app/compact/options/options_sharded_test.go` | Verify sharded option parsing and non-sharded unset `UntilTS` rejection. |
| `pkg/controller/compactbackup/compact_backup_controller.go` | Validate CompactBackup CRs before Job creation. |
| `pkg/controller/compactbackup/compact_backup_controller_sharded_test.go` | Verify `endTs` is optional only in sharded mode. |
| `cmd/backup-manager/app/restore/restore.go` | Build replication restore BR args. |
| `cmd/backup-manager/app/restore/restore_test.go` | Verify replication restore BR args. |
| `docs/superpowers/specs/2026-04-23-replication-restore-design.md` | Keep branch-local design constants consistent. |
| `docs/superpowers/plans/2026-04-29-replication-restore-pr2.md` | Keep branch-local plan examples consistent. |

---

## Task 1: Scope Compact Tuning And Checkpoint Args To Sharded Mode

**Files:**
- Modify: `cmd/backup-manager/app/compact/manager_sharded_test.go`
- Modify: `cmd/backup-manager/app/compact/manager.go`

- [ ] **Step 1: Update default-mode test to preserve non-sharded args**

In `cmd/backup-manager/app/compact/manager_sharded_test.go`, update `TestBuildCompactArgsDefaultMode` so `want` is exactly:

```go
want := []string{
	"--log-level", "INFO",
	"--log-format", "json",
	"compact-log-backup",
	"--storage-base64", "storage-base64",
	"--from", "11",
	"-N", "4",
	"--until", "22",
}
```

- [ ] **Step 2: Update sharded expected cache value**

In `TestBuildCompactArgsShardedMode`, change:

```go
"--physical-file-cache-capacity", "128G",
```

to:

```go
"--physical-file-cache-capacity", "150G",
```

In `TestBuildCompactArgsCRRModeShardedUsesCheckpointPrefix`, make the same `128G` to `150G` replacement.

- [ ] **Step 3: Remove wrong non-sharded checkpoint test**

Delete `TestBuildCompactArgsCRRModeUsesCheckpointPrefix` from `cmd/backup-manager/app/compact/manager_sharded_test.go`. Non-sharded empty `endTs` is invalid, so this test documents behavior we explicitly do not want.

- [ ] **Step 4: Run tests to verify failure**

```bash
go test ./cmd/backup-manager/app/compact/... -run 'TestBuildCompactArgs(DefaultMode|ShardedMode|CRRModeShardedUsesCheckpointPrefix)' -count=1
```

Expected: fail because implementation still appends compact tuning flags in default mode and still uses `128G`.

- [ ] **Step 5: Change argument construction**

In `cmd/backup-manager/app/compact/manager.go`, inside `buildCompactArgs`, remove these entries from the base `args` list:

```go
"--cal-shift-ts",
"--physical-file-cache-capacity",
"128G",
```

Then replace the `UntilTS` / `Sharded` block with:

```go
if cm.options.Sharded {
	args = append(args,
		"--cal-shift-ts",
		"--physical-file-cache-capacity",
		"150G",
	)
}
if cm.options.UntilTS != 0 {
	args = append(args, "--until", strconv.FormatUint(cm.options.UntilTS, 10))
} else if cm.options.Sharded {
	args = append(args, "--crr-checkpoint-prefix", cm.checkpointPrefix())
}
if cm.options.Sharded {
	// --shard tells tikv-ctl this pod's slice of the keyspace partition.
	// --minimal-compaction-size=0 disables the small-segment skip so each
	// shard compacts its full slice instead of discarding fragments.
	args = append(args,
		"--shard",
		strconv.Itoa(cm.options.ShardIndex)+"/"+strconv.Itoa(cm.options.ShardCount),
		"--minimal-compaction-size",
		"0",
	)
}
```

- [ ] **Step 6: Run tests to verify pass**

```bash
go test ./cmd/backup-manager/app/compact/... -run 'TestBuildCompactArgs(DefaultMode|ShardedMode|CRRModeShardedUsesCheckpointPrefix)' -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add cmd/backup-manager/app/compact/manager.go cmd/backup-manager/app/compact/manager_sharded_test.go
git commit -m "fix(compact): scope tuning flags to sharded mode"
```

---

## Task 2: Convert Compact `--shard` To 1-Based tikv-ctl Syntax

**Files:**
- Modify: `cmd/backup-manager/app/compact/manager_sharded_test.go`
- Modify: `cmd/backup-manager/app/compact/manager.go`
- Modify: `cmd/backup-manager/app/compact/options/options.go`
- Modify: `cmd/backup-manager/app/compact/options/options_sharded_test.go`

**Important distinction:** Kubernetes Indexed Jobs expose `JOB_COMPLETION_INDEX` as `0..N-1`. Keep parsing and validation on that runtime value. Only the tikv-ctl `--shard` numerator is converted to `index+1`, so the external CLI receives `1..N`.

- [ ] **Step 1: Update sharded arg tests for 1-based output**

In `TestBuildCompactArgsShardedMode`, keep:

```go
ShardIndex:  1,
ShardCount:  3,
```

and change:

```go
"--shard", "1/3",
```

to:

```go
"--shard", "2/3",
```

In `TestBuildCompactArgsCRRModeShardedUsesCheckpointPrefix`, make the same expected `--shard` change.

- [ ] **Step 2: Add boundary test for Kubernetes index 0**

Append this test to `cmd/backup-manager/app/compact/manager_sharded_test.go`:

```go
func TestBuildCompactArgsShardedModeConvertsKubernetesIndexToOneBasedShard(t *testing.T) {
	manager := &Manager{
		compact: &v1alpha1.CompactBackup{},
		options: options.CompactOpts{
			FromTS:      11,
			UntilTS:     22,
			Concurrency: 4,
			Sharded:     true,
			ShardIndex:  0,
			ShardCount:  3,
		},
	}

	args := manager.buildCompactArgs("storage-base64")
	assertStringSliceContainsPair(t, args, "--shard", "1/3")
}
```

Add this helper near the existing `assertStringSliceEqual` helper:

```go
func assertStringSliceContainsPair(t *testing.T, got []string, key, value string) {
	t.Helper()
	for i := 0; i+1 < len(got); i++ {
		if got[i] == key && got[i+1] == value {
			return
		}
	}
	t.Fatalf("expected args to contain %q %q, got %#v", key, value, got)
}
```

- [ ] **Step 3: Update validation test name and expected wording**

In `cmd/backup-manager/app/compact/options/options_sharded_test.go`, rename:

```go
func TestCompactOptsVerifyRejectsInvalidShardedFields(t *testing.T) {
```

to:

```go
func TestCompactOptsVerifyRejectsInvalidKubernetesShardIndex(t *testing.T) {
```

For the negative and out-of-range shard index cases, change:

```go
wantErr: "shard-index",
```

to:

```go
wantErr: "kubernetes shard-index",
```

- [ ] **Step 4: Run tests to verify failure**

```bash
go test ./cmd/backup-manager/app/compact/... -run 'TestBuildCompactArgsShardedMode|TestBuildCompactArgsCRRModeShardedUsesCheckpointPrefix|TestBuildCompactArgsShardedModeConvertsKubernetesIndexToOneBasedShard|TestCompactOptsVerifyRejectsInvalidKubernetesShardIndex' -count=1
```

Expected: fail because implementation still emits `ShardIndex/ShardCount`, and validation messages still say `shard-index`.

- [ ] **Step 5: Change compact arg builder**

In `cmd/backup-manager/app/compact/manager.go`, change:

```go
strconv.Itoa(cm.options.ShardIndex)+"/"+strconv.Itoa(cm.options.ShardCount),
```

to:

```go
strconv.Itoa(cm.options.ShardIndex+1)+"/"+strconv.Itoa(cm.options.ShardCount),
```

- [ ] **Step 6: Change validation error messages**

In `cmd/backup-manager/app/compact/options/options.go`, change:

```go
return errors.Errorf("shard-index %d must be in range [0, %d)", c.ShardIndex, c.ShardCount)
```

to:

```go
return errors.Errorf("kubernetes shard-index %d must be in range [0, %d); tikv-ctl --shard uses 1..%d after conversion", c.ShardIndex, c.ShardCount, c.ShardCount)
```

In `resolveShardIndex`, change:

```go
return 0, fmt.Errorf("JOB_COMPLETION_INDEX %d out of range [0, %d)", index, shardCount)
```

to:

```go
return 0, fmt.Errorf("JOB_COMPLETION_INDEX %d out of range [0, %d); tikv-ctl --shard uses 1..%d after conversion", index, shardCount, shardCount)
```

- [ ] **Step 7: Run tests to verify pass**

```bash
go test ./cmd/backup-manager/app/compact/... -run 'TestBuildCompactArgsShardedMode|TestBuildCompactArgsCRRModeShardedUsesCheckpointPrefix|TestBuildCompactArgsShardedModeConvertsKubernetesIndexToOneBasedShard|TestCompactOptsVerifyRejectsInvalidKubernetesShardIndex' -count=1
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add cmd/backup-manager/app/compact/manager.go cmd/backup-manager/app/compact/manager_sharded_test.go cmd/backup-manager/app/compact/options/options.go cmd/backup-manager/app/compact/options/options_sharded_test.go
git commit -m "fix(compact): pass one-based shard index to tikv-ctl"
```

---

## Task 3: Allow Empty `endTs` Only For Sharded Compact

**Files:**
- Modify: `pkg/controller/compactbackup/compact_backup_controller_sharded_test.go`
- Modify: `pkg/controller/compactbackup/compact_backup_controller.go`
- Modify: `cmd/backup-manager/app/compact/options/options_sharded_test.go`
- Modify: `cmd/backup-manager/app/compact/options/options.go`

- [ ] **Step 1: Add controller validation tests**

Append these tests near the existing validation tests in `pkg/controller/compactbackup/compact_backup_controller_sharded_test.go`:

```go
func TestValidateAllowsEmptyEndTsOnlyForShardedCCRCheckpointMode(t *testing.T) {
	c := newTestController(t)
	compact := newCompactBackupForTest()
	shardCount := int32(3)
	compact.Spec.Mode = v1alpha1.CompactModeSharded
	compact.Spec.ShardCount = &shardCount
	compact.Spec.EndTs = ""

	err := c.validate(compact)
	if err != nil {
		t.Fatalf("expected empty endTs to be accepted in sharded mode, got %v", err)
	}
}

func TestValidateRejectsEmptyEndTsForNonShardedMode(t *testing.T) {
	c := newTestController(t)
	compact := newCompactBackupForTest()
	compact.Spec.Mode = ""
	compact.Spec.ShardCount = nil
	compact.Spec.EndTs = ""

	err := c.validate(compact)
	if err == nil {
		t.Fatal("expected empty endTs to be rejected in non-sharded mode")
	}
	if !strings.Contains(err.Error(), "end-ts") {
		t.Fatalf("expected end-ts validation error, got %v", err)
	}
}
```

- [ ] **Step 2: Add backup-manager option validation test**

Append this test to `cmd/backup-manager/app/compact/options/options_sharded_test.go`:

```go
func TestCompactOptsVerifyAllowsUnsetUntilTSOnlyWhenSharded(t *testing.T) {
	sharded := CompactOpts{
		FromTS:      1,
		UntilTS:     0,
		Concurrency: 1,
		Sharded:     true,
		ShardIndex:  0,
		ShardCount:  3,
	}
	if err := sharded.Verify(); err != nil {
		t.Fatalf("expected sharded unset UntilTS to be accepted, got %v", err)
	}

	nonSharded := CompactOpts{
		FromTS:      1,
		UntilTS:     0,
		Concurrency: 1,
		Sharded:     false,
	}
	err := nonSharded.Verify()
	if err == nil {
		t.Fatal("expected non-sharded unset UntilTS to be rejected")
	}
	if !strings.Contains(err.Error(), "until-ts must be set") {
		t.Fatalf("expected until-ts validation error, got %v", err)
	}
}
```

- [ ] **Step 3: Run tests to verify failure**

```bash
go test ./pkg/controller/compactbackup/... -run 'TestValidateAllowsEmptyEndTsOnlyForShardedCCRCheckpointMode|TestValidateRejectsEmptyEndTsForNonShardedMode' -count=1
go test ./cmd/backup-manager/app/compact/options/... -run TestCompactOptsVerifyAllowsUnsetUntilTSOnlyWhenSharded -count=1
```

Expected: first command fails because controller still rejects every empty `endTs`; second command fails because options currently allow unset `UntilTS` in non-sharded mode.

- [ ] **Step 4: Change controller validation**

In `pkg/controller/compactbackup/compact_backup_controller.go`, change:

```go
if spec.EndTs == "" {
	return errors.NewNoStackError("end-ts must be set")
}
```

to:

```go
if spec.EndTs == "" && spec.Mode != v1alpha1.CompactModeSharded {
	return errors.NewNoStackError("end-ts must be set when mode is not sharded")
}
```

- [ ] **Step 5: Change backup-manager option validation**

In `cmd/backup-manager/app/compact/options/options.go`, replace:

```go
// UntilTS unset signals CCR mode: tikv-ctl reads the until-ts from the
// log-backup global checkpoint via --crr-checkpoint-prefix, so leaving
// EndTs blank in the CR is valid. When the user does provide UntilTS,
// it must still be a sane upper bound.
if c.UntilTS != untilTSUnset && c.UntilTS < c.FromTS {
	return errors.Errorf("until-ts %d must be greater than from-ts %d", c.UntilTS, c.FromTS)
}
```

with:

```go
// UntilTS unset is valid only in sharded CCR checkpoint mode: tikv-ctl
// reads the until-ts from the log-backup global checkpoint via
// --crr-checkpoint-prefix. Non-sharded compact keeps the existing
// requirement that EndTs/UntilTS must be set explicitly.
if c.UntilTS == untilTSUnset {
	if !c.Sharded {
		return errors.New("until-ts must be set")
	}
} else if c.UntilTS < c.FromTS {
	return errors.Errorf("until-ts %d must be greater than from-ts %d", c.UntilTS, c.FromTS)
}
```

- [ ] **Step 6: Run tests to verify pass**

```bash
go test ./pkg/controller/compactbackup/... -run 'TestValidateAllowsEmptyEndTsOnlyForShardedCCRCheckpointMode|TestValidateRejectsEmptyEndTsForNonShardedMode' -count=1
go test ./cmd/backup-manager/app/compact/options/... -run TestCompactOptsVerifyAllowsUnsetUntilTSOnlyWhenSharded -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add pkg/controller/compactbackup/compact_backup_controller.go pkg/controller/compactbackup/compact_backup_controller_sharded_test.go cmd/backup-manager/app/compact/options/options.go cmd/backup-manager/app/compact/options/options_sharded_test.go
git commit -m "fix(compact): allow empty endTs only for sharded mode"
```

---

## Task 4: Add Restore `--retain-latest-mvcc-version` And Rename Status Prefix

**Files:**
- Modify: `cmd/backup-manager/app/restore/restore_test.go`
- Modify: `cmd/backup-manager/app/restore/restore.go`

- [ ] **Step 1: Update failing restore flag tests**

In `cmd/backup-manager/app/restore/restore_test.go`, update `TestReplicationBRFlags`.

For phase 1, replace:

```go
{phase: 1, want: []string{
	"--replication-storage-phase=1",
	"--replication-status-sub-prefix=ccr",
	"--pitr-concurrency=1024",
}},
```

with:

```go
{phase: 1, want: []string{
	"--replication-storage-phase=1",
	"--replication-status-sub-prefix=crr-checkpoint",
	"--pitr-concurrency=1024",
	"--retain-latest-mvcc-version",
}},
```

For phase 2, replace:

```go
{phase: 2, want: []string{
	"--replication-storage-phase=2",
	"--replication-status-sub-prefix=ccr",
	"--pitr-concurrency=1024",
}},
```

with:

```go
{phase: 2, want: []string{
	"--replication-storage-phase=2",
	"--replication-status-sub-prefix=crr-checkpoint",
	"--pitr-concurrency=1024",
	"--retain-latest-mvcc-version",
}},
```

- [ ] **Step 2: Run test to verify failure**

```bash
go test ./cmd/backup-manager/app/restore/... -run TestReplicationBRFlags -count=1
```

Expected: fail because implementation still emits `ccr` and lacks `--retain-latest-mvcc-version`.

- [ ] **Step 3: Change implementation**

In `cmd/backup-manager/app/restore/restore.go`, change:

```go
replicationStatusSubPrefix = "ccr"
```

to:

```go
replicationStatusSubPrefix = "crr-checkpoint"
```

Then change `replicationBRFlags` from:

```go
return []string{
	fmt.Sprintf("--replication-storage-phase=%d", phase),
	fmt.Sprintf("--replication-status-sub-prefix=%s", replicationStatusSubPrefix),
	fmt.Sprintf("--pitr-concurrency=%d", replicationPiTRConcurrency),
}
```

to:

```go
return []string{
	fmt.Sprintf("--replication-storage-phase=%d", phase),
	fmt.Sprintf("--replication-status-sub-prefix=%s", replicationStatusSubPrefix),
	fmt.Sprintf("--pitr-concurrency=%d", replicationPiTRConcurrency),
	"--retain-latest-mvcc-version",
}
```

- [ ] **Step 4: Run test to verify pass**

```bash
go test ./cmd/backup-manager/app/restore/... -run TestReplicationBRFlags -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/backup-manager/app/restore/restore.go cmd/backup-manager/app/restore/restore_test.go
git commit -m "fix(replication-restore): update BR replication flags"
```

---

## Task 5: Update Branch-Local Docs For New Restore Prefix And Retain Flag

**Files:**
- Modify: `docs/superpowers/specs/2026-04-23-replication-restore-design.md`
- Modify: `docs/superpowers/plans/2026-04-29-replication-restore-pr2.md`

- [ ] **Step 1: Update design doc constants and BR flag examples**

In `docs/superpowers/specs/2026-04-23-replication-restore-design.md`, replace:

```text
replicationStatusSubPrefix = "ccr"
```

with:

```text
replicationStatusSubPrefix = "crr-checkpoint"
```

Replace each:

```text
--replication-status-sub-prefix=ccr
```

with:

```text
--replication-status-sub-prefix=crr-checkpoint
```

Where the BR replication flag list includes `--pitr-concurrency=1024`, add:

```text
--retain-latest-mvcc-version
```

If prose says “three BR flags”, change it to “four BR flags”.

- [ ] **Step 2: Update implementation plan examples**

In `docs/superpowers/plans/2026-04-29-replication-restore-pr2.md`, apply the same replacements:

```text
--replication-status-sub-prefix=ccr
```

to:

```text
--replication-status-sub-prefix=crr-checkpoint
```

and add:

```text
--retain-latest-mvcc-version
```

after `--pitr-concurrency=1024` in replication flag lists.

- [ ] **Step 3: Verify no stale restore prefix remains**

```bash
rg -n -- '--replication-status-sub-prefix=ccr|replicationStatusSubPrefix = "ccr"|three .*BR flag' docs/superpowers/specs/2026-04-23-replication-restore-design.md docs/superpowers/plans/2026-04-29-replication-restore-pr2.md
```

Expected: no output.

- [ ] **Step 4: Commit**

```bash
git add docs/superpowers/specs/2026-04-23-replication-restore-design.md docs/superpowers/plans/2026-04-29-replication-restore-pr2.md
git commit -m "docs(replication-restore): update BR flag defaults"
```

---

## Task 6: Final Verification

**Files:**
- No source edits unless tests expose a regression.

- [ ] **Step 1: Run focused compact tests**

```bash
go test ./cmd/backup-manager/app/compact/... ./pkg/controller/compactbackup/... -count=1
```

Expected: PASS.

- [ ] **Step 2: Run focused restore tests**

```bash
go test ./cmd/backup-manager/app/restore/... ./cmd/backup-manager/app/cmd/... ./pkg/backup/restore/... ./pkg/controller/restore/... -count=1
```

Expected: PASS.

- [ ] **Step 3: Check diff hygiene**

```bash
git diff --check
git diff --stat origin/master...HEAD
```

Expected:

- `git diff --check` prints no whitespace errors.
- No change sets compact default concurrency to `64`.
- No change adds a machine-type default such as `r8g.8xlarge`.
- Non-sharded compact tests still expect `--until` and do not expect new sharded-only compact flags.

---

## Self-Review

- Spec coverage: covers verified feedback items 2, 3, 4, 5 and the sharded-only `endTs` optional behavior; explicitly excludes item 6.
- Placeholder scan: no TBD/TODO/fill-in instructions remain.
- Type consistency: uses existing `CompactOpts.ShardIndex`, `CompactOpts.ShardCount`, `replicationBRFlags`, `replicationStatusSubPrefix`, and `Controller.validate` names exactly as present in the branch.
