// Copyright 2019 PingCAP, Inc.
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

package member

import (
	"fmt"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	operatorUtil "github.com/pingcap/tidb-operator/pkg/util"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	appslisters "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"strconv"
)

const (
	emptyRestartPodList = "empty restart pod List"
)

type Restarter interface {
	Sync(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType) error
}

type GeneralRestarter struct {
	kubeCli   kubernetes.Interface
	podLister corelisters.PodLister
	stsLister appslisters.StatefulSetLister
}

func NewGeneralRestarter(kubeCli kubernetes.Interface, podLister corelisters.PodLister, stsLister appslisters.StatefulSetLister) *GeneralRestarter {
	return &GeneralRestarter{kubeCli: kubeCli, podLister: podLister, stsLister: stsLister}
}

func (gr *GeneralRestarter) Sync(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType) error {
	sts := &apps.StatefulSet{}
	err := gr.syncRestartStatus(tc, memberType, sts)
	if err != nil {
		return err
	}
	return gr.sync(tc, memberType, sts)
}

// syncRestartStatus would check the statefulset of tc for each component whether they were restarting.
// If they are, it would check whether the restarting is finished and update the sts annotations.
func (gr *GeneralRestarter) syncRestartStatus(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType, sts *apps.StatefulSet) error {
	namespace := tc.Namespace
	stsName := operatorUtil.GetStatefulSetName(tc, memberType)
	sts, err := gr.stsLister.StatefulSets(namespace).Get(stsName)
	if err != nil {
		return err
	}
	index, existed := sts.Annotations[label.AnnPodRestarting]
	if !existed {
		return nil
	}

	ordinal, err := strconv.ParseInt(index, 10, 64)
	if err != nil {
		return err
	}
	podName := operatorUtil.GetPodName(tc, memberType, int32(ordinal))
	pod, err := gr.podLister.Pods(namespace).Get(podName)
	if err != nil {
		return err
	}
	if _, existed = pod.Annotations[label.AnnPodDeferDeleting]; existed {
		return nil
	}
	delete(sts.Annotations, label.AnnPodRestarting)
	_, err = gr.kubeCli.AppsV1().StatefulSets(namespace).Update(sts)
	if err != nil {
		return err
	}
	return controller.RequeueErrorf("tc[%s/%s]'s pod[%s/%s] is restarted,requeue", namespace, tc.Name, namespace, podName)
}

// sync func would continue to try deleting the pod which is during restarting or pop one new pod with AnnPodDeferDeleting
// to restart
func (gr *GeneralRestarter) sync(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType, sts *apps.StatefulSet) error {
	podName, existed := sts.Annotations[label.AnnPodRestarting]
	if existed {
		return gr.restart(tc, memberType, podName)
	}
	pod, err := gr.pop(tc, memberType, sts)
	if err != nil {
		if err.Error() == emptyRestartPodList {
			return nil
		}
	}
	sts.Annotations[label.AnnPodRestarting] = pod.Name
	_, err = gr.kubeCli.AppsV1().StatefulSets(tc.Namespace).Update(sts)
	if err != nil {
		return err
	}
	return gr.restart(tc, memberType, pod.Name)

}

// pod deleting webhook ensured each tc pod would be deleted safely
func (gr *GeneralRestarter) restart(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType, podName string) error {
	err := gr.kubeCli.CoreV1().Pods(tc.Namespace).Delete(podName, &meta.DeleteOptions{})
	if err != nil {
		return err
	}
	return controller.RequeueErrorf("tc[%s/%s]'s pod[%s/%s] is restarting now", tc.Namespace, tc.Name, tc.Namespace, podName)
}

func (gr *GeneralRestarter) pop(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType, sts *apps.StatefulSet) (*core.Pod, error) {
	labelSelector := &meta.LabelSelector{
		MatchLabels: map[string]string{
			label.ComponentLabelKey: memberType.String(),
			label.ManagedByLabelKey: "tidb-operator",
			label.NameLabelKey:      "tidb-cluster",
			label.InstanceLabelKey:  tc.Name,
		},
	}
	selector, err := meta.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		return nil, err
	}
	pods, err := gr.podLister.List(selector)
	if err != nil {
		return nil, err
	}
	for _, pod := range pods {
		if _, existed := pod.Annotations[label.AnnPodDeferDeleting]; existed {
			return pod, nil
		}
	}
	return nil, fmt.Errorf(emptyRestartPodList)
}

type FakeRestarter struct {
}

func NewFakeRestarter() *FakeRestarter {
	return &FakeRestarter{}
}

func (fsr *FakeRestarter) Sync(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType) error {
	return nil
}
