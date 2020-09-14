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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
)

type PodRestarter interface {
	Sync(meta metav1.Object) error
}

type podRestarter struct {
	kubeCli   kubernetes.Interface
	podLister corelisters.PodLister
}

func NewPodRestarter(kubeCli kubernetes.Interface, podLister corelisters.PodLister) *podRestarter {
	return &podRestarter{kubeCli: kubeCli, podLister: podLister}
}

func (gr *podRestarter) Sync(meta metav1.Object) error {
	namespace := meta.GetNamespace()
	var (
		selector labels.Selector
		err      error
		metaType string
	)
	switch meta.(type) {
	case *v1alpha1.TidbCluster:
		selector, err = label.New().Instance(meta.GetName()).Selector()
		metaType = "tc"
	case *v1alpha1.DMCluster:
		selector, err = label.NewDM().Instance(meta.GetName()).Selector()
		metaType = "dc"
	default:
		err = fmt.Errorf("podRestarter.Sync: unknown meta spec %s", meta)
	}
	if err != nil {
		return err
	}
	metaPods, err := gr.podLister.Pods(namespace).List(selector)
	if err != nil {
		return fmt.Errorf("podRestarter.Sync: failed to get pods list for cluster %s/%s, selector %s, error: %s", namespace, meta.GetName(), selector, err)
	}
	requeue := false
	for _, pod := range metaPods {
		if _, existed := pod.Annotations[label.AnnPodDeferDeleting]; existed {
			requeue = true
			err = gr.restart(pod)
			if err != nil {
				return err
			}
		}
	}
	if requeue {
		return controller.RequeueErrorf("%s[%s/%s] is under restarting", metaType, namespace, meta.GetName())
	}
	return nil
}

// pod deleting webhook ensured each tc pod would be deleted safely
func (gr *podRestarter) restart(pod *corev1.Pod) error {
	preconditions := metav1.Preconditions{UID: &pod.UID}
	deleteOptions := metav1.DeleteOptions{Preconditions: &preconditions}
	err := gr.kubeCli.CoreV1().Pods(pod.Namespace).Delete(pod.Name, &deleteOptions)
	if err != nil {
		return err
	}
	return nil
}

type FakeRestarter struct {
}

func NewFakePodRestarter() *FakeRestarter {
	return &FakeRestarter{}
}

func (fsr *FakeRestarter) Sync(meta metav1.Object) error {
	return nil
}
