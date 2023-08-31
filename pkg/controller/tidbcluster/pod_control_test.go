// Copyright 2021 PingCAP, Inc.
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

package tidbcluster

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pingcap/kvproto/pkg/pdpb"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/pkg/tikvapi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type kvClient struct {
	leaderCount int32
}

var _ tikvapi.TiKVClient = &kvClient{}

func (c *kvClient) GetLeaderCount() (int, error) {
	count := atomic.LoadInt32(&c.leaderCount)
	return int(count), nil
}

func TestTiKVPodSyncForEviction(t *testing.T) {
	interval := time.Millisecond * 100
	timeout := time.Minute * 1
	g := NewGomegaWithT(t)

	tc := newTidbCluster()
	pod := newTiKVPod(tc)
	tc.Status.TiKV = v1alpha1.TiKVStatus{
		Stores: map[string]v1alpha1.TiKVStore{
			"0": {
				PodName: pod.Name,
				ID:      "0",
			},
		},
	}
	deps := controller.NewFakeDependencies()
	fakeTiKVControl := deps.TiKVControl.(*tikvapi.FakeTiKVControl)
	kvClient := &kvClient{}
	fakeTiKVControl.SetTiKVPodClient(tc.Namespace, tc.Name, pod.Name, kvClient)
	c := NewPodController(deps)
	pdClient := pdapi.NewFakePDClient()
	c.testPDClient = pdClient
	c.recheckLeaderCountDuration = time.Millisecond * 100
	c.recheckClusterStableDuration = time.Millisecond * 100
	var tikvStatus atomic.Value
	tikvStatus.Store(v1alpha1.TiKVStateDown)
	pdClient.AddReaction(pdapi.GetStoresActionType, func(action *pdapi.Action) (interface{}, error) {
		storesInfo := &pdapi.StoresInfo{
			Stores: []*pdapi.StoreInfo{
				{
					Store: &pdapi.MetaStore{
						StateName: tikvStatus.Load().(string),
					},
				},
			},
		}
		return storesInfo, nil
	})

	stop := make(chan struct{})
	go func() {
		deps.KubeInformerFactory.Start(stop)
	}()
	deps.KubeInformerFactory.WaitForCacheSync(stop)
	go func() {
		deps.InformerFactory.Start(stop)
	}()
	deps.InformerFactory.WaitForCacheSync(stop)

	defer close(stop)
	go func() {
		c.Run(1, stop)
	}()

	ctx := context.Background()
	tc, err := deps.Clientset.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(ctx, tc, metav1.CreateOptions{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Eventually(func() error {
		_, err := deps.TiDBClusterLister.TidbClusters(tc.Namespace).Get(tc.Name)
		return err
	}, timeout, interval).Should(Succeed())

	pod, err = deps.KubeClientset.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Eventually(func() error {
		_, err := deps.PodLister.Pods(tc.Namespace).Get(pod.Name)
		return err
	}, timeout, interval).Should(Succeed())

	// trigger an restart
	atomic.StoreInt32(&kvClient.leaderCount, 100)
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	// add expired annotation
	pod.Annotations[v1alpha1.EvictLeaderAnnKey] = v1alpha1.EvictLeaderValueDeletePod
	pod.Annotations[v1alpha1.TiKVEvictLeaderExpirationTimeAnnKey] = metav1.Now().Time.Add(-time.Minute).UTC().Format(time.RFC3339)
	pod, err = deps.KubeClientset.CoreV1().Pods(pod.Namespace).Update(ctx, pod, metav1.UpdateOptions{})
	g.Expect(err).Should(Succeed())

	g.Consistently(func() int {
		stat := c.getPodStat(pod)
		return stat.observeAnnotationCounts
	}, time.Second*5, interval).Should(Equal(0), "should not observe pod annotation as it is added expired")

	// add annotation w/o expiration
	pod.Annotations[v1alpha1.EvictLeaderAnnKey] = v1alpha1.EvictLeaderValueDeletePod
	delete(pod.Annotations, v1alpha1.TiKVEvictLeaderExpirationTimeAnnKey)
	pod, err = deps.KubeClientset.CoreV1().Pods(pod.Namespace).Update(ctx, pod, metav1.UpdateOptions{})
	g.Expect(err).Should(Succeed())

	g.Eventually(func() int {
		stat := c.getPodStat(pod)
		return stat.observeAnnotationCounts
	}, timeout, interval).ShouldNot(Equal(0), "should observe pod annotation")

	_, err = deps.KubeClientset.CoreV1().Pods(tc.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
	g.Expect(err).Should(Succeed())

	atomic.StoreInt32(&kvClient.leaderCount, 0)

	g.Consistently(func() bool {
		_, err := deps.KubeClientset.CoreV1().Pods(tc.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
		return err == nil
	}, time.Second*5, interval).Should(BeTrue(), "should not delete pod while cluster is unstable")

	tikvStatus.Store(v1alpha1.TiKVStateUp)
	g.Eventually(func() bool {
		_, err := deps.KubeClientset.CoreV1().Pods(tc.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
		return errors.IsNotFound(err)
	}, timeout, interval).Should(BeTrue(), "should delete pod if leader count is 0 and cluster is stable")

	pod = newTiKVPod(tc)
	pod, err = deps.KubeClientset.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Eventually(func() int {
		stat := c.getPodStat(pod)
		return stat.finishAnnotationCounts
	}, timeout, interval).ShouldNot(Equal(0), "should finish annotation")
}

func TestTiKVPodSyncForReplaceVolume(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name               string
		storeState         string
		storeLabel         string
		storeStatus        string
		expectRequeue      bool
		pvcPodDeleted      bool
		deletedStore       uint64
		extraNotReadyStore bool
	}

	testFn := func(tt testcase, t *testing.T) {
		tc := newTidbCluster()
		pod := newTiKVPod(tc)
		if tt.storeLabel != "" {
			pod.Labels[label.StoreIDLabelKey] = tt.storeLabel
		}
		tc.Status.TiKV = v1alpha1.TiKVStatus{
			Stores: map[string]v1alpha1.TiKVStore{},
		}
		if tt.storeStatus != "" {
			tc.Status.TiKV.Stores[tt.storeStatus] = v1alpha1.TiKVStore{
				PodName: pod.Name,
				ID:      tt.storeStatus,
				State:   tt.storeState,
			}
		}
		if tt.extraNotReadyStore {
			tc.Status.TiKV.Stores["999"] = v1alpha1.TiKVStore{
				PodName: pod.Name,
				ID:      "999",
				State:   v1alpha1.TiKVStateDown,
			}
		}
		deps := controller.NewFakeDependencies()
		c := NewPodController(deps)
		pdClient := pdapi.NewFakePDClient()
		c.testPDClient = pdClient
		pdClient.AddReaction(pdapi.GetStoreActionType, func(action *pdapi.Action) (interface{}, error) {
			storeInfo := &pdapi.StoreInfo{
				Store: &pdapi.MetaStore{
					StateName: tt.storeState,
				},
			}
			return storeInfo, nil
		})
		var storeDeleted uint64 = 0
		pdClient.AddReaction(pdapi.DeleteStoreActionType, func(action *pdapi.Action) (interface{}, error) {
			storeDeleted = action.ID
			return nil, nil
		})
		deletePVCsAndPodCalled := false
		c.deletePVCsAndPodFn = func(deps *controller.Dependencies, ctx context.Context, pod *corev1.Pod, tc *v1alpha1.TidbCluster) (reconcile.Result, error) {
			deletePVCsAndPodCalled = true
			return reconcile.Result{}, nil
		}

		ctx := context.Background()
		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}
		pod.Annotations[v1alpha1.ReplaceVolumeAnnKey] = v1alpha1.ReplaceVolumeValueTrue
		result, err := c.syncTiKVPodForReplaceVolume(ctx, pod, tc)
		g.Expect(result.Requeue || result.RequeueAfter > 0 || err != nil).To(Equal(tt.expectRequeue))
		g.Expect(deletePVCsAndPodCalled).To(Equal(tt.pvcPodDeleted))
		g.Expect(storeDeleted).To(Equal(tt.deletedStore))
	}
	tests := []testcase{
		{
			name:          "Normal store up is deleted",
			storeState:    v1alpha1.TiKVStateUp,
			storeLabel:    "123",
			storeStatus:   "123",
			expectRequeue: true,
			pvcPodDeleted: false,
			deletedStore:  123,
		},
		{
			name:          "Normal store offline waiting",
			storeState:    v1alpha1.TiKVStateOffline,
			storeLabel:    "123",
			storeStatus:   "123",
			expectRequeue: true,
			pvcPodDeleted: false,
		},
		{
			name:          "Normal store tombstone pod deleted",
			storeState:    v1alpha1.TiKVStateTombstone,
			storeLabel:    "123",
			storeStatus:   "123",
			expectRequeue: false,
			pvcPodDeleted: true,
		},
		{
			name:          "Missing store status fall back label and tombstone pod deleted",
			storeState:    v1alpha1.TiKVStateTombstone,
			storeLabel:    "123",
			storeStatus:   "",
			expectRequeue: false,
			pvcPodDeleted: true,
		},
		{
			name:          "Missing label do not delete store",
			storeState:    v1alpha1.TiKVStateUp,
			storeLabel:    "",
			storeStatus:   "123",
			expectRequeue: true,
			pvcPodDeleted: false,
			deletedStore:  0, // No delete.
		},
		{
			name:               "Stores not ready do not delete",
			storeState:         v1alpha1.TiKVStateUp,
			storeLabel:         "123",
			storeStatus:        "123",
			expectRequeue:      true,
			pvcPodDeleted:      false,
			deletedStore:       0, // No delete.
			extraNotReadyStore: true,
		},
		{
			name:          "Conflicting store status delete with status",
			storeState:    v1alpha1.TiKVStateUp,
			storeLabel:    "124",
			storeStatus:   "123",
			expectRequeue: true,
			pvcPodDeleted: false,
			deletedStore:  123,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFn(tt, t)
		})
	}
}

func TestPDPodSyncForLeaderTransfer(t *testing.T) {
	const (
		interval = 100 * time.Millisecond
		timeout  = time.Minute
	)
	testCases := []struct {
		name                string
		replicas            int
		phase               v1alpha1.MemberPhase
		leader              int
		failed              []int
		target              int
		deleteAfterTransfer bool
		shouldTransfer      bool
		expirationTime      time.Time
	}{
		{
			name:                "transfer leader only",
			replicas:            3,
			phase:               v1alpha1.NormalPhase,
			leader:              0,
			failed:              nil,
			target:              0,
			deleteAfterTransfer: false,
			shouldTransfer:      true,
		},
		{
			name:                "transfer leader and delete pod",
			replicas:            3,
			phase:               v1alpha1.NormalPhase,
			leader:              0,
			failed:              nil,
			target:              0,
			deleteAfterTransfer: true,
			shouldTransfer:      true,
		},
		{
			name:                "delete pod only",
			replicas:            3,
			phase:               v1alpha1.NormalPhase,
			leader:              0,
			failed:              nil,
			target:              1,
			deleteAfterTransfer: true,
			shouldTransfer:      true,
		},
		{
			name:                "one pd is unhealthy",
			replicas:            3,
			phase:               v1alpha1.NormalPhase,
			leader:              0,
			failed:              []int{1},
			target:              0,
			deleteAfterTransfer: true,
			shouldTransfer:      false,
		},
		{
			name:                "transfer while upgrade",
			replicas:            3,
			phase:               v1alpha1.UpgradePhase,
			leader:              0,
			failed:              nil,
			target:              0,
			deleteAfterTransfer: true,
			shouldTransfer:      false,
		},
		{
			name:                "no expiration annotation",
			replicas:            3,
			phase:               v1alpha1.NormalPhase,
			leader:              0,
			failed:              nil,
			target:              0,
			deleteAfterTransfer: true,
			shouldTransfer:      true,
		},
		{
			name:                "expiration annotation in the future",
			replicas:            3,
			phase:               v1alpha1.NormalPhase,
			leader:              0,
			failed:              nil,
			target:              0,
			deleteAfterTransfer: true,
			shouldTransfer:      true,
			expirationTime:      metav1.Now().Time.Add(time.Hour),
		},
		{
			name:                "expiration annotation in the past",
			replicas:            3,
			phase:               v1alpha1.NormalPhase,
			leader:              0,
			failed:              nil,
			target:              0,
			deleteAfterTransfer: true,
			shouldTransfer:      false,
			expirationTime:      metav1.Now().Time.Add(-time.Minute),
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.TODO()
			g := NewGomegaWithT(t)
			tc := newTidbCluster()

			var mu sync.Mutex
			currentLeader := fmt.Sprintf("%s-%d", controller.PDMemberName(tc.Name), c.leader)
			pdClient := pdapi.NewFakePDClient()
			pdClient.AddReaction(pdapi.TransferPDLeaderActionType, func(action *pdapi.Action) (interface{}, error) {
				mu.Lock()
				defer mu.Unlock()
				currentLeader = action.Name
				return nil, nil
			})

			deps := controller.NewFakeDependencies()
			podController := NewPodController(deps)
			podController.testPDClient = pdClient

			stop := make(chan struct{})
			go func() {
				deps.KubeInformerFactory.Start(stop)
			}()
			deps.KubeInformerFactory.WaitForCacheSync(stop)
			go func() {
				deps.InformerFactory.Start(stop)
			}()
			deps.InformerFactory.WaitForCacheSync(stop)

			defer close(stop)
			go func() {
				podController.Run(1, stop)
			}()

			tc.Status.PD = v1alpha1.PDStatus{
				Synced: true,
				Phase:  c.phase,
				Leader: v1alpha1.PDMember{
					Name:   fmt.Sprintf("%s-%d", controller.PDMemberName(tc.Name), c.leader),
					Health: true,
				},
				Members: make(map[string]v1alpha1.PDMember),
			}
			healths := make([]pdapi.MemberHealth, c.replicas)
			for i := 0; i < c.replicas; i++ {
				member := fmt.Sprintf("%s-%d", controller.PDMemberName(tc.Name), i)
				tc.Status.PD.Members[member] = v1alpha1.PDMember{
					Name:   member,
					Health: true,
				}
				health := true
				for _, failed := range c.failed {
					if i == failed {
						health = false
					}
				}
				healths[i] = pdapi.MemberHealth{
					Name:   member,
					Health: health,
				}
			}
			pdClient.AddReaction(pdapi.GetHealthActionType, func(action *pdapi.Action) (interface{}, error) {
				return &pdapi.HealthInfo{Healths: healths}, nil
			})
			for _, i := range c.failed {
				member := fmt.Sprintf("%s-%d", controller.PDMemberName(tc.Name), i)
				tc.Status.PD.Members[member] = v1alpha1.PDMember{
					Name:   member,
					Health: false,
				}
			}
			tc, err := deps.Clientset.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(ctx, tc, metav1.CreateOptions{})
			g.Expect(err).NotTo(HaveOccurred())
			g.Eventually(func() error {
				_, err := deps.TiDBClusterLister.TidbClusters(tc.Namespace).Get(tc.Name)
				return err
			}, timeout, interval).Should(Succeed())

			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-%d", controller.PDMemberName(tc.Name), c.target),
					Namespace: tc.Namespace,
					Labels: map[string]string{
						label.ManagedByLabelKey: "tidb-operator",
						label.ComponentLabelKey: "pd",
						label.InstanceLabelKey:  tc.Name,
					},
					Annotations: make(map[string]string),
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "dummy-name",
						},
					},
				},
			}
			if c.deleteAfterTransfer {
				pod.Annotations[v1alpha1.PDLeaderTransferAnnKey] = v1alpha1.TransferLeaderValueDeletePod
			} else {
				pod.Annotations[v1alpha1.PDLeaderTransferAnnKey] = v1alpha1.TransferLeaderValueNone
			}
			if !c.expirationTime.IsZero() {
				pod.Annotations[v1alpha1.PDLeaderTransferExpirationTimeAnnKey] = c.expirationTime.Format(time.RFC3339)
			}

			pod, err = deps.KubeClientset.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
			g.Expect(err).NotTo(HaveOccurred())

			// verify end state
			if c.shouldTransfer {
				g.Eventually(func() bool {
					mu.Lock()
					defer mu.Unlock()

					// verify leader has been transferred
					if currentLeader == pod.Name {
						// leader has not been transferred
						return false
					}

					if c.deleteAfterTransfer {
						_, err := deps.KubeClientset.CoreV1().Pods(tc.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
						return errors.IsNotFound(err)
					}
					return true
				}, timeout, interval).Should(BeTrue(), "should delete pod")
			} else {
				g.Consistently(func() bool {
					// verify leader remain the same as initial state
					if currentLeader != fmt.Sprintf("%s-%d", controller.PDMemberName(tc.Name), c.leader) {
						return false
					}
					_, err := deps.KubeClientset.CoreV1().Pods(tc.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
					return err == nil
				}, timeout, interval).Should(BeTrue(), "should not transfer leader")
			}
		})
	}
}

