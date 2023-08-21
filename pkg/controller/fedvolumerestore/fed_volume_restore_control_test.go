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
	"context"
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/client/federation/clientset/versioned/fake"
	"github.com/pingcap/tidb-operator/pkg/fedvolumebackup/restore"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRestoreControlUpdateRestore(t *testing.T) {
	type testcase struct {
		name           string
		updateError    bool
		updateStatus   bool
		newRestore     bool
		cleanedRestore bool
		expectFn       func(g gomega.Gomega, err error, statusUpdated bool, finalizers []string)
	}

	testFn := func(c *testcase) {
		t.Log("test:", c.name)
		g := gomega.NewWithT(t)
		cli := fake.NewSimpleClientset()
		rm := restore.NewFakeRestoreManager()
		ctl := NewDefaultVolumeRestoreControl(cli, rm)
		volumeRestore := newVolumeRestore()

		if !c.newRestore {
			volumeRestore.Finalizers = append(volumeRestore.Finalizers, label.VolumeRestoreFederationFinalizer)
		}
		if c.cleanedRestore {
			volumeRestore.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			v1alpha1.UpdateVolumeRestoreCondition(&volumeRestore.Status, &v1alpha1.VolumeRestoreCondition{
				Type:   v1alpha1.VolumeRestoreCleaned,
				Status: corev1.ConditionTrue,
			})
		}
		if c.updateError {
			rm.SetSyncError(errors.New("update error"))
		}
		if c.updateStatus {
			rm.SetUpdateStatus()
		}

		_, err := cli.FederationV1alpha1().VolumeRestores(volumeRestore.Namespace).Create(context.Background(), volumeRestore, metav1.CreateOptions{})
		g.Expect(err).To(gomega.BeNil())
		err = ctl.UpdateRestore(volumeRestore)
		c.expectFn(g, err, rm.IsStatusUpdated(), volumeRestore.Finalizers)
	}

	testcases := []*testcase{
		{
			name:       "new volume restore",
			newRestore: true,
			expectFn: func(g gomega.Gomega, err error, statusUpdated bool, finalizers []string) {
				g.Expect(err).To(gomega.BeNil())
				g.Expect(statusUpdated).To(gomega.BeFalse())
				g.Expect(finalizers).NotTo(gomega.BeEmpty())
			},
		},
		{
			name: "old restore with status not changed",
			expectFn: func(g gomega.Gomega, err error, statusUpdated bool, finalizers []string) {
				g.Expect(err).To(gomega.BeNil())
				g.Expect(statusUpdated).To(gomega.BeFalse())
				g.Expect(finalizers).NotTo(gomega.BeEmpty())
			},
		},
		{
			name:         "old restore with status changed",
			updateStatus: true,
			expectFn: func(g gomega.Gomega, err error, statusUpdated bool, finalizers []string) {
				g.Expect(err).To(gomega.BeNil())
				g.Expect(statusUpdated).To(gomega.BeTrue())
				g.Expect(finalizers).NotTo(gomega.BeEmpty())
			},
		},
		{
			name:        "old restore with error",
			updateError: true,
			expectFn: func(g gomega.Gomega, err error, statusUpdated bool, finalizers []string) {
				g.Expect(err).To(gomega.HaveOccurred())
				g.Expect(statusUpdated).To(gomega.BeFalse())
				g.Expect(finalizers).NotTo(gomega.BeEmpty())
			},
		},
		{
			name:         "old restore with error and status changed",
			updateStatus: true,
			updateError:  true,
			expectFn: func(g gomega.Gomega, err error, statusUpdated bool, finalizers []string) {
				g.Expect(err).To(gomega.HaveOccurred())
				g.Expect(statusUpdated).To(gomega.BeTrue())
				g.Expect(finalizers).NotTo(gomega.BeEmpty())
			},
		},
		{
			name:           "cleaned restore",
			cleanedRestore: true,
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

func TestRestoreControlUpdateRestoreStatus(t *testing.T) {
	g := gomega.NewWithT(t)
	cli := fake.NewSimpleClientset()
	rm := restore.NewFakeRestoreManager()
	ctl := NewDefaultVolumeRestoreControl(cli, rm)

	volumeRestore := newVolumeRestore()
	status := volumeRestore.Status.DeepCopy()
	v1alpha1.UpdateVolumeRestoreCondition(status, &v1alpha1.VolumeRestoreCondition{
		Type:   v1alpha1.VolumeRestoreComplete,
		Status: corev1.ConditionTrue,
	})

	err := ctl.UpdateStatus(volumeRestore, status)
	g.Expect(err).To(gomega.BeNil())
	g.Expect(rm.IsStatusUpdated()).To(gomega.BeTrue())
}
