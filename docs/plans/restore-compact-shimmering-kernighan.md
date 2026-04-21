# CompactBackup Sharded Mode Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `CompactBackup.spec.mode=sharded` support that creates a Kubernetes Indexed Job and passes shard-specific `tikv-ctl compact-log-backup` arguments, without changing any Restore behavior or the default non-sharded CompactBackup path.

**Architecture:** Keep the existing CompactBackup reconcile flow and backup-manager compact flow intact, then add one explicit `mode=sharded` branch in each layer. The controller owns CR validation, Kubernetes version gating, Indexed Job construction, and informational shard-status propagation; backup-manager owns shard env parsing and `tikv-ctl` argument construction. Default mode continues to use the current single-Job behavior unchanged.

**Tech Stack:** Go, Kubernetes `batch/v1` Indexed Job, client-go fake clients/informers, generated deepcopy/openapi/CRD artifacts, unit tests under `pkg/controller/compactbackup` and `cmd/backup-manager/app/compact`.

---

## Scope and Guardrails

- Only implement CompactBackup sharded mode.
- Do not modify Restore API, Restore controller, Restore status transforms, or Restore backup-manager code.
- Preserve the current default CompactBackup behavior when `spec.mode` is empty.
- Treat `status.completedIndexes` and `status.failedIndexes` as informational mirrors of Job status only; do not use them for controller decisions.
- In sharded mode, treat missing, invalid, or out-of-range `JOB_COMPLETION_INDEX` as a hard error.
- Do not use `make generate`; regenerate only the minimal API/codegen closure needed for this feature and verify the resulting diff.

## File Structure

**Modify:**
- `pkg/apis/pingcap/v1alpha1/types.go`
- `pkg/apis/pingcap/v1alpha1/zz_generated.deepcopy.go`
- `pkg/apis/pingcap/v1alpha1/openapi_generated.go`
- `manifests/crd.yaml`
- `pkg/controller/compactbackup/compact_backup_controller.go`
- `pkg/controller/compact_status_updater.go`
- `cmd/backup-manager/app/compact/options/options.go`
- `cmd/backup-manager/app/compact/manager.go`
- `cmd/backup-manager/app/cmd/compact.go`

**Create:**
- `pkg/apis/pingcap/v1alpha1/types_compact_sharded_test.go`
- `pkg/controller/compactbackup/server_version.go`
- `pkg/controller/compactbackup/server_version_test.go`
- `pkg/controller/compactbackup/testhelpers_test.go`
- `pkg/controller/compactbackup/compact_backup_controller_sharded_test.go`
- `cmd/backup-manager/app/compact/options/options_sharded_test.go`
- `cmd/backup-manager/app/compact/manager_sharded_test.go`

**Explicitly not in this plan:**
- `docs/design-proposals/...`
- Any `pkg/controller/restore/...`
- Any `cmd/backup-manager/app/restore/...`

## Task 1: Add CompactBackup API Fields and Regenerate Minimal API Artifacts

**Files:**
- Modify: `pkg/apis/pingcap/v1alpha1/types.go`
- Regenerate: `pkg/apis/pingcap/v1alpha1/zz_generated.deepcopy.go`
- Regenerate: `pkg/apis/pingcap/v1alpha1/openapi_generated.go`
- Regenerate: `manifests/crd.yaml`
- Test: `pkg/apis/pingcap/v1alpha1/types_compact_sharded_test.go`

- [ ] **Step 1: Write failing API tests**

Create `pkg/apis/pingcap/v1alpha1/types_compact_sharded_test.go`:

```go
package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

func TestCompactSpecShardedFieldsDeepCopy(t *testing.T) {
	orig := &CompactBackup{
		Spec: CompactSpec{
			Mode:       CompactModeSharded,
			ShardCount: ptr.To[int32](3),
		},
	}

	cp := orig.DeepCopy()
	require.Equal(t, CompactModeSharded, cp.Spec.Mode)
	require.NotNil(t, cp.Spec.ShardCount)
	require.Equal(t, int32(3), *cp.Spec.ShardCount)

	cp.Spec.Mode = CompactMode("")
	*cp.Spec.ShardCount = 9

	require.Equal(t, CompactModeSharded, orig.Spec.Mode)
	require.Equal(t, int32(3), *orig.Spec.ShardCount)
}

func TestCompactStatusShardedFieldsDeepCopy(t *testing.T) {
	orig := &CompactBackup{
		Status: CompactStatus{
			CompletedIndexes: "0-2",
			FailedIndexes:    "4",
		},
	}

	cp := orig.DeepCopy()
	require.Equal(t, "0-2", cp.Status.CompletedIndexes)
	require.Equal(t, "4", cp.Status.FailedIndexes)
}
```

