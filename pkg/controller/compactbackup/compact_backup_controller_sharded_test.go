package compact

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

func TestMakeCompactJobDefaultModeUnchanged(t *testing.T) {
	c := newTestController(t)
	compact := newCompactBackupForTest()

	job, reason, err := c.makeCompactJob(compact)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if reason != "" {
		t.Fatalf("expected empty reason, got %q", reason)
	}
	if job.Spec.CompletionMode != nil {
		t.Fatalf("expected CompletionMode to be nil, got %v", *job.Spec.CompletionMode)
	}
	if job.Spec.Completions != nil {
		t.Fatalf("expected Completions to be nil, got %d", *job.Spec.Completions)
	}
	if job.Spec.Parallelism != nil {
		t.Fatalf("expected Parallelism to be nil, got %d", *job.Spec.Parallelism)
	}
	if job.Spec.BackoffLimitPerIndex != nil {
		t.Fatalf("expected BackoffLimitPerIndex to be nil, got %d", *job.Spec.BackoffLimitPerIndex)
	}
	if job.Spec.MaxFailedIndexes != nil {
		t.Fatalf("expected MaxFailedIndexes to be nil, got %d", *job.Spec.MaxFailedIndexes)
	}
	if job.Spec.BackoffLimit == nil {
		t.Fatal("expected BackoffLimit to be set")
	}
	if *job.Spec.BackoffLimit != compact.Spec.MaxRetryTimes {
		t.Fatalf("expected BackoffLimit %d, got %d", compact.Spec.MaxRetryTimes, *job.Spec.BackoffLimit)
	}
	if job.Spec.Template.Spec.RestartPolicy != corev1.RestartPolicyOnFailure {
		t.Fatalf("expected RestartPolicy %q, got %q", corev1.RestartPolicyOnFailure, job.Spec.Template.Spec.RestartPolicy)
	}
}

func TestMakeCompactJobShardedMode(t *testing.T) {
	c := newTestController(t)
	compact := newCompactBackupForTest()
	shardCount := int32(3)
	compact.Spec.Mode = v1alpha1.CompactModeSharded
	compact.Spec.ShardCount = &shardCount

	job, reason, err := c.makeCompactJob(compact)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if reason != "" {
		t.Fatalf("expected empty reason, got %q", reason)
	}
	if job.Spec.CompletionMode == nil || *job.Spec.CompletionMode != batchv1.IndexedCompletion {
		t.Fatalf("expected Indexed completion mode, got %v", job.Spec.CompletionMode)
	}
	if job.Spec.Completions == nil || *job.Spec.Completions != shardCount {
		t.Fatalf("expected Completions %d, got %v", shardCount, job.Spec.Completions)
	}
	if job.Spec.Parallelism == nil || *job.Spec.Parallelism != shardCount {
		t.Fatalf("expected Parallelism %d, got %v", shardCount, job.Spec.Parallelism)
	}
	if job.Spec.BackoffLimitPerIndex == nil || *job.Spec.BackoffLimitPerIndex != compact.Spec.MaxRetryTimes {
		t.Fatalf("expected BackoffLimitPerIndex %d, got %v", compact.Spec.MaxRetryTimes, job.Spec.BackoffLimitPerIndex)
	}
	if job.Spec.MaxFailedIndexes == nil || *job.Spec.MaxFailedIndexes != shardCount {
		t.Fatalf("expected MaxFailedIndexes %d, got %v", shardCount, job.Spec.MaxFailedIndexes)
	}
	if job.Spec.BackoffLimit != nil {
		t.Fatalf("expected BackoffLimit to be nil, got %d", *job.Spec.BackoffLimit)
	}
	if job.Spec.Template.Spec.RestartPolicy != corev1.RestartPolicyNever {
		t.Fatalf("expected RestartPolicy %q, got %q", corev1.RestartPolicyNever, job.Spec.Template.Spec.RestartPolicy)
	}
}