func TestTiDBPodSyncForGracefulShutdown(t *testing.T) {
	const (
		interval = 100 * time.Millisecond
		timeout  = time.Minute
	)
	testCases := []struct {
		name         string
		pdPhase      v1alpha1.MemberPhase
		tikvPhase    v1alpha1.MemberPhase
		tidbPhase    v1alpha1.MemberPhase
		shouldDelete bool
		target       int
	}{
		{
			name:         "delete tidb pod successfully",
			pdPhase:      v1alpha1.NormalPhase,
			tikvPhase:    v1alpha1.NormalPhase,
			tidbPhase:    v1alpha1.NormalPhase,
			shouldDelete: true,
			target:       0,
		},
		{
			name:         "PD rolling restart",
			pdPhase:      v1alpha1.UpgradePhase,
			tikvPhase:    v1alpha1.NormalPhase,
			tidbPhase:    v1alpha1.NormalPhase,
			shouldDelete: false,
			target:       0,
		},
		{
			name:         "TiKV rolling restart",
			pdPhase:      v1alpha1.NormalPhase,
			tikvPhase:    v1alpha1.UpgradePhase,
			tidbPhase:    v1alpha1.NormalPhase,
			shouldDelete: false,
			target:       0,
		},
		{
			name:         "TiDB rolling restart",
			pdPhase:      v1alpha1.NormalPhase,
			tikvPhase:    v1alpha1.NormalPhase,
			tidbPhase:    v1alpha1.UpgradePhase,
			shouldDelete: false,
			target:       0,
		},
	}
	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.TODO()
			g := NewGomegaWithT(t)
			tc := newTidbCluster()
			deps := controller.NewFakeDependencies()
			podController := NewPodController(deps)

			stop := make(chan struct{})
			go func() {
				deps.KubeInformerFactory.Start(stop)
			}()
			deps.KubeInformerFactory.WaitForCacheSync(stop)
			go func() {
				deps.InformerFactory.Start(stop)
			}()
			deps.InformerFactory.WaitForCacheSync(stop)

			defer close(stop)
			go func() {
				podController.Run(1, stop)
			}()

			tc.Status.PD = v1alpha1.PDStatus{
				Synced: true,
				Phase:  c.pdPhase,
				Leader: v1alpha1.PDMember{
					Name:   fmt.Sprintf("%s-%d", controller.PDMemberName(tc.Name), 0),
					Health: true,
				},
				Members: make(map[string]v1alpha1.PDMember),
			}

			tc.Status.TiKV = v1alpha1.TiKVStatus{
				Synced: true,
				Phase:  c.tikvPhase,
				Stores: map[string]v1alpha1.TiKVStore{
					"0": {
						PodName: fmt.Sprintf("%s-%d", controller.TiKVMemberName(tc.Name), 0),
						ID:      "0",
					},
				},
			}

			tc.Status.TiDB = v1alpha1.TiDBStatus{
				Phase: c.tidbPhase,
			}

			tc, err := deps.Clientset.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(ctx, tc, metav1.CreateOptions{})
			g.Expect(err).NotTo(HaveOccurred())
			g.Eventually(func() error {
				_, err := deps.TiDBClusterLister.TidbClusters(tc.Namespace).Get(tc.Name)
				return err
			}, timeout, interval).Should(Succeed())

			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-%d", controller.TiDBMemberName(tc.Name), c.target),
					Namespace: tc.Namespace,
					Labels: map[string]string{
						label.ManagedByLabelKey: "tidb-operator",
						label.ComponentLabelKey: "tidb",
						label.InstanceLabelKey:  tc.Name,
					},
					Annotations: make(map[string]string),
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "dummy-name",
						},
					},
				},
			}

			pod.Annotations[v1alpha1.TiDBGracefulShutdownAnnKey] = v1alpha1.TiDBPodDeletionDeletePod

			pod, err = deps.KubeClientset.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
			g.Expect(err).NotTo(HaveOccurred())

			// verify end state
			if c.shouldDelete {
				g.Eventually(func() bool {
					_, err := deps.KubeClientset.CoreV1().Pods(tc.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
					return errors.IsNotFound(err)
				}, timeout, interval).Should(BeTrue(), "should delete pod")
			} else {
				g.Consistently(func() bool {
					_, err := deps.KubeClientset.CoreV1().Pods(tc.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
					return err == nil
				}, timeout, interval).Should(BeTrue(), "should not delete pod")
			}
		})
	}
}