- [ ] **Step 2: Run the focused API tests and confirm they fail**

Run:

```bash
go test ./pkg/apis/pingcap/v1alpha1 -run 'TestCompact(Spec|Status)ShardedFieldsDeepCopy' -v
```

Expected: fail with missing `CompactModeSharded`, `Mode`, `ShardCount`, `CompletedIndexes`, or `FailedIndexes`.

- [ ] **Step 3: Add new API types and fields in `types.go`**

Add to `CompactSpec`:

```go
	// Mode controls how CompactBackup runs.
	// Empty keeps the current single-Job behavior.
	// "sharded" creates a Kubernetes Indexed Job and runs one shard per Pod.
	// +optional
	// +kubebuilder:validation:Enum=sharded
	Mode CompactMode `json:"mode,omitempty"`

	// ShardCount is the total number of shards in sharded mode.
	// It is only used when Mode is "sharded".
	// +optional
	// +kubebuilder:validation:Minimum=1
	ShardCount *int32 `json:"shardCount,omitempty"`
```

Add the mode type near `CompactSpec`:

```go
type CompactMode string

const (
	CompactModeSharded CompactMode = "sharded"
)
```

Add to `CompactStatus`:

```go
	// CompletedIndexes mirrors Job.status.completedIndexes in sharded mode.
	// +optional
	CompletedIndexes string `json:"completedIndexes,omitempty"`

	// FailedIndexes mirrors Job.status.failedIndexes in sharded mode.
	// +optional
	FailedIndexes string `json:"failedIndexes,omitempty"`
```

Runtime validation for "`mode=sharded` requires `shardCount >= 1`" is handled later in the controller; schema only enforces shape and minimum value when the field is present.

- [ ] **Step 4: Regenerate the minimal API/codegen closure**

Run:

```bash
bash hack/update-codegen.sh
bash hack/update-openapi-spec.sh
bash hack/update-crd.sh
```

Expected generated updates:
- `pkg/apis/pingcap/v1alpha1/zz_generated.deepcopy.go`
- `pkg/apis/pingcap/v1alpha1/openapi_generated.go`
- `manifests/crd.yaml`

- [ ] **Step 5: Verify the generated diff is scoped correctly**

Run:

```bash
git diff --name-only -- \
  pkg/apis/pingcap/v1alpha1/types.go \
  pkg/apis/pingcap/v1alpha1/zz_generated.deepcopy.go \
  pkg/apis/pingcap/v1alpha1/openapi_generated.go \
  manifests/crd.yaml \
  pkg/client
```

Expected:
- the four API-related files above are changed
- no unexpected semantic changes appear under `pkg/client`; if extra generated files change, inspect them before proceeding

- [ ] **Step 6: Re-run the focused API tests**

Run:

```bash
go test ./pkg/apis/pingcap/v1alpha1 -run 'TestCompact(Spec|Status)ShardedFieldsDeepCopy' -v
```

Expected: pass.

- [ ] **Step 7: Commit the API slice**

```bash
git add \
  pkg/apis/pingcap/v1alpha1/types.go \
  pkg/apis/pingcap/v1alpha1/zz_generated.deepcopy.go \
  pkg/apis/pingcap/v1alpha1/openapi_generated.go \
  manifests/crd.yaml \
  pkg/apis/pingcap/v1alpha1/types_compact_sharded_test.go
git commit -m "feat(compactbackup): add sharded mode API fields"
```

## Task 2: Add Sharded Controller Validation, K8s Version Gate, and Indexed Job Construction

**Files:**
- Modify: `pkg/controller/compactbackup/compact_backup_controller.go`
- Create: `pkg/controller/compactbackup/server_version.go`
- Test: `pkg/controller/compactbackup/server_version_test.go`
- Test: `pkg/controller/compactbackup/testhelpers_test.go`
- Test: `pkg/controller/compactbackup/compact_backup_controller_sharded_test.go`

- [ ] **Step 1: Write failing server-version helper tests**

Create `pkg/controller/compactbackup/server_version_test.go`:

```go
package compact

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/version"
	discoveryfake "k8s.io/client-go/discovery/fake"
	k8stesting "k8s.io/client-go/testing"
)

func newFakeDiscovery(major, minor string) *discoveryfake.FakeDiscovery {
	return &discoveryfake.FakeDiscovery{
		Fake: &k8stesting.Fake{},
		FakedServerVersion: &version.Info{
			Major: major,
			Minor: minor,
		},
	}
}

func TestRequireShardedJobK8sVersion(t *testing.T) {
	require.NoError(t, requireShardedJobK8sVersion(newFakeDiscovery("1", "29")))
	require.NoError(t, requireShardedJobK8sVersion(newFakeDiscovery("1", "29+")))
	require.NoError(t, requireShardedJobK8sVersion(newFakeDiscovery("1", "30")))

	err := requireShardedJobK8sVersion(newFakeDiscovery("1", "28"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "Kubernetes >= 1.29")
}
```

- [ ] **Step 2: Write failing controller tests for default-mode invariants and sharded Job shape**

Create `pkg/controller/compactbackup/testhelpers_test.go`:

```go
package compact

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newTestController(t *testing.T) *Controller {
	t.Helper()

	deps := controller.NewFakeDependencies()
	stop := make(chan struct{})
	t.Cleanup(func() { close(stop) })

	deps.InformerFactory.Start(stop)
	deps.KubeInformerFactory.Start(stop)
	deps.InformerFactory.WaitForCacheSync(stop)
	deps.KubeInformerFactory.WaitForCacheSync(stop)

	tc := &v1alpha1.TidbCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "tc",
		},
		Spec: v1alpha1.TidbClusterSpec{
			Version: "v8.5.0",
			TiKV: &v1alpha1.TiKVSpec{
				BaseImage: "pingcap/tikv",
			},
		},
	}
	err := deps.InformerFactory.Pingcap().V1alpha1().TidbClusters().Informer().GetIndexer().Add(tc)
	require.NoError(t, err)

	return NewController(deps)
}

func newCompactForTest() *v1alpha1.CompactBackup {
	return &v1alpha1.CompactBackup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "compact-demo",
		},
		Spec: v1alpha1.CompactSpec{
			StartTs:       "100",
			EndTs:         "200",
			Concurrency:   4,
			MaxRetryTimes: 3,
			StorageProvider: v1alpha1.StorageProvider{
				S3: &v1alpha1.S3StorageProvider{
					Bucket: "bucket",
				},
			},
			BR: &v1alpha1.BRConfig{
				Cluster:          "tc",
				ClusterNamespace: "ns",
			},
		},
	}
}
```

Create `pkg/controller/compactbackup/compact_backup_controller_sharded_test.go`:

```go
package compact

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestMakeCompactJobDefaultModeUnchanged(t *testing.T) {
	c := newTestController(t)
	job, _, err := c.makeCompactJob(newCompactForTest())
	require.NoError(t, err)
	require.Nil(t, job.Spec.CompletionMode)
	require.Nil(t, job.Spec.Completions)
	require.Nil(t, job.Spec.Parallelism)
	require.Nil(t, job.Spec.BackoffLimitPerIndex)
	require.Nil(t, job.Spec.MaxFailedIndexes)
	require.NotNil(t, job.Spec.BackoffLimit)
	require.Equal(t, int32(3), *job.Spec.BackoffLimit)
	require.Equal(t, corev1.RestartPolicyOnFailure, job.Spec.Template.Spec.RestartPolicy)
}

func TestMakeCompactJobShardedMode(t *testing.T) {
	c := newTestController(t)
	cb := newCompactForTest()
	cb.Spec.Mode = v1alpha1.CompactModeSharded
	cb.Spec.ShardCount = ptr.To[int32](4)

	job, _, err := c.makeCompactJob(cb)
	require.NoError(t, err)
	require.NotNil(t, job.Spec.CompletionMode)
	require.Equal(t, batchv1.IndexedCompletion, *job.Spec.CompletionMode)
	require.NotNil(t, job.Spec.Completions)
	require.Equal(t, int32(4), *job.Spec.Completions)
	require.NotNil(t, job.Spec.Parallelism)
	require.Equal(t, int32(4), *job.Spec.Parallelism)
	require.NotNil(t, job.Spec.BackoffLimitPerIndex)
	require.Equal(t, int32(3), *job.Spec.BackoffLimitPerIndex)
	require.NotNil(t, job.Spec.MaxFailedIndexes)
	require.Equal(t, int32(4), *job.Spec.MaxFailedIndexes)
	require.Nil(t, job.Spec.BackoffLimit)
	require.Equal(t, corev1.RestartPolicyNever, job.Spec.Template.Spec.RestartPolicy)
}
```