func TestValidateShardedModeRequiresPositiveShardCount(t *testing.T) {
	c := newTestController(t)
	compact := newCompactBackupForTest()
	compact.Spec.Mode = v1alpha1.CompactModeSharded

	err := c.validate(compact)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !strings.Contains(err.Error(), "shardCount") {
		t.Fatalf("expected shardCount validation error, got %v", err)
	}
}

func TestSyncShardedModeRequiresSupportedK8sVersion(t *testing.T) {
	c := newTestController(t)
	compact := newCompactBackupForTest()
	shardCount := int32(2)
	compact.Spec.Mode = v1alpha1.CompactModeSharded
	compact.Spec.ShardCount = &shardCount

	fakeDiscovery := c.deps.KubeClientset.Discovery().(*fakediscovery.FakeDiscovery)
	fakeDiscovery.FakedServerVersion = &version.Info{Major: "1", Minor: "28"}

	if err := c.deps.InformerFactory.Pingcap().V1alpha1().CompactBackups().Informer().GetIndexer().Add(compact.DeepCopy()); err != nil {
		t.Fatalf("failed to seed compact backup indexer: %v", err)
	}
	if _, err := c.deps.Clientset.PingcapV1alpha1().CompactBackups(compact.Namespace).Create(context.TODO(), compact.DeepCopy(), metav1.CreateOptions{}); err != nil {
		t.Fatalf("failed to seed compact backup client: %v", err)
	}

	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(compact)
	if err != nil {
		t.Fatalf("failed to build key: %v", err)
	}

	if err := c.sync(key); err != nil {
		t.Fatalf("expected sync to fail closed without requeue, got %v", err)
	}

	jobControl := c.deps.JobControl.(*controller.FakeJobControl)
	if len(jobControl.JobIndexer.List()) != 0 {
		t.Fatalf("expected no Job to be created, got %d", len(jobControl.JobIndexer.List()))
	}

	updated, err := c.deps.Clientset.PingcapV1alpha1().CompactBackups(compact.Namespace).Get(context.TODO(), compact.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to fetch updated compact backup: %v", err)
	}
	if updated.Status.State != string(v1alpha1.BackupFailed) {
		t.Fatalf("expected state %q, got %q", v1alpha1.BackupFailed, updated.Status.State)
	}
	if !strings.Contains(updated.Status.Message, "Kubernetes >= 1.29") {
		t.Fatalf("expected version gate message, got %q", updated.Status.Message)
	}
}

func TestSyncShardedModeRequeuesOnDiscoveryError(t *testing.T) {
	c := newTestController(t)
	compact := newCompactBackupForTest()
	shardCount := int32(2)
	compact.Spec.Mode = v1alpha1.CompactModeSharded
	compact.Spec.ShardCount = &shardCount

	fakeDiscovery := c.deps.KubeClientset.Discovery().(*fakediscovery.FakeDiscovery)
	fakeDiscovery.PrependReactor("get", "version", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("discovery temporarily unavailable")
	})

	if err := c.deps.InformerFactory.Pingcap().V1alpha1().CompactBackups().Informer().GetIndexer().Add(compact); err != nil {
		t.Fatalf("failed to seed compact backup indexer: %v", err)
	}
	if _, err := c.deps.Clientset.PingcapV1alpha1().CompactBackups(compact.Namespace).Create(context.TODO(), compact, metav1.CreateOptions{}); err != nil {
		t.Fatalf("failed to seed compact backup client: %v", err)
	}

	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(compact)
	if err != nil {
		t.Fatalf("failed to build key: %v", err)
	}

	err = c.sync(key)
	if err == nil || !controller.IsRequeueError(err) {
		t.Fatalf("expected requeue error, got %v", err)
	}

	jobControl := c.deps.JobControl.(*controller.FakeJobControl)
	if len(jobControl.JobIndexer.List()) != 0 {
		t.Fatalf("expected no Job to be created, got %d", len(jobControl.JobIndexer.List()))
	}

	updated, err := c.deps.Clientset.PingcapV1alpha1().CompactBackups(compact.Namespace).Get(context.TODO(), compact.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to fetch updated compact backup: %v", err)
	}
	if updated.Status.State == string(v1alpha1.BackupFailed) {
		t.Fatalf("expected transient discovery error to avoid terminal failure, got state %q", updated.Status.State)
	}
}

