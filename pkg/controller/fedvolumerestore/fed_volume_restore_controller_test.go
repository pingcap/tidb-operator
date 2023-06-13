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

package fedvolumerestore

import (
	"fmt"
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1"
	pingcapv1alpha1 "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func TestUpdateRestore(t *testing.T) {
	type testcase struct {
		name          string
		deleted       bool
		conditionType v1alpha1.VolumeRestoreConditionType
		expectFn      func(g gomega.Gomega, ctl *Controller)
	}

	testFn := func(c *testcase) {
		t.Log("test:", c.name)
		g := gomega.NewWithT(t)
		ctl := NewController(controller.NewFakeBrFedDependencies())
		volumeRestore := newVolumeRestore()
		if c.deleted {
			volumeRestore.DeletionTimestamp = &metav1.Time{Time: time.Now()}
		}
		if len(c.conditionType) > 0 {
			volumeRestore.Status.Conditions = append(volumeRestore.Status.Conditions, v1alpha1.VolumeRestoreCondition{
				Type:   c.conditionType,
				Status: corev1.ConditionTrue,
			})
		}

		ctl.updateRestore(volumeRestore)
		c.expectFn(g, ctl)
	}

	testcases := []*testcase{
		{
			name:    "deleted restore",
			deleted: true,
			expectFn: func(g gomega.Gomega, ctl *Controller) {
				g.Expect(ctl.queue.Len()).To(gomega.Equal(1))
			},
		},
		{
			name: "new restore",
			expectFn: func(g gomega.Gomega, ctl *Controller) {
				g.Expect(ctl.queue.Len()).To(gomega.Equal(1))
			},
		},
		{
			name:          "running restore",
			conditionType: v1alpha1.VolumeRestoreRunning,
			expectFn: func(g gomega.Gomega, ctl *Controller) {
				g.Expect(ctl.queue.Len()).To(gomega.Equal(1))
			},
		},
		{
			name:          "complete restore",
			conditionType: v1alpha1.VolumeRestoreComplete,
			expectFn: func(g gomega.Gomega, ctl *Controller) {
				g.Expect(ctl.queue.Len()).To(gomega.Equal(0))
			},
		},
		{
			name:          "failed restore",
			conditionType: v1alpha1.VolumeRestoreFailed,
			expectFn: func(g gomega.Gomega, ctl *Controller) {
				g.Expect(ctl.queue.Len()).To(gomega.Equal(0))
			},
		},
	}

	for _, c := range testcases {
		testFn(c)
	}
}

func TestControllerSync(t *testing.T) {
	type testcase struct {
		name          string
		invalidKey    bool
		addToInformer bool
		syncError     bool
		expectFn      func(g gomega.Gomega, err error)
	}

	testFn := func(c *testcase) {
		t.Log("test:", c.name)
		g := gomega.NewWithT(t)
		ctl := NewController(controller.NewFakeBrFedDependencies())
		restoreInformer := ctl.deps.InformerFactory.Federation().V1alpha1().VolumeRestores()
		ctl.control = NewFakeRestoreControl(restoreInformer)
		volumeRestore := newVolumeRestore()

		var key string
		if c.invalidKey {
			key = fmt.Sprintf("k/%s/%s", volumeRestore.Namespace, volumeRestore.Name)
		} else {
			key = fmt.Sprintf("%s/%s", volumeRestore.Namespace, volumeRestore.Name)
		}
		if c.addToInformer {
			err := restoreInformer.Informer().GetIndexer().Add(volumeRestore)
			g.Expect(err).To(gomega.BeNil())
		}
		if c.syncError {
			ctl.control.(*FakeRestoreControl).SetUpdateRestoreError(errors.New("update error"), 0)
		}

		err := ctl.sync(key)
		c.expectFn(g, err)
	}

	testcases := []*testcase{
		{
			name:       "invalid key",
			invalidKey: true,
			expectFn: func(g gomega.Gomega, err error) {
				g.Expect(err).To(gomega.HaveOccurred())
			},
		},
		{
			name: "key not found",
			expectFn: func(g gomega.Gomega, err error) {
				g.Expect(err).To(gomega.BeNil())
			},
		},
		{
			name:          "sync normally",
			addToInformer: true,
			expectFn: func(g gomega.Gomega, err error) {
				g.Expect(err).To(gomega.BeNil())
			},
		},
		{
			name:          "sync error",
			addToInformer: true,
			syncError:     true,
			expectFn: func(g gomega.Gomega, err error) {
				g.Expect(err).To(gomega.HaveOccurred())
			},
		},
	}

	for _, c := range testcases {
		testFn(c)
	}
}

func newVolumeRestore() *v1alpha1.VolumeRestore {
	return &v1alpha1.VolumeRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-restore",
			Namespace: "test-ns",
		},
		Spec: v1alpha1.VolumeRestoreSpec{
			Clusters: []v1alpha1.VolumeRestoreMemberCluster{
				{
					K8sClusterName: controller.FakeDataPlaneName1,
					Backup: v1alpha1.VolumeRestoreMemberBackupInfo{
						StorageProvider: pingcapv1alpha1.StorageProvider{
							S3: &pingcapv1alpha1.S3StorageProvider{
								Prefix: "prefix-1",
							},
						},
					},
				},
				{
					K8sClusterName: controller.FakeDataPlaneName2,
					Backup: v1alpha1.VolumeRestoreMemberBackupInfo{
						StorageProvider: pingcapv1alpha1.StorageProvider{
							S3: &pingcapv1alpha1.S3StorageProvider{
								Prefix: "prefix-2",
							},
						},
					},
				},
				{
					K8sClusterName: controller.FakeDataPlaneName3,
					Backup: v1alpha1.VolumeRestoreMemberBackupInfo{
						StorageProvider: pingcapv1alpha1.StorageProvider{
							S3: &pingcapv1alpha1.S3StorageProvider{
								Prefix: "prefix-3",
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