- [ ] **Step 3: Run the focused controller tests and confirm they fail**

Run:

```bash
go test ./pkg/controller/compactbackup -run 'Test(RequireShardedJobK8sVersion|MakeCompactJob(DefaultModeUnchanged|ShardedMode))' -v
```

Expected: fail because the helper and sharded Job branch do not exist yet.

- [ ] **Step 4: Implement the version gate helper**

Create `pkg/controller/compactbackup/server_version.go`:

```go
package compact

import (
	"fmt"
	"strconv"
	"strings"

	"k8s.io/client-go/discovery"
)

func requireShardedJobK8sVersion(d discovery.DiscoveryInterface) error {
	info, err := d.ServerVersion()
	if err != nil {
		return fmt.Errorf("discover Kubernetes server version: %w", err)
	}

	major, err := strconv.Atoi(strings.TrimSuffix(info.Major, "+"))
	if err != nil {
		return fmt.Errorf("parse Kubernetes major version %q: %w", info.Major, err)
	}
	minor, err := strconv.Atoi(strings.TrimSuffix(info.Minor, "+"))
	if err != nil {
		return fmt.Errorf("parse Kubernetes minor version %q: %w", info.Minor, err)
	}

	if major > 1 || (major == 1 && minor >= 29) {
		return nil
	}
	return fmt.Errorf("CompactBackup sharded mode requires Kubernetes >= 1.29, got %s.%s", info.Major, info.Minor)
}
```

- [ ] **Step 5: Extend controller validation and `sync()` gate**

In `pkg/controller/compactbackup/compact_backup_controller.go`, add sharded validation to `validate()`:

```go
	if spec.Mode == v1alpha1.CompactModeSharded {
		if spec.ShardCount == nil || *spec.ShardCount < 1 {
			return errors.NewNoStackError("shardCount must be set to a value >= 1 when mode is sharded")
		}
	}
```

Then in `sync()`, after `validate()` succeeds and before `checkJobStatus()`:

```go
	if compact.Spec.Mode == v1alpha1.CompactModeSharded {
		if err := requireShardedJobK8sVersion(c.deps.KubeClientset.Discovery()); err != nil {
			return c.UpdateStatus(compact, string(v1alpha1.BackupFailed), err.Error())
		}
	}
```

- [ ] **Step 6: Add the sharded Job branch in `makeCompactJob()`**

Refactor the final JobSpec construction to branch on `compact.Spec.Mode`:

```go
	jobSpec := batchv1.JobSpec{
		Template: *podSpec,
	}

	if compact.Spec.Mode == v1alpha1.CompactModeSharded {
		shardCount := *compact.Spec.ShardCount
		completionMode := batchv1.IndexedCompletion
		retryPerIndex := compact.Spec.MaxRetryTimes

		jobSpec.CompletionMode = &completionMode
		jobSpec.Completions = &shardCount
		jobSpec.Parallelism = &shardCount
		jobSpec.BackoffLimitPerIndex = &retryPerIndex
		jobSpec.MaxFailedIndexes = &shardCount
		jobSpec.Template.Spec.RestartPolicy = corev1.RestartPolicyNever
	} else {
		jobSpec.BackoffLimit = ptr.To(compact.Spec.MaxRetryTimes)
	}
```

Everything else in `makeCompactJob()` stays shared so the default path does not drift.

- [ ] **Step 7: Re-run the focused controller tests**

Run:

```bash
go test ./pkg/controller/compactbackup -run 'Test(RequireShardedJobK8sVersion|MakeCompactJob(DefaultModeUnchanged|ShardedMode))' -v
```

Expected: pass.

- [ ] **Step 8: Commit the controller/job construction slice**

```bash
git add \
  pkg/controller/compactbackup/server_version.go \
  pkg/controller/compactbackup/server_version_test.go \
  pkg/controller/compactbackup/testhelpers_test.go \
  pkg/controller/compactbackup/compact_backup_controller.go \
  pkg/controller/compactbackup/compact_backup_controller_sharded_test.go
git commit -m "feat(compactbackup): add sharded Indexed Job controller path"
```

## Task 3: Propagate `completedIndexes` and `failedIndexes` into CompactBackup Status

**Files:**
- Modify: `pkg/controller/compactbackup/compact_backup_controller.go`
- Modify: `pkg/controller/compact_status_updater.go`
- Test: `pkg/controller/compactbackup/compact_backup_controller_sharded_test.go`