func TestCheckJobStatusMirrorsShardIndexes(t *testing.T) {
	c := newTestController(t)
	compact := newCompactBackupForTest()
	shardCount := int32(3)
	compact.Spec.Mode = v1alpha1.CompactModeSharded
	compact.Spec.ShardCount = &shardCount

	if err := c.deps.InformerFactory.Pingcap().V1alpha1().CompactBackups().Informer().GetIndexer().Add(compact); err != nil {
		t.Fatalf("failed to seed compact backup indexer: %v", err)
	}
	if _, err := c.deps.Clientset.PingcapV1alpha1().CompactBackups(compact.Namespace).Create(context.TODO(), compact, metav1.CreateOptions{}); err != nil {
		t.Fatalf("failed to seed compact backup client: %v", err)
	}

	failedIndexes := "1"
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      compact.Name,
			Namespace: compact.Namespace,
		},
		Status: batchv1.JobStatus{
			CompletedIndexes: "0,2",
			FailedIndexes:    &failedIndexes,
		},
	}
	if _, err := c.deps.KubeClientset.BatchV1().Jobs(compact.Namespace).Create(context.TODO(), job, metav1.CreateOptions{}); err != nil {
		t.Fatalf("failed to seed job client: %v", err)
	}

	ok, err := c.checkJobStatus(compact.DeepCopy())
	if err != nil {
		t.Fatalf("expected no error from checkJobStatus, got %v", err)
	}
	if ok {
		t.Fatal("expected checkJobStatus to block job creation while the sharded job is running")
	}

	updated, err := c.deps.Clientset.PingcapV1alpha1().CompactBackups(compact.Namespace).Get(context.TODO(), compact.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to fetch updated compact backup: %v", err)
	}
	if updated.Status.CompletedIndexes != job.Status.CompletedIndexes {
		t.Fatalf("expected CompletedIndexes %q, got %q", job.Status.CompletedIndexes, updated.Status.CompletedIndexes)
	}
	if updated.Status.FailedIndexes != *job.Status.FailedIndexes {
		t.Fatalf("expected FailedIndexes %q, got %q", *job.Status.FailedIndexes, updated.Status.FailedIndexes)
	}
}

