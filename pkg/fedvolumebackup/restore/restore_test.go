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

package restore

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

	fakeBackupBucket  = "bucket-1"
	fakeBackupPrefix1 = "prefix-1"
	fakeBackupPrefix2 = "prefix-2"
	fakeBackupPrefix3 = "prefix-3"
)

type helper struct {
	g    gomega.Gomega
	deps *controller.BrFedDependencies
	rm   *restoreManager

	restoreName      string
	restoreNamespace string

	dataPlaneClient1 versioned.Interface
	dataPlaneClient2 versioned.Interface
	dataPlaneClient3 versioned.Interface

	restoreMemberName1 string
	restoreMemberName2 string
	restoreMemberName3 string
}

func newHelper(t *testing.T, restoreName, restoreNamespace string) *helper {
	h := &helper{}
	h.g = gomega.NewWithT(t)
	h.deps = controller.NewFakeBrFedDependencies()
	h.rm = NewRestoreManager(h.deps).(*restoreManager)
	h.restoreName = restoreName
	h.restoreNamespace = restoreNamespace

	h.dataPlaneClient1 = h.deps.FedClientset[controller.FakeDataPlaneName1]
	h.dataPlaneClient2 = h.deps.FedClientset[controller.FakeDataPlaneName2]
	h.dataPlaneClient3 = h.deps.FedClientset[controller.FakeDataPlaneName3]

	h.restoreMemberName1 = h.rm.generateRestoreMemberName(restoreName, controller.FakeDataPlaneName1)
	h.restoreMemberName2 = h.rm.generateRestoreMemberName(restoreName, controller.FakeDataPlaneName2)
	h.restoreMemberName3 = h.rm.generateRestoreMemberName(restoreName, controller.FakeDataPlaneName3)
	return h
}

