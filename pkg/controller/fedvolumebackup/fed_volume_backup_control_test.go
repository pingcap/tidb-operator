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
	"context"
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/client/federation/clientset/versioned/fake"
	"github.com/pingcap/tidb-operator/pkg/fedvolumebackup/backup"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBackupControlUpdateBackup(t *testing.T) {
	type testcase struct {
		name          string
		updateError   bool
		updateStatus  bool
		newBackup     bool
		cleanedBackup bool
		expectFn      func(g gomega.Gomega, err error, statusUpdated bool, finalizers []string)
	}

	testFn := func(c *testcase) {
		t.Log("test:", c.name)
		g := gomega.NewWithT(t)
		cli := fake.NewSimpleClientset()
		bm := backup.NewFakeBackupManager()
		ctl := NewDefaultVolumeBackupControl(cli, bm)
		volumeBackup := newVolumeBackup()

		if !c.newBackup {
			volumeBackup.Finalizers = append(volumeBackup.Finalizers, label.BackupProtectionFinalizer)
		}
		if c.cleanedBackup {
			volumeBackup.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			v1alpha1.UpdateVolumeBackupCondition(&volumeBackup.Status, &v1alpha1.VolumeBackupCondition{
				Type:   v1alpha1.VolumeBackupCleaned,
				Status: corev1.ConditionTrue,
			})
		}
		if c.updateError {
			bm.SetSyncError(errors.New("update error"))
		}
		if c.updateStatus {
			bm.SetUpdateStatus()
		}

		_, err := cli.FederationV1alpha1().VolumeBackups(volumeBackup.Namespace).Create(context.Background(), volumeBackup, metav1.CreateOptions{})
		g.Expect(err).To(gomega.BeNil())
		err = ctl.UpdateBackup(volumeBackup)
		c.expectFn(g, err, bm.IsStatusUpdated(), volumeBackup.Finalizers)
	}

	testcases := []*testcase{
		{
			name:      "new volume backup",
			newBackup: true,
			expectFn: func(g gomega.Gomega, err error, statusUpdated bool, finalizers []string) {
				g.Expect(err).To(gomega.BeNil())
				g.Expect(statusUpdated).To(gomega.BeFalse())
				g.Expect(finalizers).NotTo(gomega.BeEmpty())
			},
		},
		{
			name: "old backup with status not changed",
			expectFn: func(g gomega.Gomega, err error, statusUpdated bool, finalizers []string) {
				g.Expect(err).To(gomega.BeNil())
				g.Expect(statusUpdated).To(gomega.BeFalse())
				g.Expect(finalizers).NotTo(gomega.BeEmpty())
			},
		},
		{
			name:         "old backup with status changed",
			updateStatus: true,
			expectFn: func(g gomega.Gomega, err error, statusUpdated bool, finalizers []string) {
				g.Expect(err).To(gomega.BeNil())
				g.Expect(statusUpdated).To(gomega.BeTrue())
				g.Expect(finalizers).NotTo(gomega.BeEmpty())
			},
		},
		{
			name:        "old backup with error",
			updateError: true,
			expectFn: func(g gomega.Gomega, err error, statusUpdated bool, finalizers []string) {
				g.Expect(err).To(gomega.HaveOccurred())
				g.Expect(statusUpdated).To(gomega.BeFalse())
				g.Expect(finalizers).NotTo(gomega.BeEmpty())
			},
		},
		{
			name:         "old backup with error and status changed",
			updateStatus: true,
			updateError:  true,
			expectFn: func(g gomega.Gomega, err error, statusUpdated bool, finalizers []string) {
				g.Expect(err).To(gomega.HaveOccurred())
				g.Expect(statusUpdated).To(gomega.BeFalse())
				g.Expect(finalizers).NotTo(gomega.BeEmpty())
			},
		},
		{
			name:          "cleaned backup",
			cleanedBackup: true,
			expectFn: func(g gomega.Gomega, err error, statusUpdated bool, finalizers []string) {
				g.Expect(err).To(gomega.HaveOccurred())
				g.Expect(statusUpdated).To(gomega.BeFalse())
				g.Expect(finalizers).To(gomega.BeEmpty())
			},
		},
	}

	for _, c := range testcases {
		testFn(c)
	}
}

func TestBackupControlUpdateBackupStatus(t *testing.T) {
	g := gomega.NewWithT(t)
	cli := fake.NewSimpleClientset()
	bm := backup.NewFakeBackupManager()
	ctl := NewDefaultVolumeBackupControl(cli, bm)

	volumeBackup := newVolumeBackup()
	status := volumeBackup.Status.DeepCopy()
	v1alpha1.UpdateVolumeBackupCondition(status, &v1alpha1.VolumeBackupCondition{
		Type:   v1alpha1.VolumeBackupComplete,
		Status: corev1.ConditionTrue,
	})

	err := ctl.UpdateStatus(volumeBackup, status)
	g.Expect(err).To(gomega.BeNil())
	g.Expect(bm.IsStatusUpdated()).To(gomega.BeTrue())
}
