// Copyright 2023 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package backup

import (
	"context"
	"testing"

	"github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1"
	pingcapv1alpha1 "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

const (
	fakeTcName1      = "db-1"
	fakeTcNamespace1 = "ns-1"
	fakeTcName2      = "db-2"
	fakeTcNamespace2 = "ns-2"
	fakeTcName3      = "db-3"
	fakeTcNamespace3 = "ns-3"
)

type helper struct {
	g    gomega.Gomega
	deps *controller.BrFedDependencies
	bm   *backupManager

	backupName      string
	backupNamespace string

	dataPlaneClient1 versioned.Interface
	dataPlaneClient2 versioned.Interface
	dataPlaneClient3 versioned.Interface

	backupMemberName1 string
	backupMemberName2 string
	backupMemberName3 string
}

func newHelper(t *testing.T, backupName, backupNamespace string) *helper {
	h := &helper{}
	h.g = gomega.NewWithT(t)
	h.deps = controller.NewFakeBrFedDependencies()
	h.bm = NewBackupManager(h.deps).(*backupManager)
	h.backupName = backupName
	h.backupNamespace = backupNamespace

	h.dataPlaneClient1 = h.deps.FedClientset[controller.FakeDataPlaneName1]
	h.dataPlaneClient2 = h.deps.FedClientset[controller.FakeDataPlaneName2]
	h.dataPlaneClient3 = h.deps.FedClientset[controller.FakeDataPlaneName3]

	h.backupMemberName1 = h.bm.generateBackupMemberName(backupName, controller.FakeDataPlaneName1)
	h.backupMemberName2 = h.bm.generateBackupMemberName(backupName, controller.FakeDataPlaneName2)
	h.backupMemberName3 = h.bm.generateBackupMemberName(backupName, controller.FakeDataPlaneName3)
	return h
}

func (h *helper) createVolumeBackup(ctx context.Context) *v1alpha1.VolumeBackup {
	volumeBackup := generateVolumeBackup(h.backupName, h.backupNamespace)
	volumeBackup, err := h.deps.Clientset.FederationV1alpha1().VolumeBackups(h.backupNamespace).Create(ctx, volumeBackup, metav1.CreateOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	h.g.Expect(volumeBackup.Status.TimeStarted.Unix() < 0).To(gomega.BeTrue())
	h.g.Expect(volumeBackup.Status.TimeCompleted.Unix() < 0).To(gomega.BeTrue())
	h.g.Expect(len(volumeBackup.Status.TimeTaken)).To(gomega.Equal(0))
	return volumeBackup
}

func (h *helper) assertRunInitialize(ctx context.Context, volumeBackup *v1alpha1.VolumeBackup) {
	h.g.Expect(volumeBackup.Status.Phase).To(gomega.Equal(v1alpha1.VolumeBackupRunning))
	h.g.Expect(volumeBackup.Status.TimeStarted.Unix() > 0).To(gomega.BeTrue())
	backupMember1, err := h.dataPlaneClient1.PingcapV1alpha1().Backups(fakeTcNamespace1).Get(ctx, h.backupMemberName1, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	h.g.Expect(backupMember1.Spec.FederalVolumeBackupPhase).To(gomega.Equal(pingcapv1alpha1.FederalVolumeBackupInitialize))
}

func (h *helper) assertRunExecute(ctx context.Context, volumeBackup *v1alpha1.VolumeBackup) {
	h.g.Expect(len(volumeBackup.Status.Backups)).To(gomega.Equal(1))

	backupMember1, err := h.dataPlaneClient1.PingcapV1alpha1().Backups(fakeTcNamespace1).Get(ctx, h.backupMemberName1, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	h.g.Expect(backupMember1.Spec.FederalVolumeBackupPhase).To(gomega.Equal(pingcapv1alpha1.FederalVolumeBackupExecute))
	backupMember2, err := h.dataPlaneClient2.PingcapV1alpha1().Backups(fakeTcNamespace2).Get(ctx, h.backupMemberName2, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	h.g.Expect(backupMember2.Spec.FederalVolumeBackupPhase).To(gomega.Equal(pingcapv1alpha1.FederalVolumeBackupExecute))
	backupMember3, err := h.dataPlaneClient3.PingcapV1alpha1().Backups(fakeTcNamespace3).Get(ctx, h.backupMemberName3, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	h.g.Expect(backupMember3.Spec.FederalVolumeBackupPhase).To(gomega.Equal(pingcapv1alpha1.FederalVolumeBackupExecute))
}

func (h *helper) assertRunTeardown(ctx context.Context, volumeBackup *v1alpha1.VolumeBackup, initializeFailed bool) {
	backupMember1, err := h.dataPlaneClient1.PingcapV1alpha1().Backups(fakeTcNamespace1).Get(ctx, h.backupMemberName1, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	h.g.Expect(backupMember1.Spec.FederalVolumeBackupPhase).To(gomega.Equal(pingcapv1alpha1.FederalVolumeBackupTeardown))
	if initializeFailed {
		h.g.Expect(len(volumeBackup.Status.Backups)).To(gomega.Equal(1))
		return
	}

	h.g.Expect(len(volumeBackup.Status.Backups)).To(gomega.Equal(3))
	backupMember2, err := h.dataPlaneClient2.PingcapV1alpha1().Backups(fakeTcNamespace2).Get(ctx, h.backupMemberName2, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	h.g.Expect(backupMember2.Spec.FederalVolumeBackupPhase).To(gomega.Equal(pingcapv1alpha1.FederalVolumeBackupTeardown))
	backupMember3, err := h.dataPlaneClient3.PingcapV1alpha1().Backups(fakeTcNamespace3).Get(ctx, h.backupMemberName3, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	h.g.Expect(backupMember3.Spec.FederalVolumeBackupPhase).To(gomega.Equal(pingcapv1alpha1.FederalVolumeBackupTeardown))
}

func (h *helper) assertComplete(volumeBackup *v1alpha1.VolumeBackup) {
	h.g.Expect(v1alpha1.IsVolumeBackupComplete(volumeBackup)).To(gomega.BeTrue())
	h.g.Expect(volumeBackup.Status.CommitTs).To(gomega.Equal("123"))
	h.g.Expect(volumeBackup.Status.TimeCompleted.Unix() > 0).To(gomega.BeTrue())
	h.g.Expect(len(volumeBackup.Status.TimeTaken) > 0).To(gomega.BeTrue())
}

func (h *helper) assertFailed(volumeBackup *v1alpha1.VolumeBackup) {
	h.g.Expect(v1alpha1.IsVolumeBackupFailed(volumeBackup)).To(gomega.BeTrue())
	h.g.Expect(volumeBackup.Status.TimeCompleted.Unix() > 0).To(gomega.BeTrue())
	h.g.Expect(len(volumeBackup.Status.TimeTaken) > 0).To(gomega.BeTrue())
}

func (h *helper) setDataPlaneInitialized(ctx context.Context) {
	backupMember1, err := h.dataPlaneClient1.PingcapV1alpha1().Backups(fakeTcNamespace1).Get(ctx, h.backupMemberName1, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	pingcapv1alpha1.UpdateBackupCondition(&backupMember1.Status, &pingcapv1alpha1.BackupCondition{
		Status: corev1.ConditionTrue,
		Type:   pingcapv1alpha1.VolumeBackupInitialized,
	})
	_, err = h.dataPlaneClient1.PingcapV1alpha1().Backups(fakeTcNamespace1).UpdateStatus(ctx, backupMember1, metav1.UpdateOptions{})
	h.g.Expect(err).To(gomega.BeNil())
}

func (h *helper) setDataPlaneVolumeComplete(ctx context.Context) {
	backupMember1, err := h.dataPlaneClient1.PingcapV1alpha1().Backups(fakeTcNamespace1).Get(ctx, h.backupMemberName1, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	pingcapv1alpha1.UpdateBackupCondition(&backupMember1.Status, &pingcapv1alpha1.BackupCondition{
		Status: corev1.ConditionTrue,
		Type:   pingcapv1alpha1.VolumeBackupComplete,
	})
	_, err = h.dataPlaneClient1.PingcapV1alpha1().Backups(fakeTcNamespace1).UpdateStatus(ctx, backupMember1, metav1.UpdateOptions{})
	h.g.Expect(err).To(gomega.BeNil())

	backupMember2, err := h.dataPlaneClient2.PingcapV1alpha1().Backups(fakeTcNamespace2).Get(ctx, h.backupMemberName2, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	pingcapv1alpha1.UpdateBackupCondition(&backupMember2.Status, &pingcapv1alpha1.BackupCondition{
		Status: corev1.ConditionTrue,
		Type:   pingcapv1alpha1.VolumeBackupComplete,
	})
	_, err = h.dataPlaneClient2.PingcapV1alpha1().Backups(fakeTcNamespace2).UpdateStatus(ctx, backupMember2, metav1.UpdateOptions{})
	h.g.Expect(err).To(gomega.BeNil())

	backupMember3, err := h.dataPlaneClient3.PingcapV1alpha1().Backups(fakeTcNamespace3).Get(ctx, h.backupMemberName3, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	pingcapv1alpha1.UpdateBackupCondition(&backupMember3.Status, &pingcapv1alpha1.BackupCondition{
		Status: corev1.ConditionTrue,
		Type:   pingcapv1alpha1.VolumeBackupComplete,
	})
	_, err = h.dataPlaneClient3.PingcapV1alpha1().Backups(fakeTcNamespace3).UpdateStatus(ctx, backupMember3, metav1.UpdateOptions{})
	h.g.Expect(err).To(gomega.BeNil())
}

func (h *helper) setDataPlaneComplete(ctx context.Context) {
	backupMember1, err := h.dataPlaneClient1.PingcapV1alpha1().Backups(fakeTcNamespace1).Get(ctx, h.backupMemberName1, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	pingcapv1alpha1.UpdateBackupCondition(&backupMember1.Status, &pingcapv1alpha1.BackupCondition{
		Status: corev1.ConditionTrue,
		Type:   pingcapv1alpha1.BackupComplete,
	})
	backupMember1.Status.CommitTs = "123"
	_, err = h.dataPlaneClient1.PingcapV1alpha1().Backups(fakeTcNamespace1).UpdateStatus(ctx, backupMember1, metav1.UpdateOptions{})
	h.g.Expect(err).To(gomega.BeNil())

	backupMember2, err := h.dataPlaneClient2.PingcapV1alpha1().Backups(fakeTcNamespace2).Get(ctx, h.backupMemberName2, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	pingcapv1alpha1.UpdateBackupCondition(&backupMember2.Status, &pingcapv1alpha1.BackupCondition{
		Status: corev1.ConditionTrue,
		Type:   pingcapv1alpha1.BackupComplete,
	})
	backupMember2.Status.CommitTs = "124"
	_, err = h.dataPlaneClient2.PingcapV1alpha1().Backups(fakeTcNamespace2).UpdateStatus(ctx, backupMember2, metav1.UpdateOptions{})
	h.g.Expect(err).To(gomega.BeNil())

	backupMember3, err := h.dataPlaneClient3.PingcapV1alpha1().Backups(fakeTcNamespace3).Get(ctx, h.backupMemberName3, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	pingcapv1alpha1.UpdateBackupCondition(&backupMember3.Status, &pingcapv1alpha1.BackupCondition{
		Status: corev1.ConditionTrue,
		Type:   pingcapv1alpha1.BackupComplete,
	})
	backupMember3.Status.CommitTs = "125"
	_, err = h.dataPlaneClient3.PingcapV1alpha1().Backups(fakeTcNamespace3).UpdateStatus(ctx, backupMember3, metav1.UpdateOptions{})
	h.g.Expect(err).To(gomega.BeNil())
}

func (h *helper) setDataPlaneInitializeFailed(ctx context.Context) {
	backupMember1, err := h.dataPlaneClient1.PingcapV1alpha1().Backups(fakeTcNamespace1).Get(ctx, h.backupMemberName1, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	pingcapv1alpha1.UpdateBackupCondition(&backupMember1.Status, &pingcapv1alpha1.BackupCondition{
		Status: corev1.ConditionTrue,
		Type:   pingcapv1alpha1.VolumeBackupInitializeFailed,
	})
	_, err = h.dataPlaneClient1.PingcapV1alpha1().Backups(fakeTcNamespace1).UpdateStatus(ctx, backupMember1, metav1.UpdateOptions{})
	h.g.Expect(err).To(gomega.BeNil())
}

func (h *helper) setDataPlaneVolumeFailed(ctx context.Context) {
	backupMember1, err := h.dataPlaneClient1.PingcapV1alpha1().Backups(fakeTcNamespace1).Get(ctx, h.backupMemberName1, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	pingcapv1alpha1.UpdateBackupCondition(&backupMember1.Status, &pingcapv1alpha1.BackupCondition{
		Status: corev1.ConditionTrue,
		Type:   pingcapv1alpha1.VolumeBackupFailed,
	})
	_, err = h.dataPlaneClient1.PingcapV1alpha1().Backups(fakeTcNamespace1).UpdateStatus(ctx, backupMember1, metav1.UpdateOptions{})
	h.g.Expect(err).To(gomega.BeNil())
}

func (h *helper) setDataPlaneFailed(ctx context.Context) {
	backupMember1, err := h.dataPlaneClient1.PingcapV1alpha1().Backups(fakeTcNamespace1).Get(ctx, h.backupMemberName1, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	pingcapv1alpha1.UpdateBackupCondition(&backupMember1.Status, &pingcapv1alpha1.BackupCondition{
		Status: corev1.ConditionTrue,
		Type:   pingcapv1alpha1.BackupFailed,
	})
	_, err = h.dataPlaneClient1.PingcapV1alpha1().Backups(fakeTcNamespace1).UpdateStatus(ctx, backupMember1, metav1.UpdateOptions{})
	h.g.Expect(err).To(gomega.BeNil())
}

func TestVolumeBackup(t *testing.T) {
	ctx := context.Background()
	backupName := "backup-1"
	backupNamespace := "ns-1"
	h := newHelper(t, backupName, backupNamespace)

	// create volume backup
	volumeBackup := h.createVolumeBackup(ctx)

	// run initialize phase
	err := h.bm.Sync(volumeBackup)
	h.g.Expect(err).To(gomega.BeNil())
	h.assertRunInitialize(ctx, volumeBackup)

	// wait initialized
	err = h.bm.Sync(volumeBackup)
	h.g.Expect(err).To(gomega.HaveOccurred())

	// initialized, run execute phase
	h.setDataPlaneInitialized(ctx)
	err = h.bm.Sync(volumeBackup)
	h.g.Expect(err).To(gomega.BeNil())
	h.assertRunExecute(ctx, volumeBackup)

	// wait volume complete
	err = h.bm.Sync(volumeBackup)
	h.g.Expect(err).To(gomega.HaveOccurred())

	// volume complete, run teardown phase
	h.setDataPlaneVolumeComplete(ctx)
	err = h.bm.Sync(volumeBackup)
	h.g.Expect(err).To(gomega.BeNil())
	h.assertRunTeardown(ctx, volumeBackup, false)

	// volume backup complete
	h.setDataPlaneComplete(ctx)
	err = h.bm.Sync(volumeBackup)
	h.g.Expect(err).To(gomega.BeNil())
	h.assertComplete(volumeBackup)
}

func TestVolumeBackupInitializeFailed(t *testing.T) {
	ctx := context.Background()
	backupName := "backup-2"
	backupNamespace := "ns-2"
	h := newHelper(t, backupName, backupNamespace)

	// create volume backup
	volumeBackup := h.createVolumeBackup(ctx)

	// run initialize phase
	err := h.bm.Sync(volumeBackup)
	h.g.Expect(err).To(gomega.BeNil())
	h.assertRunInitialize(ctx, volumeBackup)

	// initialize failed, run teardown
	h.setDataPlaneInitializeFailed(ctx)
	err = h.bm.Sync(volumeBackup)
	h.g.Expect(err).To(gomega.BeNil())
	h.assertRunTeardown(ctx, volumeBackup, true)

	// volume backup failed
	err = h.bm.Sync(volumeBackup)
	h.g.Expect(err).To(gomega.BeNil())
	h.assertFailed(volumeBackup)
}

func TestVolumeBackupVolumeFailed(t *testing.T) {
	ctx := context.Background()
	backupName := "backup-3"
	backupNamespace := "ns-3"
	h := newHelper(t, backupName, backupNamespace)

	// create volume backup
	volumeBackup := h.createVolumeBackup(ctx)

	// run initialize phase
	err := h.bm.Sync(volumeBackup)
	h.g.Expect(err).To(gomega.BeNil())
	h.assertRunInitialize(ctx, volumeBackup)

	// initialized, run execute phase
	h.setDataPlaneInitialized(ctx)
	err = h.bm.Sync(volumeBackup)
	h.g.Expect(err).To(gomega.BeNil())
	h.assertRunExecute(ctx, volumeBackup)

	// volume failed, run teardown
	h.setDataPlaneVolumeFailed(ctx)
	err = h.bm.Sync(volumeBackup)
	h.g.Expect(err).To(gomega.BeNil())
	h.assertRunTeardown(ctx, volumeBackup, false)

	// volume backup failed
	h.setDataPlaneFailed(ctx)
	err = h.bm.Sync(volumeBackup)
	h.g.Expect(err).To(gomega.BeNil())
	h.assertFailed(volumeBackup)
}

func generateVolumeBackup(backupName, backupNamespace string) *v1alpha1.VolumeBackup {
	return &v1alpha1.VolumeBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backupName,
			Namespace: backupNamespace,
		},
		Spec: v1alpha1.VolumeBackupSpec{
			Clusters: []v1alpha1.VolumeBackupMemberCluster{
				{
					K8sClusterName: controller.FakeDataPlaneName1,
					TCName:         fakeTcName1,
					TCNamespace:    fakeTcNamespace1,
				},
				{
					K8sClusterName: controller.FakeDataPlaneName2,
					TCName:         fakeTcName2,
					TCNamespace:    fakeTcNamespace2,
				},
				{
					K8sClusterName: controller.FakeDataPlaneName3,
					TCName:         fakeTcName3,
					TCNamespace:    fakeTcNamespace3,
				},
			},
			Template: v1alpha1.VolumeBackupMemberSpec{
				BR: &v1alpha1.BRConfig{
					SendCredToTikv: pointer.BoolPtr(false),
				},
				StorageProvider: pingcapv1alpha1.StorageProvider{
					S3: &pingcapv1alpha1.S3StorageProvider{
						Provider: pingcapv1alpha1.S3StorageProviderTypeAWS,
						Region:   "us-west-2",
						Bucket:   "bucket-1",
						Prefix:   "volume-backup-test-1",
					},
				},
				ServiceAccount: "tidb-backup-manager",
				CleanPolicy:    pingcapv1alpha1.CleanPolicyTypeDelete,
			},
		},
	}
}