func (h *helper) createVolumeRestore(ctx context.Context) *v1alpha1.VolumeRestore {
	volumeRestore := generateVolumeRestore(h.restoreName, h.restoreNamespace)
	volumeRestore, err := h.deps.Clientset.FederationV1alpha1().VolumeRestores(h.restoreNamespace).Create(ctx, volumeRestore, metav1.CreateOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	h.g.Expect(volumeRestore.Status.TimeStarted.Unix() < 0).To(gomega.BeTrue())
	h.g.Expect(volumeRestore.Status.TimeCompleted.Unix() < 0).To(gomega.BeTrue())
	h.g.Expect(len(volumeRestore.Status.TimeTaken)).To(gomega.Equal(0))
	return volumeRestore
}

func (h *helper) assertRunRestoreVolume(ctx context.Context, volumeRestore *v1alpha1.VolumeRestore) {
	h.g.Expect(volumeRestore.Status.Phase).To(gomega.Equal(v1alpha1.VolumeRestoreRunning))
	h.g.Expect(volumeRestore.Status.TimeStarted.Unix() > 0).To(gomega.BeTrue())
	h.g.Expect(len(volumeRestore.Status.Restores)).To(gomega.Equal(3))
	h.g.Expect(len(volumeRestore.Status.Steps)).To(gomega.Equal(1))

	restoreMember1, err := h.dataPlaneClient1.PingcapV1alpha1().Restores(fakeTcNamespace1).Get(ctx, h.restoreMemberName1, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	h.g.Expect(restoreMember1.Spec.FederalVolumeRestorePhase).To(gomega.Equal(pingcapv1alpha1.FederalVolumeRestoreVolume))
	restoreMember2, err := h.dataPlaneClient2.PingcapV1alpha1().Restores(fakeTcNamespace2).Get(ctx, h.restoreMemberName2, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	h.g.Expect(restoreMember2.Spec.FederalVolumeRestorePhase).To(gomega.Equal(pingcapv1alpha1.FederalVolumeRestoreVolume))
	restoreMember3, err := h.dataPlaneClient3.PingcapV1alpha1().Restores(fakeTcNamespace3).Get(ctx, h.restoreMemberName3, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	h.g.Expect(restoreMember3.Spec.FederalVolumeRestorePhase).To(gomega.Equal(pingcapv1alpha1.FederalVolumeRestoreVolume))
}

func (h *helper) assertRestoreVolumeComplete(ctx context.Context, volumeRestore *v1alpha1.VolumeRestore) {
	h.g.Expect(volumeRestore.Status.Phase).To(gomega.Equal(v1alpha1.VolumeRestoreVolumeComplete))
	h.g.Expect(len(volumeRestore.Status.Steps)).To(gomega.Equal(2))
}

func (h *helper) assertRunRestoreData(ctx context.Context, volumeRestore *v1alpha1.VolumeRestore) {
	h.g.Expect(volumeRestore.Status.Phase).To(gomega.Equal(v1alpha1.VolumeRestoreTiKVComplete))
	h.g.Expect(len(volumeRestore.Status.Steps)).To(gomega.Equal(3))

	// in setDataPlaneVolumeComplete function, we set restore member1 with minimal commit ts
	// so restore member1 should execute restore data phase
	restoreMember1, err := h.dataPlaneClient1.PingcapV1alpha1().Restores(fakeTcNamespace1).Get(ctx, h.restoreMemberName1, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	h.g.Expect(restoreMember1.Spec.FederalVolumeRestorePhase).To(gomega.Equal(pingcapv1alpha1.FederalVolumeRestoreData))

	// other restore members should still in restore volume phase
	restoreMember2, err := h.dataPlaneClient2.PingcapV1alpha1().Restores(fakeTcNamespace2).Get(ctx, h.restoreMemberName2, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	h.g.Expect(restoreMember2.Spec.FederalVolumeRestorePhase).To(gomega.Equal(pingcapv1alpha1.FederalVolumeRestoreVolume))
	restoreMember3, err := h.dataPlaneClient3.PingcapV1alpha1().Restores(fakeTcNamespace3).Get(ctx, h.restoreMemberName3, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	h.g.Expect(restoreMember3.Spec.FederalVolumeRestorePhase).To(gomega.Equal(pingcapv1alpha1.FederalVolumeRestoreVolume))
}

func (h *helper) assertRunRestoreFinish(ctx context.Context, volumeRestore *v1alpha1.VolumeRestore) {
	h.g.Expect(volumeRestore.Status.Phase).To(gomega.Equal(v1alpha1.VolumeRestoreDataComplete))
	h.g.Expect(len(volumeRestore.Status.Steps)).To(gomega.Equal(4))
	restoreMember1, err := h.dataPlaneClient1.PingcapV1alpha1().Restores(fakeTcNamespace1).Get(ctx, h.restoreMemberName1, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	h.g.Expect(restoreMember1.Spec.FederalVolumeRestorePhase).To(gomega.Equal(pingcapv1alpha1.FederalVolumeRestoreFinish))
	restoreMember2, err := h.dataPlaneClient2.PingcapV1alpha1().Restores(fakeTcNamespace2).Get(ctx, h.restoreMemberName2, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	h.g.Expect(restoreMember2.Spec.FederalVolumeRestorePhase).To(gomega.Equal(pingcapv1alpha1.FederalVolumeRestoreFinish))
	restoreMember3, err := h.dataPlaneClient3.PingcapV1alpha1().Restores(fakeTcNamespace3).Get(ctx, h.restoreMemberName3, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	h.g.Expect(restoreMember3.Spec.FederalVolumeRestorePhase).To(gomega.Equal(pingcapv1alpha1.FederalVolumeRestoreFinish))
}

func (h *helper) assertRestoreComplete(volumeRestore *v1alpha1.VolumeRestore) {
	h.g.Expect(volumeRestore.Status.Phase).To(gomega.Equal(v1alpha1.VolumeRestoreComplete))
	h.g.Expect(volumeRestore.Status.CommitTs).To(gomega.Equal("121"))
	h.g.Expect(volumeRestore.Status.TimeCompleted.Unix() > 0).To(gomega.BeTrue())
	h.g.Expect(len(volumeRestore.Status.TimeTaken) > 0).To(gomega.BeTrue())
}

func (h *helper) assertRestoreFailed(volumeRestore *v1alpha1.VolumeRestore) {
	h.g.Expect(volumeRestore.Status.Phase).To(gomega.Equal(v1alpha1.VolumeRestoreFailed))
	h.g.Expect(volumeRestore.Status.CommitTs).To(gomega.BeEmpty())
	h.g.Expect(volumeRestore.Status.TimeCompleted.Unix() > 0).To(gomega.BeTrue())
	h.g.Expect(len(volumeRestore.Status.TimeTaken) > 0).To(gomega.BeTrue())
}

func (h *helper) setDataPlaneVolumeComplete(ctx context.Context) {
	restoreMember1, err := h.dataPlaneClient1.PingcapV1alpha1().Restores(fakeTcNamespace1).Get(ctx, h.restoreMemberName1, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	pingcapv1alpha1.UpdateRestoreCondition(&restoreMember1.Status, &pingcapv1alpha1.RestoreCondition{
		Type:   pingcapv1alpha1.RestoreVolumeComplete,
		Status: corev1.ConditionTrue,
	})
	restoreMember1.Status.CommitTs = "121"
	_, err = h.dataPlaneClient1.PingcapV1alpha1().Restores(fakeTcNamespace1).UpdateStatus(ctx, restoreMember1, metav1.UpdateOptions{})
	h.g.Expect(err).To(gomega.BeNil())

	restoreMember2, err := h.dataPlaneClient2.PingcapV1alpha1().Restores(fakeTcNamespace2).Get(ctx, h.restoreMemberName2, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	pingcapv1alpha1.UpdateRestoreCondition(&restoreMember2.Status, &pingcapv1alpha1.RestoreCondition{
		Type:   pingcapv1alpha1.RestoreVolumeComplete,
		Status: corev1.ConditionTrue,
	})
	restoreMember2.Status.CommitTs = "122"
	_, err = h.dataPlaneClient2.PingcapV1alpha1().Restores(fakeTcNamespace2).UpdateStatus(ctx, restoreMember2, metav1.UpdateOptions{})
	h.g.Expect(err).To(gomega.BeNil())

	restoreMember3, err := h.dataPlaneClient3.PingcapV1alpha1().Restores(fakeTcNamespace3).Get(ctx, h.restoreMemberName3, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	pingcapv1alpha1.UpdateRestoreCondition(&restoreMember3.Status, &pingcapv1alpha1.RestoreCondition{
		Type:   pingcapv1alpha1.RestoreVolumeComplete,
		Status: corev1.ConditionTrue,
	})
	restoreMember3.Status.CommitTs = "123"
	_, err = h.dataPlaneClient3.PingcapV1alpha1().Restores(fakeTcNamespace3).UpdateStatus(ctx, restoreMember3, metav1.UpdateOptions{})
	h.g.Expect(err).To(gomega.BeNil())
}

func (h *helper) setDataPlaneTikvComplete(ctx context.Context) {
	restoreMember1, err := h.dataPlaneClient1.PingcapV1alpha1().Restores(fakeTcNamespace1).Get(ctx, h.restoreMemberName1, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	pingcapv1alpha1.UpdateRestoreCondition(&restoreMember1.Status, &pingcapv1alpha1.RestoreCondition{
		Type:   pingcapv1alpha1.RestoreTiKVComplete,
		Status: corev1.ConditionTrue,
	})
	_, err = h.dataPlaneClient1.PingcapV1alpha1().Restores(fakeTcNamespace1).UpdateStatus(ctx, restoreMember1, metav1.UpdateOptions{})
	h.g.Expect(err).To(gomega.BeNil())

	restoreMember2, err := h.dataPlaneClient2.PingcapV1alpha1().Restores(fakeTcNamespace2).Get(ctx, h.restoreMemberName2, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	pingcapv1alpha1.UpdateRestoreCondition(&restoreMember2.Status, &pingcapv1alpha1.RestoreCondition{
		Type:   pingcapv1alpha1.RestoreTiKVComplete,
		Status: corev1.ConditionTrue,
	})
	_, err = h.dataPlaneClient2.PingcapV1alpha1().Restores(fakeTcNamespace2).UpdateStatus(ctx, restoreMember2, metav1.UpdateOptions{})
	h.g.Expect(err).To(gomega.BeNil())

	restoreMember3, err := h.dataPlaneClient3.PingcapV1alpha1().Restores(fakeTcNamespace3).Get(ctx, h.restoreMemberName3, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	pingcapv1alpha1.UpdateRestoreCondition(&restoreMember3.Status, &pingcapv1alpha1.RestoreCondition{
		Type:   pingcapv1alpha1.RestoreTiKVComplete,
		Status: corev1.ConditionTrue,
	})
	_, err = h.dataPlaneClient3.PingcapV1alpha1().Restores(fakeTcNamespace3).UpdateStatus(ctx, restoreMember3, metav1.UpdateOptions{})
	h.g.Expect(err).To(gomega.BeNil())
}

func (h *helper) setDataPlaneDataComplete(ctx context.Context) {
	restoreMember1, err := h.dataPlaneClient1.PingcapV1alpha1().Restores(fakeTcNamespace1).Get(ctx, h.restoreMemberName1, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	pingcapv1alpha1.UpdateRestoreCondition(&restoreMember1.Status, &pingcapv1alpha1.RestoreCondition{
		Type:   pingcapv1alpha1.RestoreDataComplete,
		Status: corev1.ConditionTrue,
	})
	restoreMember1.Status.CommitTs = "121"
	_, err = h.dataPlaneClient1.PingcapV1alpha1().Restores(fakeTcNamespace1).UpdateStatus(ctx, restoreMember1, metav1.UpdateOptions{})
	h.g.Expect(err).To(gomega.BeNil())
}

func (h *helper) setDataPlaneComplete(ctx context.Context) {
	restoreMember1, err := h.dataPlaneClient1.PingcapV1alpha1().Restores(fakeTcNamespace1).Get(ctx, h.restoreMemberName1, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	pingcapv1alpha1.UpdateRestoreCondition(&restoreMember1.Status, &pingcapv1alpha1.RestoreCondition{
		Type:   pingcapv1alpha1.RestoreComplete,
		Status: corev1.ConditionTrue,
	})
	_, err = h.dataPlaneClient1.PingcapV1alpha1().Restores(fakeTcNamespace1).UpdateStatus(ctx, restoreMember1, metav1.UpdateOptions{})
	h.g.Expect(err).To(gomega.BeNil())

	restoreMember2, err := h.dataPlaneClient2.PingcapV1alpha1().Restores(fakeTcNamespace2).Get(ctx, h.restoreMemberName2, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	pingcapv1alpha1.UpdateRestoreCondition(&restoreMember2.Status, &pingcapv1alpha1.RestoreCondition{
		Type:   pingcapv1alpha1.RestoreComplete,
		Status: corev1.ConditionTrue,
	})
	_, err = h.dataPlaneClient2.PingcapV1alpha1().Restores(fakeTcNamespace2).UpdateStatus(ctx, restoreMember2, metav1.UpdateOptions{})
	h.g.Expect(err).To(gomega.BeNil())

	restoreMember3, err := h.dataPlaneClient3.PingcapV1alpha1().Restores(fakeTcNamespace3).Get(ctx, h.restoreMemberName3, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	pingcapv1alpha1.UpdateRestoreCondition(&restoreMember3.Status, &pingcapv1alpha1.RestoreCondition{
		Type:   pingcapv1alpha1.RestoreComplete,
		Status: corev1.ConditionTrue,
	})
	_, err = h.dataPlaneClient3.PingcapV1alpha1().Restores(fakeTcNamespace3).UpdateStatus(ctx, restoreMember3, metav1.UpdateOptions{})
	h.g.Expect(err).To(gomega.BeNil())
}

func (h *helper) setDataPlaneFailed(ctx context.Context) {
	restoreMember1, err := h.dataPlaneClient1.PingcapV1alpha1().Restores(fakeTcNamespace1).Get(ctx, h.restoreMemberName1, metav1.GetOptions{})
	h.g.Expect(err).To(gomega.BeNil())
	pingcapv1alpha1.UpdateRestoreCondition(&restoreMember1.Status, &pingcapv1alpha1.RestoreCondition{
		Type:   pingcapv1alpha1.RestoreFailed,
		Status: corev1.ConditionTrue,
	})
	_, err = h.dataPlaneClient1.PingcapV1alpha1().Restores(fakeTcNamespace1).UpdateStatus(ctx, restoreMember1, metav1.UpdateOptions{})
	h.g.Expect(err).To(gomega.BeNil())
}

func TestVolumeRestore(t *testing.T) {
	restoreName := "restore-1"
	restoreNamespace := "ns-1"
	ctx := context.Background()
	h := newHelper(t, restoreName, restoreNamespace)

	volumeRestore := h.createVolumeRestore(ctx)

	// run restore volume phase
	err := h.rm.Sync(volumeRestore)
	h.g.Expect(err).To(gomega.BeNil())
	h.assertRunRestoreVolume(ctx, volumeRestore)

	// wait restore volume phase
	err = h.rm.Sync(volumeRestore)
	h.g.Expect(err).To(gomega.HaveOccurred())

	// data plane volume complete, still need to wait tikv ready
	h.setDataPlaneVolumeComplete(ctx)
	err = h.rm.Sync(volumeRestore)
	h.g.Expect(err).To(gomega.HaveOccurred())

	// data plane tikv complete, run restore data phase
	h.setDataPlaneTikvComplete(ctx)
	err = h.rm.Sync(volumeRestore)
	h.g.Expect(err).To(gomega.BeNil())
	h.assertRunRestoreData(ctx, volumeRestore)

	// wait restore data phase
	err = h.rm.Sync(volumeRestore)
	h.g.Expect(err).To(gomega.HaveOccurred())

	// data plane data complete, run restore finish phase
	h.setDataPlaneDataComplete(ctx)
	err = h.rm.Sync(volumeRestore)
	h.g.Expect(err).To(gomega.BeNil())
	h.assertRunRestoreFinish(ctx, volumeRestore)

	// wait restore complete
	err = h.rm.Sync(volumeRestore)
	h.g.Expect(err).To(gomega.HaveOccurred())

	// data plane complete, restore complete
	h.setDataPlaneComplete(ctx)
	err = h.rm.Sync(volumeRestore)
	h.g.Expect(err).To(gomega.BeNil())
	h.assertRestoreComplete(volumeRestore)
}

func TestVolumeRestore_RestoreVolumeFailed(t *testing.T) {
	restoreName := "restore-1"
	restoreNamespace := "ns-1"
	ctx := context.Background()
	h := newHelper(t, restoreName, restoreNamespace)

	volumeRestore := h.createVolumeRestore(ctx)

	// run restore volume phase
	err := h.rm.Sync(volumeRestore)
	h.g.Expect(err).To(gomega.BeNil())
	h.assertRunRestoreVolume(ctx, volumeRestore)

	// restore volume failed, restore failed
	h.setDataPlaneFailed(ctx)
	err = h.rm.Sync(volumeRestore)
	h.g.Expect(err).To(gomega.BeNil())
	h.assertRestoreFailed(volumeRestore)
}

func TestVolumeRestore_RestoreDataFailed(t *testing.T) {
	restoreName := "restore-1"
	restoreNamespace := "ns-1"
	ctx := context.Background()
	h := newHelper(t, restoreName, restoreNamespace)

	volumeRestore := h.createVolumeRestore(ctx)

	// run restore volume phase
	err := h.rm.Sync(volumeRestore)
	h.g.Expect(err).To(gomega.BeNil())
	h.assertRunRestoreVolume(ctx, volumeRestore)

	// wait restore volume phase
	err = h.rm.Sync(volumeRestore)
	h.g.Expect(err).To(gomega.HaveOccurred())

	// data plane volume complete, still need to wait tikv ready
	h.setDataPlaneVolumeComplete(ctx)
	err = h.rm.Sync(volumeRestore)
	h.g.Expect(err).To(gomega.HaveOccurred())

	// data plane tikv complete, run restore data phase
	h.setDataPlaneTikvComplete(ctx)
	err = h.rm.Sync(volumeRestore)
	h.g.Expect(err).To(gomega.BeNil())
	h.assertRunRestoreData(ctx, volumeRestore)

	// restore data failed, restore failed
	h.setDataPlaneFailed(ctx)
	err = h.rm.Sync(volumeRestore)
	h.g.Expect(err).To(gomega.BeNil())
	h.assertRestoreFailed(volumeRestore)
}

func TestVolumeRestore_RestoreFinishFailed(t *testing.T) {
	restoreName := "restore-1"
	restoreNamespace := "ns-1"
	ctx := context.Background()
	h := newHelper(t, restoreName, restoreNamespace)

	volumeRestore := h.createVolumeRestore(ctx)

	// run restore volume phase
	err := h.rm.Sync(volumeRestore)
	h.g.Expect(err).To(gomega.BeNil())
	h.assertRunRestoreVolume(ctx, volumeRestore)

	// wait restore volume phase
	err = h.rm.Sync(volumeRestore)
	h.g.Expect(err).To(gomega.HaveOccurred())

	// data plane volume complete, still need to wait tikv ready
	h.setDataPlaneVolumeComplete(ctx)
	err = h.rm.Sync(volumeRestore)
	h.g.Expect(err).To(gomega.HaveOccurred())

	// data plane tikv complete, run restore data phase
	h.setDataPlaneTikvComplete(ctx)
	err = h.rm.Sync(volumeRestore)
	h.g.Expect(err).To(gomega.BeNil())
	h.assertRunRestoreData(ctx, volumeRestore)

	// wait restore data phase
	err = h.rm.Sync(volumeRestore)
	h.g.Expect(err).To(gomega.HaveOccurred())

	// data plane data complete, run restore finish phase
	h.setDataPlaneDataComplete(ctx)
	err = h.rm.Sync(volumeRestore)
	h.g.Expect(err).To(gomega.BeNil())
	h.assertRunRestoreFinish(ctx, volumeRestore)

	// restore finish failed, restore failed
	h.setDataPlaneFailed(ctx)
	err = h.rm.Sync(volumeRestore)
	h.g.Expect(err).To(gomega.BeNil())
	h.assertRestoreFailed(volumeRestore)
}

func generateVolumeRestore(restoreName, restoreNamespace string) *v1alpha1.VolumeRestore {
	return &v1alpha1.VolumeRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      restoreName,
			Namespace: restoreNamespace,
		},
		Spec: v1alpha1.VolumeRestoreSpec{
			Clusters: []v1alpha1.VolumeRestoreMemberCluster{
				{
					K8sClusterName: controller.FakeDataPlaneName1,
					TCName:         fakeTcName1,
					TCNamespace:    fakeTcNamespace1,
					Backup: v1alpha1.VolumeRestoreMemberBackupInfo{
						StorageProvider: pingcapv1alpha1.StorageProvider{
							S3: &pingcapv1alpha1.S3StorageProvider{
								Bucket: fakeBackupBucket,
								Prefix: fakeBackupPrefix1,
							},
						},
					},
				},
				{
					K8sClusterName: controller.FakeDataPlaneName2,
					TCName:         fakeTcName2,
					TCNamespace:    fakeTcNamespace2,
					Backup: v1alpha1.VolumeRestoreMemberBackupInfo{
						StorageProvider: pingcapv1alpha1.StorageProvider{
							S3: &pingcapv1alpha1.S3StorageProvider{
								Bucket: fakeBackupBucket,
								Prefix: fakeBackupPrefix2,
							},
						},
					},
				},
				{
					K8sClusterName: controller.FakeDataPlaneName3,
					TCName:         fakeTcName3,
					TCNamespace:    fakeTcNamespace3,
					Backup: v1alpha1.VolumeRestoreMemberBackupInfo{
						StorageProvider: pingcapv1alpha1.StorageProvider{
							S3: &pingcapv1alpha1.S3StorageProvider{
								Bucket: fakeBackupBucket,
								Prefix: fakeBackupPrefix3,
							},
						},
					},
				},
			},
			Template: v1alpha1.VolumeRestoreMemberSpec{
				BR: &v1alpha1.BRConfig{
					SendCredToTikv: pointer.BoolPtr(false),
				},
				ServiceAccount: "tidb-backup-manager",
			},
		},
	}
}
