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
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	appslisters "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
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
	return gr.sync(tc, memberType)
}

func (gr *GeneralRestarter) sync(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType) error {
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
		return err
	}
	pods, err := gr.list(selector)
	if err != nil {
		if err.Error() == emptyRestartPodList {
			return nil
		}
	}
	return gr.restart(tc, pods[0])
}

func (gr *GeneralRestarter) restart(tc *v1alpha1.TidbCluster, pod *core.Pod) error {
	err := gr.kubeCli.CoreV1().Pods(tc.Namespace).Delete(pod.Name, &meta.DeleteOptions{})
	if err != nil {
		return err
	}
	return controller.RequeueErrorf("pod[%s/%s] is going to restart now", pod.Namespace, pod.Name)
}

func (gr *GeneralRestarter) list(selector labels.Selector) ([]*core.Pod, error) {
	pods, err := gr.podLister.List(selector)
	if err != nil {
		return nil, err
	}
	var restartMarkedPods []*core.Pod
	for _, pod := range pods {
		if _, existed := pod.Annotations[label.AnnPodDeferDeleting]; existed {
			restartMarkedPods = append(restartMarkedPods, pod)
		}
	}
	if len(restartMarkedPods) < 1 {
		return nil, fmt.Errorf(emptyRestartPodList)
	}
	return restartMarkedPods, nil
}

type FakeRestarter struct {
}

func NewFakeRestarter() *FakeRestarter {
	return &FakeRestarter{}
}

func (fsr *FakeRestarter) Sync(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType) error {
	return nil
}
