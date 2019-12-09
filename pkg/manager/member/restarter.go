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
	"github.com/pingcap/tidb-operator/pkg/label"
	operatorUtils "github.com/pingcap/tidb-operator/pkg/util"
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

// 0. 只有 webhook 启用时才支持 RestartManager
// 1. pod 打上 annotation
// 2. RestartManager 检查 tcStatus 有没有正在重启的pod,如果有，则发送 delete 请求后，Requeue
// 3. 如果没有，List 所有需要重启的 pod
// 4. 按照规则 pop 一个 pod 出来重启，给对应 sts 打上 anno，标识某个 pod处于重启状态中
type Restarter interface {
	Sync(tc *v1alpha1.TidbCluster) error
	//Pop(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType) (*core.Pod, error)
	//Restart(tc *v1alpha1.TidbCluster, pod *core.Pod) error
}

type GeneralRestarter struct {
	kubeCli   kubernetes.Interface
	podLister corelisters.PodLister
	stsLister appslisters.StatefulSetLister
}

func NewGeneralRestarter(kubeCli kubernetes.Interface, podLister corelisters.PodLister, stsLister appslisters.StatefulSetLister) *GeneralRestarter {
	return &GeneralRestarter{kubeCli: kubeCli, podLister: podLister, stsLister: stsLister}
}

func (gr *GeneralRestarter) Sync(tc *v1alpha1.TidbCluster) error {

	pod, err := gr.sync(tc, v1alpha1.PDMemberType)
	if err != nil {
		if err.Error() == emptyRestartPodList {

		}
	}
	return nil
}

func (gr *GeneralRestarter) restart(tc *v1alpha1.TidbCluster, pod *core.Pod) error {
	return gr.kubeCli.CoreV1().Pods(tc.Namespace).Delete(pod.Name, &meta.DeleteOptions{})
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

func (gr *GeneralRestarter) pop(podList []*core.Pod) (*core.Pod, error) {
	if len(podList) < 1 {
		return nil, fmt.Errorf(emptyRestartPodList)
	}
	if len(podList) == 1 {
		return podList[0], nil
	}
	maxOrdinal := -1
	var returnerd *core.Pod
	for _, pod := range podList {
		ordinal, err := operatorUtils.GetOrdinalFromPodName(pod.Name)
		if err != nil {
			return nil, err
		}
		if int(ordinal) > maxOrdinal {
			maxOrdinal = int(ordinal)
			returnerd = pod
		}
	}
	return returnerd, nil
}

func (gr *GeneralRestarter) isInRestarting(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType) (bool, error) {
	sts, err := gr.stsLister.StatefulSets(tc.Namespace).Get(operatorUtils.GetStatefulSetName(tc, memberType))
	if err != nil {
		return false, err
	}

}

func (gr *GeneralRestarter) sync(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType) (*core.Pod, error) {
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
	pods, err := gr.list(selector)
	if err != nil {
		return nil, err
	}
	return gr.pop(pods)
}
