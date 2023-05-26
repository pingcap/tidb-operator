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
	"context"
	"fmt"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
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

	// create a pod with failed status in the pod informer.
	createFailedPod := func(g *GomegaWithT, rtc *Controller, restore *v1alpha1.Restore) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      restore.Name,
				Namespace: restore.Namespace,
				Labels:    label.NewRestore().Instance(restore.GetInstanceName()).RestoreJob().Restore(restore.Name),
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodFailed,
			},
		}
		_, err := rtc.deps.KubeClientset.CoreV1().Pods(restore.Namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
		g.Expect(err).To(Succeed())
		rtc.deps.KubeInformerFactory.Start(context.TODO().Done())
		cache.WaitForCacheSync(context.TODO().Done(), rtc.deps.KubeInformerFactory.Core().V1().Pods().Informer().HasSynced)
	}

	updatingToFail := func(g *GomegaWithT, rtc *Controller, restore *v1alpha1.Restore) {
		control := rtc.control.(*FakeRestoreControl)
		condition := control.condition
		g.Expect(condition).NotTo(Equal(nil))
		g.Expect(condition.Type).To(Equal(v1alpha1.RestoreFailed))
		g.Expect(condition.Reason).To(Equal("AlreadyFailed"))
	}

	tests := []struct {
		name           string
		conditionType  v1alpha1.RestoreConditionType // only one condition used now.
		beforeUpdateFn func(*GomegaWithT, *Controller, *v1alpha1.Restore)
		expectFn       func(*GomegaWithT, *Controller)
		afterUpdateFn  func(*GomegaWithT, *Controller, *v1alpha1.Restore)
	}{
		{
			name:          "restore is invalid",
			conditionType: v1alpha1.RestoreInvalid,
			expectFn: func(g *GomegaWithT, rtc *Controller) {
				g.Expect(rtc.queue.Len()).To(Equal(0))
			},
		},
		{
			name:          "restore has been completed",
			conditionType: v1alpha1.RestoreComplete,
			expectFn: func(g *GomegaWithT, rtc *Controller) {
				g.Expect(rtc.queue.Len()).To(Equal(0))
			},
		},
		{
			name:          "restore has been scheduled",
			conditionType: v1alpha1.RestoreScheduled,
			expectFn: func(g *GomegaWithT, rtc *Controller) {
				g.Expect(rtc.queue.Len()).To(Equal(0))
			},
		},
		{
			name:          "restore is newly created",
			conditionType: "", // no condition
			expectFn: func(g *GomegaWithT, rtc *Controller) {
				g.Expect(rtc.queue.Len()).To(Equal(1))
			},
		},
		{
			name:          "restore has been failed",
			conditionType: v1alpha1.RestoreFailed,
			expectFn: func(g *GomegaWithT, rtc *Controller) {
				g.Expect(rtc.queue.Len()).To(Equal(0))
			},
		},
		{
			name:           "restore has been scheduled with failed pod",
			conditionType:  v1alpha1.RestoreScheduled,
			beforeUpdateFn: createFailedPod,
			expectFn: func(g *GomegaWithT, rtc *Controller) {
				g.Expect(rtc.queue.Len()).To(Equal(0))
			},
			afterUpdateFn: updatingToFail,
		},
		{
			name:           "restore has been running with failed pod",
			conditionType:  v1alpha1.RestoreRunning,
			beforeUpdateFn: createFailedPod,
			expectFn: func(g *GomegaWithT, rtc *Controller) {
				g.Expect(rtc.queue.Len()).To(Equal(0))
			},
			afterUpdateFn: updatingToFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			restore := newRestore()
			rtc, _, _ := newFakeRestoreController()

			if len(tt.conditionType) > 0 {
				restore.Status.Conditions = []v1alpha1.RestoreCondition{
					{
						Type:   tt.conditionType,
						Status: corev1.ConditionTrue,
					},
				}
			}

			if tt.beforeUpdateFn != nil {
				tt.beforeUpdateFn(g, rtc, restore)
			}

			rtc.updateRestore(restore)
			if tt.expectFn != nil {
				tt.expectFn(g, rtc)
			}

			if tt.afterUpdateFn != nil {
				tt.afterUpdateFn(g, rtc, restore)
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
			To: &v1alpha1.TiDBAccessConfig{
				Host:       "10.1.1.2",
				Port:       v1alpha1.DefaultTiDBServerPort,
				User:       v1alpha1.DefaultTidbUser,
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