- [ ] **Step 1: Write a failing status-propagation test**

Append to `pkg/controller/compactbackup/compact_backup_controller_sharded_test.go`:

```go
func TestCheckJobStatusMirrorsShardIndexes(t *testing.T) {
	c := newTestController(t)
	cb := newCompactForTest()
	cb.Spec.Mode = v1alpha1.CompactModeSharded
	cb.Spec.ShardCount = ptr.To[int32](3)

	_, err := c.deps.Clientset.PingcapV1alpha1().CompactBackups(cb.Namespace).Create(context.TODO(), cb.DeepCopy(), metav1.CreateOptions{})
	require.NoError(t, err)
	err = c.deps.InformerFactory.Pingcap().V1alpha1().CompactBackups().Informer().GetIndexer().Add(cb.DeepCopy())
	require.NoError(t, err)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cb.Namespace,
			Name:      cb.Name,
		},
		Status: batchv1.JobStatus{
			CompletedIndexes: "0-1",
			FailedIndexes:    ptr.To("2"),
		},
	}
	_, err = c.deps.KubeClientset.BatchV1().Jobs(cb.Namespace).Create(context.TODO(), job, metav1.CreateOptions{})
	require.NoError(t, err)

	allowed, err := c.checkJobStatus(cb)
	require.NoError(t, err)
	require.False(t, allowed)

	updated, err := c.deps.Clientset.PingcapV1alpha1().CompactBackups(cb.Namespace).Get(context.TODO(), cb.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "0-1", updated.Status.CompletedIndexes)
	require.Equal(t, "2", updated.Status.FailedIndexes)
}
```

- [ ] **Step 2: Run the focused test and confirm it fails**

Run:

```bash
go test ./pkg/controller/compactbackup -run TestCheckJobStatusMirrorsShardIndexes -v
```

Expected: fail because the fields are not copied into CompactBackup status.

- [ ] **Step 3: Expose direct status updates through the compact status updater**

In `pkg/controller/compact_status_updater.go`, update the interface:

```go
type CompactStatusUpdaterInterface interface {
	UpdateStatus(compact *v1alpha1.CompactBackup, newStatus v1alpha1.CompactStatus) error
	OnSchedule(ctx context.Context, compact *v1alpha1.CompactBackup, err error) error
	OnCreateJob(ctx context.Context, compact *v1alpha1.CompactBackup, err error) error
	OnStart(ctx context.Context, compact *v1alpha1.CompactBackup) error
	OnProgress(ctx context.Context, compact *v1alpha1.CompactBackup, p *Progress, endTs string) error
	OnFinish(ctx context.Context, compact *v1alpha1.CompactBackup, err error) error
	OnJobFailed(ctx context.Context, compact *v1alpha1.CompactBackup, reason string) error
}
```

Then in `UpdateStatus()`, add:

```go
	if newStatus.CompletedIndexes != compact.Status.CompletedIndexes {
		compact.Status.CompletedIndexes = newStatus.CompletedIndexes
		updated = true
	}
	if newStatus.FailedIndexes != compact.Status.FailedIndexes {
		compact.Status.FailedIndexes = newStatus.FailedIndexes
		updated = true
	}
```

- [ ] **Step 4: Mirror shard index status in `checkJobStatus()`**

At the top of `checkJobStatus()` after loading the Job, add:

```go
	if compact.Spec.Mode == v1alpha1.CompactModeSharded {
		completed := job.Status.CompletedIndexes
		failed := ""
		if job.Status.FailedIndexes != nil {
			failed = *job.Status.FailedIndexes
		}
		if completed != compact.Status.CompletedIndexes || failed != compact.Status.FailedIndexes {
			if err := c.statusUpdater.UpdateStatus(compact, v1alpha1.CompactStatus{
				CompletedIndexes: completed,
				FailedIndexes:    failed,
			}); err != nil {
				klog.Warningf("Compact: [%s/%s] failed to update shard indexes: %v", ns, name, err)
			}
		}
	}
```

This is status-only bookkeeping. The controller must still rely on Job conditions for terminal-state decisions.

- [ ] **Step 5: Re-run the focused test**

Run:

```bash
go test ./pkg/controller/compactbackup -run TestCheckJobStatusMirrorsShardIndexes -v
```

Expected: pass.

- [ ] **Step 6: Run the whole compactbackup controller package**

Run:

```bash
go test ./pkg/controller/compactbackup -v
```

Expected: all tests in the package pass.

