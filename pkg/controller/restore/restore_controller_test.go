// Copyright 2020 PingCAP, Inc.
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
	"fmt"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	"github.com/pingcap/tidb-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"
)

func TestRestoreControllerEnqueueRestore(t *testing.T) {
	g := NewGomegaWithT(t)
	restore := newRestore()
	rtc, _, _ := newFakeRestoreController()
	rtc.enqueueRestore(restore)
	g.Expect(rtc.queue.Len()).To(Equal(1))
}

func TestRestoreControllerEnqueueRestoreFailed(t *testing.T) {
	g := NewGomegaWithT(t)
	rtc, _, _ := newFakeRestoreController()
	rtc.enqueueRestore(struct{}{})
	g.Expect(rtc.queue.Len()).To(Equal(0))
}

func TestRestoreControllerUpdateRestore(t *testing.T) {
	g := NewGomegaWithT(t)

	tests := []struct {
		name                    string
		restoreIsInvalid        bool
		restoreHasBeenCompleted bool
		restoreHasBeenScheduled bool
		expectFn                func(*GomegaWithT, *Controller)
	}{
		{
			name:                    "restore is invalid",
			restoreIsInvalid:        true,
			restoreHasBeenCompleted: false,
			restoreHasBeenScheduled: false,
			expectFn: func(g *GomegaWithT, rtc *Controller) {
				g.Expect(rtc.queue.Len()).To(Equal(0))
			},
		},
		{
			name:                    "restore has been completed",
			restoreIsInvalid:        false,
			restoreHasBeenCompleted: true,
			restoreHasBeenScheduled: false,
			expectFn: func(g *GomegaWithT, rtc *Controller) {
				g.Expect(rtc.queue.Len()).To(Equal(0))
			},
		},
		{
			name:                    "restore has been scheduled",
			restoreIsInvalid:        false,
			restoreHasBeenCompleted: false,
			restoreHasBeenScheduled: true,
			expectFn: func(g *GomegaWithT, rtc *Controller) {
				g.Expect(rtc.queue.Len()).To(Equal(0))
			},
		},
		{
			name:                    "restore is newly created",
			restoreIsInvalid:        false,
			restoreHasBeenCompleted: false,
			restoreHasBeenScheduled: false,
			expectFn: func(g *GomegaWithT, rtc *Controller) {
				g.Expect(rtc.queue.Len()).To(Equal(1))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			restore := newRestore()
			rtc, _, _ := newFakeRestoreController()

			if tt.restoreIsInvalid {
				restore.Status.Conditions = []v1alpha1.RestoreCondition{
					{
						Type:   v1alpha1.RestoreInvalid,
						Status: corev1.ConditionTrue,
					},
				}
			}

			if tt.restoreHasBeenCompleted {
				restore.Status.Conditions = []v1alpha1.RestoreCondition{
					{
						Type:   v1alpha1.RestoreComplete,
						Status: corev1.ConditionTrue,
					},
				}
			}

			if tt.restoreHasBeenScheduled {
				restore.Status.Conditions = []v1alpha1.RestoreCondition{
					{
						Type:   v1alpha1.RestoreScheduled,
						Status: corev1.ConditionTrue,
					},
				}
			}

			rtc.updateRestore(restore)
			if tt.expectFn != nil {
				tt.expectFn(g, rtc)
			}
		})
	}
}

func TestRestoreControllerSync(t *testing.T) {
	g := NewGomegaWithT(t)
	tests := []struct {
		name                 string
		addRestoreToIndexer  bool
		errWhenUpdateRestore bool
		invalidKeyFn         func(restore *v1alpha1.Restore) string
		errExpectFn          func(*GomegaWithT, error)
	}{
		{
			name:                 "normal",
			addRestoreToIndexer:  true,
			errWhenUpdateRestore: false,
			invalidKeyFn:         nil,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
		},
		{
			name:                 "invalid restore key",
			addRestoreToIndexer:  true,
			errWhenUpdateRestore: false,
			invalidKeyFn: func(restore *v1alpha1.Restore) string {
				return fmt.Sprintf("test/demo/%s", restore.GetName())
			},
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
			},
		},
		{
			name:                 "can't found restore",
			addRestoreToIndexer:  false,
			errWhenUpdateRestore: false,
			invalidKeyFn:         nil,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
		},
		{
			name:                 "update restore failed",
			addRestoreToIndexer:  true,
			errWhenUpdateRestore: true,
			invalidKeyFn:         nil,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "update restore failed")).To(Equal(true))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restore := newRestore()
			rtc, restoreIndexer, restoreControl := newFakeRestoreController()

			if tt.addRestoreToIndexer {
				err := restoreIndexer.Add(restore)
				g.Expect(err).NotTo(HaveOccurred())
			}

			key, _ := cache.DeletionHandlingMetaNamespaceKeyFunc(restore)
			if tt.invalidKeyFn != nil {
				key = tt.invalidKeyFn(restore)
			}

			if tt.errWhenUpdateRestore {
				restoreControl.SetUpdateRestoreError(fmt.Errorf("update restore failed"), 0)
			}

			err := rtc.sync(key)

			if tt.errExpectFn != nil {
				tt.errExpectFn(g, err)
			}
		})
	}
}

func newFakeRestoreController() (*Controller, cache.Indexer, *FakeRestoreControl) {
	fakeDeps := controller.NewFakeDependencies()
	rtc := NewController(fakeDeps)
	restoreInformer := fakeDeps.InformerFactory.Pingcap().V1alpha1().Restores()
	restoreControl := NewFakeRestoreControl(restoreInformer)
	rtc.control = restoreControl
	return rtc, restoreInformer.Informer().GetIndexer(), restoreControl
}

func newRestore() *v1alpha1.Restore {
	return &v1alpha1.Restore{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Restore",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-restore",
			Namespace: corev1.NamespaceDefault,
			UID:       "test-rt",
		},
		Spec: v1alpha1.RestoreSpec{
			To: v1alpha1.TiDBAccessConfig{
				Host:       "10.1.1.2",
				Port:       constants.DefaultTidbPort,
				User:       constants.DefaultTidbUser,
				SecretName: "demo1-tidb-secret",
			},
			StorageProvider: v1alpha1.StorageProvider{
				S3: &v1alpha1.S3StorageProvider{
					Provider:   v1alpha1.S3StorageProviderTypeCeph,
					Endpoint:   "http://10.0.0.1",
					Bucket:     "test1-demo1",
					SecretName: "demo",
				},
			},
			StorageClassName: pointer.StringPtr("local-storage"),
			StorageSize:      "1Gi",
		},
	}
}
