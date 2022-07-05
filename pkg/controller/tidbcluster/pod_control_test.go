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
	"sync/atomic"
	"testing"
	"time"

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

func TestPodControllerSync(t *testing.T) {
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
	c.testPDClient = pdapi.NewFakePDClient()
	c.recheckLeaderCountDuration = time.Millisecond * 100

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
	pod.Annotations[v1alpha1.EvictLeaderAnnKey] = v1alpha1.EvictLeaderValueDeletePod
	pod, err = deps.KubeClientset.CoreV1().Pods(pod.Namespace).Update(ctx, pod, metav1.UpdateOptions{})
	g.Expect(err).Should(Succeed())

	g.Eventually(func() int {
		stat := c.getPodStat(pod)
		return stat.observeAnnotationCounts
	}, timeout, interval).ShouldNot(Equal(0), "should observe pod annotation")

	_, err = deps.KubeClientset.CoreV1().Pods(tc.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
	g.Expect(err).Should(Succeed())

	atomic.StoreInt32(&kvClient.leaderCount, 0)
	g.Eventually(func() bool {
		_, err := deps.KubeClientset.CoreV1().Pods(tc.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
		return errors.IsNotFound(err)
	}, timeout, interval).Should(BeTrue(), "should delete pod if leader count is 0")

	pod = newTiKVPod(tc)
	pod, err = deps.KubeClientset.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Eventually(func() int {
		stat := c.getPodStat(pod)
		return stat.finishAnnotationCounts
	}, timeout, interval).ShouldNot(Equal(0), "should finish annotation")
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