func TestCheckJobStatusMarksShardedCompactComplete(t *testing.T) {
	c := newTestController(t)
	compact := newCompactBackupForTest()
	shardCount := int32(3)
	compact.Spec.Mode = v1alpha1.CompactModeSharded
	compact.Spec.ShardCount = &shardCount
	compact.Status.State = string(v1alpha1.BackupRunning)

	if err := c.deps.InformerFactory.Pingcap().V1alpha1().CompactBackups().Informer().GetIndexer().Add(compact.DeepCopy()); err != nil {
		t.Fatalf("failed to seed compact backup indexer: %v", err)
	}
	if _, err := c.deps.Clientset.PingcapV1alpha1().CompactBackups(compact.Namespace).Create(context.TODO(), compact.DeepCopy(), metav1.CreateOptions{}); err != nil {
		t.Fatalf("failed to seed compact backup client: %v", err)
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      compact.Name,
			Namespace: compact.Namespace,
		},
		Status: batchv1.JobStatus{
			CompletedIndexes: "0-2",
			Conditions: []batchv1.JobCondition{
				{
					Type:   batchv1.JobComplete,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	if _, err := c.deps.KubeClientset.BatchV1().Jobs(compact.Namespace).Create(context.TODO(), job, metav1.CreateOptions{}); err != nil {
		t.Fatalf("failed to seed job client: %v", err)
	}

	ok, err := c.checkJobStatus(compact.DeepCopy())
	if err != nil {
		t.Fatalf("expected no error from checkJobStatus, got %v", err)
	}
	if ok {
		t.Fatal("expected checkJobStatus to block new job creation once the sharded job completed")
	}

	updated, err := c.deps.Clientset.PingcapV1alpha1().CompactBackups(compact.Namespace).Get(context.TODO(), compact.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to fetch updated compact backup: %v", err)
	}
	if updated.Status.State != string(v1alpha1.BackupComplete) {
		t.Fatalf("expected state %q, got %q", v1alpha1.BackupComplete, updated.Status.State)
	}
	if updated.Status.CompletedIndexes != job.Status.CompletedIndexes {
		t.Fatalf("expected CompletedIndexes %q, got %q", job.Status.CompletedIndexes, updated.Status.CompletedIndexes)
	}
}

func TestCheckJobStatusMarksShardedCompactFailedWithoutDroppingIndexes(t *testing.T) {
	c := newTestController(t)
	compact := newCompactBackupForTest()
	shardCount := int32(3)
	compact.Spec.Mode = v1alpha1.CompactModeSharded
	compact.Spec.ShardCount = &shardCount
	compact.Status.State = string(v1alpha1.BackupRunning)

	if err := c.deps.InformerFactory.Pingcap().V1alpha1().CompactBackups().Informer().GetIndexer().Add(compact); err != nil {
		t.Fatalf("failed to seed compact backup indexer: %v", err)
	}
	if _, err := c.deps.Clientset.PingcapV1alpha1().CompactBackups(compact.Namespace).Create(context.TODO(), compact, metav1.CreateOptions{}); err != nil {
		t.Fatalf("failed to seed compact backup client: %v", err)
	}

	failedIndexes := "1"
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      compact.Name,
			Namespace: compact.Namespace,
		},
		Status: batchv1.JobStatus{
			CompletedIndexes: "0,2",
			FailedIndexes:    &failedIndexes,
			Conditions: []batchv1.JobCondition{
				{
					Type:    batchv1.JobFailed,
					Status:  corev1.ConditionTrue,
					Message: "indexed job failed",
				},
			},
		},
	}
	if _, err := c.deps.KubeClientset.BatchV1().Jobs(compact.Namespace).Create(context.TODO(), job, metav1.CreateOptions{}); err != nil {
		t.Fatalf("failed to seed job client: %v", err)
	}

	ok, err := c.checkJobStatus(compact)
	if err != nil {
		t.Fatalf("expected no error from checkJobStatus, got %v", err)
	}
	if ok {
		t.Fatal("expected checkJobStatus to block new job creation once the sharded job failed")
	}

	updated, err := c.deps.Clientset.PingcapV1alpha1().CompactBackups(compact.Namespace).Get(context.TODO(), compact.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to fetch updated compact backup: %v", err)
	}
	if updated.Status.State != string(v1alpha1.BackupFailed) {
		t.Fatalf("expected state %q, got %q", v1alpha1.BackupFailed, updated.Status.State)
	}
	if updated.Status.CompletedIndexes != job.Status.CompletedIndexes {
		t.Fatalf("expected CompletedIndexes %q, got %q", job.Status.CompletedIndexes, updated.Status.CompletedIndexes)
	}
	if updated.Status.FailedIndexes != *job.Status.FailedIndexes {
		t.Fatalf("expected FailedIndexes %q, got %q", *job.Status.FailedIndexes, updated.Status.FailedIndexes)
	}
}

func TestHandleJobEventEnqueuesOwningCompactBackup(t *testing.T) {
	c := newTestController(t)
	compact := newCompactBackupForTest()
	shardCount := int32(3)
	compact.Spec.Mode = v1alpha1.CompactModeSharded
	compact.Spec.ShardCount = &shardCount

	if err := c.deps.InformerFactory.Pingcap().V1alpha1().CompactBackups().Informer().GetIndexer().Add(compact); err != nil {
		t.Fatalf("failed to seed compact backup indexer: %v", err)
	}
	if _, err := c.deps.Clientset.PingcapV1alpha1().CompactBackups(compact.Namespace).Create(context.TODO(), compact, metav1.CreateOptions{}); err != nil {
		t.Fatalf("failed to seed compact backup client: %v", err)
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      compact.Name,
			Namespace: compact.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				controller.GetCompactBackupOwnerRef(compact),
			},
		},
	}

	c.handleJobEvent(job)

	if got := c.queue.Len(); got != 1 {
		t.Fatalf("expected compact queue length 1 after job event, got %d", got)
	}
}