- [ ] **Step 7: Commit the status-propagation slice**

```bash
git add \
  pkg/controller/compactbackup/compact_backup_controller.go \
  pkg/controller/compact_status_updater.go \
  pkg/controller/compactbackup/compact_backup_controller_sharded_test.go
git commit -m "feat(compactbackup): mirror Indexed Job shard status"
```

## Task 4: Add Sharded Runtime Options, Env Parsing, and `tikv-ctl` Argument Construction

**Files:**
- Modify: `cmd/backup-manager/app/compact/options/options.go`
- Modify: `cmd/backup-manager/app/compact/manager.go`
- Modify: `cmd/backup-manager/app/cmd/compact.go`
- Test: `cmd/backup-manager/app/compact/options/options_sharded_test.go`
- Test: `cmd/backup-manager/app/compact/manager_sharded_test.go`
- Test: `cmd/backup-manager/app/cmd/compact_sharded_test.go`

- [ ] **Step 1: Write failing option-parsing tests**

Create `cmd/backup-manager/app/compact/options/options_sharded_test.go`:

```go
package options

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"k8s.io/utils/ptr"
)

func TestParseCompactOptionsShardedFields(t *testing.T) {
	cb := &v1alpha1.CompactBackup{
		Spec: v1alpha1.CompactSpec{
			StartTs:     "100",
			EndTs:       "200",
			Concurrency: 4,
			Mode:        v1alpha1.CompactModeSharded,
			ShardCount:  ptr.To[int32](5),
		},
	}

	var opts CompactOpts
	err := ParseCompactOptions(cb, &opts)
	require.NoError(t, err)
	require.Equal(t, 5, opts.ShardCount)
}
```

- [ ] **Step 2: Write failing compact-args tests**

Create `cmd/backup-manager/app/compact/manager_sharded_test.go`:

```go
package compact

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/compact/options"
)

func TestBuildCompactArgsDefaultMode(t *testing.T) {
	cm := &Manager{
		options: options.CompactOpts{
			FromTS:      100,
			UntilTS:     200,
			Concurrency: 4,
		},
	}

	joined := strings.Join(cm.buildCompactArgs("BASE64"), " ")
	require.Contains(t, joined, "--from 100")
	require.Contains(t, joined, "--until 200")
	require.NotContains(t, joined, "--shard")
	require.NotContains(t, joined, "--minimal-compaction-size")
}

func TestBuildCompactArgsShardedMode(t *testing.T) {
	cm := &Manager{
		options: options.CompactOpts{
			FromTS:      100,
			UntilTS:     200,
			Concurrency: 4,
			ShardIndex:  1,
			ShardCount:  3,
		},
	}

	joined := strings.Join(cm.buildCompactArgs("BASE64"), " ")
	require.Contains(t, joined, "--from 100")
	require.Contains(t, joined, "--until 18446744073709551615")
	require.Contains(t, joined, "--shard 1/3")
	require.Contains(t, joined, "--minimal-compaction-size 0")
}
```

- [ ] **Step 3: Extend the option-parsing tests to cover shard-index env failures**

Add to `cmd/backup-manager/app/compact/options/options_sharded_test.go`:

```go
func TestParseCompactOptionsShardedFieldsRequireRuntimeIndex(t *testing.T) {
	cb := &v1alpha1.CompactBackup{
		Spec: v1alpha1.CompactSpec{
			StartTs:     "100",
			EndTs:       "200",
			Concurrency: 4,
			Mode:        v1alpha1.CompactModeSharded,
			ShardCount:  ptr.To[int32](4),
		},
	}

	var opts CompactOpts
	require.NoError(t, os.Unsetenv("JOB_COMPLETION_INDEX"))

	err := ParseCompactOptions(cb, &opts)
	require.Error(t, err)
	require.Contains(t, err.Error(), "JOB_COMPLETION_INDEX")
}

func TestParseCompactOptionsShardedFieldsRejectInvalidRuntimeIndex(t *testing.T) {
	cb := &v1alpha1.CompactBackup{
		Spec: v1alpha1.CompactSpec{
			StartTs:     "100",
			EndTs:       "200",
			Concurrency: 4,
			Mode:        v1alpha1.CompactModeSharded,
			ShardCount:  ptr.To[int32](4),
		},
	}

	var opts CompactOpts

	t.Setenv("JOB_COMPLETION_INDEX", "x")
	err := ParseCompactOptions(cb, &opts)
	require.Error(t, err)

	t.Setenv("JOB_COMPLETION_INDEX", "4")
	err = ParseCompactOptions(cb, &opts)
	require.Error(t, err)
	require.Contains(t, err.Error(), "out of range")
}
```

