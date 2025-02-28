// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package runtime

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	brv1alpha1 "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/datacontainer"
)

// Job is the interface for a job. for example, backup, restore, etc.
type Job interface {
	Object
	// Failed returns true if the job is Failed
	Failed() bool
	// Completed returns true if the job is Completed
	Completed() bool
	// Invalid returns true if the job is invalid
	Invalid() bool
	// NeedAddFinalizer returns true if the job needs a finalizer
	NeedAddFinalizer() bool
	// NeedRemoveFinalizer returns true if the job needs to remove the finalizer
	NeedRemoveFinalizer() bool

	// K8sJob returns the k8s job object key
	K8sJob() client.ObjectKey

	Object() client.Object
}

type RetryResult string

const (
	RetryResultTriggered   RetryResult = "Triggered"
	RetryResultExceedLimit RetryResult = "ExceedLimit"
)

type PostRetryData struct {
	Result      RetryResult
	Reason      string
	Message     string
	TriggeredAt time.Time // set if the result is Triggered

	IsExceedRetryTimes bool
	IsRetryTimeout     bool
}
type RetriableJob interface {
	Job

	// NeedRetry returns true if the job needs to be retried
	NeedRetry() bool
	// LastRetryRecord returns the last retry record of the job
	LastRetryRecord() (brv1alpha1.BackoffRetryRecord, bool)
	// Retry is called when the job is retried
	Retry(reason, originalReason string, now time.Time) (nextRunAfter *time.Duration, _ error)
	// PostRetry is called when the job is retried
	PostRetry(now time.Time) error
}

// br
type (
	Backup brv1alpha1.Backup
)

func FromBackup(b *brv1alpha1.Backup) *Backup {
	return (*Backup)(b)
}

func ToBackup(b *Backup) *brv1alpha1.Backup {
	return (*brv1alpha1.Backup)(b)
}

var _ Job = &Backup{}

func (b *Backup) NeedAddFinalizer() bool {
	bk := ToBackup(b)
	return bk.DeletionTimestamp == nil && brv1alpha1.IsCleanCandidate(bk)
}

func (b *Backup) NeedRemoveFinalizer() bool {
	bk := ToBackup(b)
	return brv1alpha1.IsCleanCandidate(bk) && isBackupDeletionCandidate(bk) && brv1alpha1.IsBackupClean(bk)
}

func isBackupDeletionCandidate(backup *v1alpha1.Backup) bool {
	return backup.DeletionTimestamp != nil && datacontainer.ContainsString(backup.Finalizers, metav1alpha1.Finalizer, nil)
}

// TODO(ideascf): move this to pkg/utils/k8s

func (b *Backup) Completed() bool {
	return brv1alpha1.IsBackupComplete(ToBackup(b))
}

func (b *Backup) Failed() bool {
	return brv1alpha1.IsBackupFailed(ToBackup(b))
}

func (b *Backup) Invalid() bool {
	return brv1alpha1.IsBackupInvalid(ToBackup(b))
}

func (b *Backup) K8sJob() client.ObjectKey {
	return client.ObjectKey{
		Namespace: b.Namespace,
		Name:      ToBackup(b).GetBackupJobName(),
	}
}

func (b *Backup) NeedRetry() bool {
	// only snapshot backup can be retried
	return b.Spec.Mode == brv1alpha1.BackupModeSnapshot
}

func (b *Backup) LastRetryRecord() (brv1alpha1.BackoffRetryRecord, bool) {
	return ToBackup(b).LastRetryRecord()
}

func (b *Backup) Retry(reason, originalReason string, now time.Time) (nextRunAt *time.Duration, _ error) {
	return ToBackup(b).Retry(reason, originalReason, now)
}

func (b *Backup) PostRetry(now time.Time) error {
	return ToBackup(b).PostRetry(now)
}

func (b *Backup) SetCluster(cluster string) {
	b.Spec.BR.Cluster = cluster
}

func (b *Backup) Cluster() string {
	return b.Spec.BR.Cluster
}

func (b *Backup) Component() string {
	return brv1alpha1.LabelValComponentBackup
}

func (b *Backup) Conditions() []metav1.Condition {
	return b.Status.Conditions
}

func (b *Backup) SetConditions(conds []metav1.Condition) {
	b.Status.Conditions = conds
}

func (b *Backup) ObservedGeneration() int64 {
	// dummy function
	return 0
}

func (b *Backup) SetObservedGeneration(g int64) {
	// dummy function
}

func (b *Backup) Object() client.Object {
	return (*brv1alpha1.Backup)(b)
}

type (
	Restore brv1alpha1.Restore
)

var _ Job = &Restore{}

func (r *Restore) NeedAddFinalizer() bool {
	return !(r.Completed() || r.Failed())
}

func (r *Restore) NeedRemoveFinalizer() bool {
	return r.Completed() || r.Failed()
}

func (r *Restore) Completed() bool {
	return brv1alpha1.IsRestoreComplete(ToRestore(r))
}

func (r *Restore) Failed() bool {
	return brv1alpha1.IsRestoreFailed(ToRestore(r))
}

func (r *Restore) Invalid() bool {
	return brv1alpha1.IsRestoreInvalid(ToRestore(r))
}

func (r *Restore) K8sJob() client.ObjectKey {
	return client.ObjectKey{
		Namespace: r.Namespace,
		Name:      ToRestore(r).GetRestoreJobName(),
	}
}

func (r *Restore) SetCluster(cluster string) {
	r.Spec.BR.Cluster = cluster
}

func (r *Restore) Cluster() string {
	return r.Spec.BR.Cluster
}

func (r *Restore) Component() string {
	return brv1alpha1.LabelValComponentRestore
}

func (r *Restore) Conditions() []metav1.Condition {
	return r.Status.Conditions
}

func (r *Restore) SetConditions(conds []metav1.Condition) {
	r.Status.Conditions = conds
}

func (r *Restore) ObservedGeneration() int64 {
	// dummy function
	return 0
}

func (r *Restore) SetObservedGeneration(g int64) {
	// dummy function
}

func (r *Restore) Object() client.Object {
	return (*brv1alpha1.Restore)(r)
}

func FromRestore(r *brv1alpha1.Restore) *Restore {
	return (*Restore)(r)
}

func ToRestore(r *Restore) *brv1alpha1.Restore {
	return (*brv1alpha1.Restore)(r)
}