func TestNeedEvictLeader(t *testing.T) {
	g := NewGomegaWithT(t)

	pod := &corev1.Pod{}
	pod.Annotations = map[string]string{}

	// none key
	_, _, exist := needEvictLeader(pod.DeepCopy())
	g.Expect(exist).To(BeFalse())

	// any key is exist
	for _, key := range v1alpha1.EvictLeaderAnnKeys {
		cur := pod.DeepCopy()
		cur.Annotations[key] = v1alpha1.EvictLeaderValueDeletePod
		usedkey, val, exist := needEvictLeader(cur)
		g.Expect(exist).To(BeTrue())
		g.Expect(key).To(Equal(usedkey))
		g.Expect(val).To(Equal(cur.Annotations[key]))
	}

}

func newTiKVPod(tc *v1alpha1.TidbCluster) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      controller.TiKVMemberName(tc.Name) + "-0",
			Namespace: tc.Namespace,
			Labels: map[string]string{
				label.ManagedByLabelKey: "tidb-operator",
				label.ComponentLabelKey: "tikv",
				label.InstanceLabelKey:  tc.Name,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "dummy-name",
				},
			},
		},
	}
}

func TestPdPodSyncForReplaceVolume(t *testing.T) {
	g := NewGomegaWithT(t)

	tc := newTidbCluster()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-0", controller.PDMemberName(tc.Name)),
			Namespace: tc.Namespace,
			Labels: map[string]string{
				label.MemberIDLabelKey: "123",
			},
			Annotations: map[string]string{
				v1alpha1.ReplaceVolumeAnnKey: v1alpha1.ReplaceVolumeValueTrue,
			},
		},
	}
	deps := controller.NewFakeDependencies()
	c := NewPodController(deps)
	pdClient := pdapi.NewFakePDClient()
	c.testPDClient = pdClient

	var leaderId uint64 = 123
	pdClient.AddReaction(pdapi.GetPDLeaderActionType, func(action *pdapi.Action) (interface{}, error) {
		return &pdpb.Member{
			MemberId: leaderId,
		}, nil
	})

	pdClient.AddReaction(pdapi.GetMembersActionType, func(action *pdapi.Action) (interface{}, error) {
		return &pdapi.MembersInfo{
			Members: []*pdpb.Member{
				{
					MemberId: 123,
					Name:     "pd-0",
				},
				{
					MemberId: 124,
					Name:     "pd-1",
				},
				{
					MemberId: 125,
					Name:     "pd-2",
				},
			},
		}, nil
	})

	transferName := ""
	pdClient.AddReaction(pdapi.TransferPDLeaderActionType, func(action *pdapi.Action) (interface{}, error) {
		transferName = action.Name
		return nil, nil
	})

	deletePVCsAndPodCalled := false
	c.deletePVCsAndPodFn = func(deps *controller.Dependencies, ctx context.Context, pod *corev1.Pod, tc *v1alpha1.TidbCluster) (reconcile.Result, error) {
		deletePVCsAndPodCalled = true
		return reconcile.Result{}, nil
	}

	ctx := context.Background()

	// Current pod is leader, check leader transferred
	leaderId = 123
	transferName = ""
	result, err := c.syncPDPodForReplaceVolume(ctx, pod, tc)
	g.Expect(err).Should(Succeed())
	g.Expect(result.Requeue).To(BeTrue())
	g.Expect(transferName).To(Equal("pd-1"))
	g.Expect(deletePVCsAndPodCalled).To(BeFalse())
	// Check deleted when not leader
	leaderId = 124
	transferName = ""
	result, err = c.syncPDPodForReplaceVolume(ctx, pod, tc)
	g.Expect(err).Should(Succeed())
	g.Expect(result.Requeue).To(BeFalse())
	g.Expect(transferName).To(BeEmpty())
	g.Expect(deletePVCsAndPodCalled).To(BeTrue())
	// Check not delete when not all PD's are ready.
	deletePVCsAndPodCalled = false
	tc.Status.PD.Members = map[string]v1alpha1.PDMember{
		"125": {
			Health: false,
		},
	}
	result, err = c.syncPDPodForReplaceVolume(ctx, pod, tc)
	g.Expect(err).To(HaveOccurred())
	g.Expect(result.Requeue).To(BeTrue())
	g.Expect(transferName).To(BeEmpty())
	g.Expect(deletePVCsAndPodCalled).To(BeFalse())
}
