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

package fedvolumebackup

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

func TestUpdateBackup(t *testing.T) {
	type testcase struct {
		name          string
		deleted       bool
		conditionType v1alpha1.VolumeBackupConditionType
		expectFn      func(g gomega.Gomega, ctl *Controller)
	}

	testFn := func(c *testcase) {
		t.Log("test:", c.name)
		g := gomega.NewWithT(t)
		ctl := NewController(controller.NewFakeBrFedDependencies())
		volumeBackup := newVolumeBackup()
		if c.deleted {
			volumeBackup.DeletionTimestamp = &metav1.Time{Time: time.Now()}
		}
		if len(c.conditionType) > 0 {
			volumeBackup.Status.Conditions = append(volumeBackup.Status.Conditions, v1alpha1.VolumeBackupCondition{
				Type:   c.conditionType,
				Status: corev1.ConditionTrue,
			})
		}

		ctl.updateBackup(volumeBackup)
		c.expectFn(g, ctl)
	}

	testcases := []*testcase{
		{
			name:    "deleted backup",
			deleted: true,
			expectFn: func(g gomega.Gomega, ctl *Controller) {
				g.Expect(ctl.queue.Len()).To(gomega.Equal(1))
			},
		},
		{
			name: "new backup",
			expectFn: func(g gomega.Gomega, ctl *Controller) {
				g.Expect(ctl.queue.Len()).To(gomega.Equal(1))
			},
		},
		{
			name:          "running backup",
			conditionType: v1alpha1.VolumeBackupRunning,
			expectFn: func(g gomega.Gomega, ctl *Controller) {
				g.Expect(ctl.queue.Len()).To(gomega.Equal(1))
			},
		},
		{
			name:          "invalid backup",
			conditionType: v1alpha1.VolumeBackupInvalid,
			expectFn: func(g gomega.Gomega, ctl *Controller) {
				g.Expect(ctl.queue.Len()).To(gomega.Equal(0))
			},
		},
		{
			name:          "complete backup",
			conditionType: v1alpha1.VolumeBackupComplete,
			expectFn: func(g gomega.Gomega, ctl *Controller) {
				g.Expect(ctl.queue.Len()).To(gomega.Equal(0))
			},
		},
		{
			name:          "failed backup",
			conditionType: v1alpha1.VolumeBackupFailed,
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
		backupInformer := ctl.deps.InformerFactory.Federation().V1alpha1().VolumeBackups()
		ctl.control = NewFakeBackupControl(backupInformer)
		volumeBackup := newVolumeBackup()

		var key string
		if c.invalidKey {
			key = fmt.Sprintf("k/%s/%s", volumeBackup.Namespace, volumeBackup.Name)
		} else {
			key = fmt.Sprintf("%s/%s", volumeBackup.Namespace, volumeBackup.Name)
		}
		if c.addToInformer {
			err := backupInformer.Informer().GetIndexer().Add(volumeBackup)
			g.Expect(err).To(gomega.BeNil())
		}
		if c.syncError {
			ctl.control.(*FakeBackupControl).SetUpdateBackupError(errors.New("update error"), 0)
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

func newVolumeBackup() *v1alpha1.VolumeBackup {
	return &v1alpha1.VolumeBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-backup",
			Namespace: "test-namespace",
		},
		Spec: v1alpha1.VolumeBackupSpec{
			Clusters: []v1alpha1.VolumeBackupMemberCluster{
				{
					K8sClusterName: controller.FakeDataPlaneName1,
				},
				{
					K8sClusterName: controller.FakeDataPlaneName2,
				},
				{
					K8sClusterName: controller.FakeDataPlaneName3,
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