func TestHandleJobEventEnqueuesOwningCompactBackupFromTombstone(t *testing.T) {
	c := newTestController(t)
	compact := newCompactBackupForTest()
	shardCount := int32(3)
	compact.Spec.Mode = v1alpha1.CompactModeSharded
	compact.Spec.ShardCount = &shardCount

	if err := c.deps.InformerFactory.Pingcap().V1alpha1().CompactBackups().Informer().GetIndexer().Add(compact); err != nil {
		t.Fatalf("failed to seed compact backup indexer: %v", err)
	}
	if _, err := c.deps.Clientset.PingcapV1alpha1().CompactBackups(compact.Namespace).Create(context.TODO(), compact, metav1.CreateOptions{}); err != nil {
		t.Fatalf("failed to seed compact backup client: %v", err)
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      compact.Name,
			Namespace: compact.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				controller.GetCompactBackupOwnerRef(compact),
			},
		},
	}

	c.handleJobEvent(cache.DeletedFinalStateUnknown{Obj: job})

	if got := c.queue.Len(); got != 1 {
		t.Fatalf("expected compact queue length 1 after tombstone event, got %d", got)
	}
}

func TestUpdateShardIndexesClearsStaleValues(t *testing.T) {
	c := newTestController(t)
	compact := newCompactBackupForTest()
	shardCount := int32(3)
	compact.Spec.Mode = v1alpha1.CompactModeSharded
	compact.Spec.ShardCount = &shardCount
	compact.Status.FailedIndexes = "stale"
	compact.Status.CompletedIndexes = "old-complete"

	if err := c.deps.InformerFactory.Pingcap().V1alpha1().CompactBackups().Informer().GetIndexer().Add(compact); err != nil {
		t.Fatalf("failed to seed compact backup indexer: %v", err)
	}
	if _, err := c.deps.Clientset.PingcapV1alpha1().CompactBackups(compact.Namespace).Create(context.TODO(), compact, metav1.CreateOptions{}); err != nil {
		t.Fatalf("failed to seed compact backup client: %v", err)
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      compact.Name,
			Namespace: compact.Namespace,
		},
		Status: batchv1.JobStatus{},
	}

	if err := c.statusUpdater.UpdateShardIndexes(compact, job.Status); err != nil {
		t.Fatalf("expected no error from UpdateShardIndexes, got %v", err)
	}

	updated, err := c.deps.Clientset.PingcapV1alpha1().CompactBackups(compact.Namespace).Get(context.TODO(), compact.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to fetch updated compact backup: %v", err)
	}
	if updated.Status.CompletedIndexes != "" {
		t.Fatalf("expected CompletedIndexes to clear, got %q", updated.Status.CompletedIndexes)
	}
	if updated.Status.FailedIndexes != "" {
		t.Fatalf("expected FailedIndexes to clear, got %q", updated.Status.FailedIndexes)
	}
}

func newCompactBackupForTest() *v1alpha1.CompactBackup {
	return &v1alpha1.CompactBackup{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CompactBackup",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-compact",
			Namespace: "demo-ns",
			UID:       types.UID("demo-compact-uid"),
		},
		Spec: v1alpha1.CompactSpec{
			StartTs:       "100",
			EndTs:         "200",
			Concurrency:   4,
			MaxRetryTimes: 6,
			StorageProvider: v1alpha1.StorageProvider{
				Local: &v1alpha1.LocalStorageProvider{},
			},
			BR: &v1alpha1.BRConfig{
				Cluster:          "demo-tc",
				ClusterNamespace: "demo-ns",
			},
		},
	}
}