- [ ] **Step 4: Run the focused runtime tests and confirm they fail**

Run:

```bash
go test ./cmd/backup-manager/app/compact/options -run TestParseCompactOptionsShardedFields -v
go test ./cmd/backup-manager/app/compact -run 'TestBuildCompactArgs(DefaultMode|ShardedMode)' -v
```

Expected: fail because `CompactOpts` lacks shard fields, the build helper does not exist, and `ParseCompactOptions()` does not yet enforce shard env parsing.

- [ ] **Step 5: Extend `CompactOpts` and parse CR shard settings**

In `cmd/backup-manager/app/compact/options/options.go`, add:

```go
	ShardIndex int
	ShardCount int
	Sharded    bool
```

Then in `ParseCompactOptions()`:

```go
	opts.Sharded = compact.Spec.Mode == v1alpha1.CompactModeSharded
	opts.ShardIndex = 0
	opts.ShardCount = 0
	if compact.Spec.ShardCount != nil {
		opts.ShardCount = int(*compact.Spec.ShardCount)
	}
	if opts.Sharded {
		opts.ShardIndex, err = resolveShardIndex(opts.ShardCount)
		if err != nil {
			return err
		}
	}
```

Add runtime checks in `Verify()`:

```go
	if c.Sharded {
		if c.ShardCount <= 0 {
			return errors.New("shard-count must be greater than 0 in sharded mode")
		}
		if c.ShardIndex < 0 || c.ShardIndex >= c.ShardCount {
			return errors.Errorf("shard-index %d out of range for shard-count %d", c.ShardIndex, c.ShardCount)
		}
	}
```

- [ ] **Step 6: Extract a pure args builder and add sharded arguments**

In `cmd/backup-manager/app/compact/manager.go`, add:

```go
func (cm *Manager) buildCompactArgs(base64Storage string) []string {
	until := cm.options.UntilTS
	if cm.options.Sharded {
		until = math.MaxUint64
	}

	args := []string{
		"--log-level", "INFO",
		"--log-format", "json",
		"compact-log-backup",
		"--storage-base64", base64Storage,
		"--from", strconv.FormatUint(cm.options.FromTS, 10),
		"--until", strconv.FormatUint(until, 10),
		"-N", strconv.FormatUint(cm.options.Concurrency, 10),
	}

	if cm.options.Sharded {
		args = append(args,
			"--shard", fmt.Sprintf("%d/%d", cm.options.ShardIndex, cm.options.ShardCount),
			"--minimal-compaction-size", "0",
		)
	}

	return args
}
```

Then make `compactCmd()` call `cm.buildCompactArgs(base64Storage)`.

- [ ] **Step 7: Resolve `JOB_COMPLETION_INDEX` inside the option boundary and simplify `runCompact()`**

In `cmd/backup-manager/app/compact/options/options.go`, add:

```go
func resolveShardIndex(shardCount int) (int, error) {
	raw, ok := os.LookupEnv("JOB_COMPLETION_INDEX")
	if !ok {
		return 0, fmt.Errorf("JOB_COMPLETION_INDEX must be set in sharded mode")
	}

	idx, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("parse JOB_COMPLETION_INDEX=%q: %w", raw, err)
	}
	if idx < 0 || idx >= shardCount {
		return 0, fmt.Errorf("JOB_COMPLETION_INDEX %d out of range for shardCount %d", idx, shardCount)
	}
	return idx, nil
}
```

Then simplify `cmd/backup-manager/app/cmd/compact.go` so `runCompact()` no longer prefetches the CR just to resolve the shard index:

```go
	cm := compact.NewManager(compactInformer.Lister(), statusUpdater, compactOpts)
	return cm.ProcessCompact()
```

- [ ] **Step 8: Re-run the focused runtime tests**

Run:

```bash
go test ./cmd/backup-manager/app/compact/options -run 'TestParseCompactOptions(ShardedFields|ShardedFieldsRequireRuntimeIndex|ShardedFieldsRejectInvalidRuntimeIndex|DefaultModeClearsShardedFields)' -v
go test ./cmd/backup-manager/app/compact -run 'TestBuildCompactArgs(DefaultMode|ShardedMode)' -v
```

Expected: pass.

- [ ] **Step 9: Run the affected backup-manager packages**

Run:

```bash
go test ./cmd/backup-manager/app/compact/... -v
go test ./cmd/backup-manager/app/cmd -v
```

Expected: all affected tests pass.

- [ ] **Step 10: Commit the runtime slice**

```bash
git add \
  cmd/backup-manager/app/compact/options/options.go \
  cmd/backup-manager/app/compact/options/options_sharded_test.go \
  cmd/backup-manager/app/compact/manager.go \
  cmd/backup-manager/app/compact/manager_sharded_test.go \
  cmd/backup-manager/app/cmd/compact.go
git commit -m "feat(compact): add sharded compact runtime arguments"
```

## Task 5: Final Verification and Change-Scope Audit

**Files:**
- No new files; verification only

- [ ] **Step 1: Run the exact affected test matrix**

Run:

```bash
go test ./pkg/apis/pingcap/v1alpha1 -run 'TestCompact(Spec|Status)ShardedFieldsDeepCopy' -v
go test ./pkg/controller/compactbackup -v
go test ./cmd/backup-manager/app/compact/... -v
go test ./cmd/backup-manager/app/cmd -run TestResolveShardIndex -v
```

Expected: all pass.

- [ ] **Step 2: Audit the final diff stays inside the approved scope**

Run:

```bash
git diff --name-only
```

Expected changed files are limited to:
- `docs/plans/restore-compact-shimmering-kernighan.md`
- the API files and generated artifacts listed in Task 1
- the compactbackup controller/status updater files and tests listed in Tasks 2-3
- the backup-manager compact files and tests listed in Task 4

If any Restore files or unrelated docs/config files changed, stop and inspect before continuing.

- [ ] **Step 3: Optional smoke check for generated schema**

Run:

```bash
rg -n '"mode"|"shardCount"|"completedIndexes"|"failedIndexes"' \
  pkg/apis/pingcap/v1alpha1/openapi_generated.go \
  manifests/crd.yaml
```

Expected: all four fields appear in both generated outputs.

- [ ] **Step 4: Final commit or handoff**

If implementation was done as separate commits above, either keep them as-is or squash only if explicitly requested.

If a final aggregate commit is requested instead, use:

```bash
git add \
  pkg/apis/pingcap/v1alpha1/types.go \
  pkg/apis/pingcap/v1alpha1/zz_generated.deepcopy.go \
  pkg/apis/pingcap/v1alpha1/openapi_generated.go \
  manifests/crd.yaml \
  pkg/apis/pingcap/v1alpha1/types_compact_sharded_test.go \
  pkg/controller/compactbackup/server_version.go \
  pkg/controller/compactbackup/server_version_test.go \
  pkg/controller/compactbackup/testhelpers_test.go \
  pkg/controller/compactbackup/compact_backup_controller.go \
  pkg/controller/compactbackup/compact_backup_controller_sharded_test.go \
  pkg/controller/compact_status_updater.go \
  cmd/backup-manager/app/compact/options/options.go \
  cmd/backup-manager/app/compact/options/options_sharded_test.go \
  cmd/backup-manager/app/compact/manager.go \
  cmd/backup-manager/app/compact/manager_sharded_test.go \
  cmd/backup-manager/app/cmd/compact.go \
  cmd/backup-manager/app/cmd/compact_sharded_test.go
git commit -m "feat(compactbackup): add sharded compact mode"
```

## Self-Review

### Coverage Check

- API additions: covered in Task 1
- Default non-sharded invariants: covered in Task 2 tests and Task 4 tests
- Indexed Job construction and K8s 1.29 gate: covered in Task 2
- Shard-status propagation: covered in Task 3
- Shard env parsing and `tikv-ctl` args: covered in Task 4
- Generated files update: covered in Task 1 and Task 5
- Restore untouched: enforced by scope audit in Task 5

### Placeholder Scan

- No `TODO` or `TBD` placeholders remain.
- All commands are concrete.
- All code-bearing steps include concrete snippets.

### Type/Name Consistency Check

- API type name: `CompactMode`
- Sharded mode constant: `CompactModeSharded`
- Helper name: `requireShardedJobK8sVersion`
- Runtime flags in options: `Sharded`, `ShardCount`, `ShardIndex`
- Status fields: `CompletedIndexes`, `FailedIndexes`

## Execution Handoff

Plan updated in `docs/plans/restore-compact-shimmering-kernighan.md`.

Execution approach on approval:
- Recommended: `subagent-driven-development`
- Fallback: inline execution in this session

Do not start implementation until the user explicitly approves execution.
