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

package tests

import (
	"fmt"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/tests/slack"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

type CRDOperatorActions interface {
	GetTidbClusterOrDie(name, namespace string) *v1alpha1.TidbCluster
	UpdateTidbClusterOrDie(tc *v1alpha1.TidbCluster)
	CreateTidbClusterOrDie(tc *v1alpha1.TidbCluster)
	CheckDisasterToleranceOrDie(tc *v1alpha1.TidbCluster)
}

type crdOperatorActions struct {
	kubeCli kubernetes.Interface
	cli     versioned.Interface
}

func NewCRDOperatorActions(cli versioned.Interface, kubeCli kubernetes.Interface) CRDOperatorActions {
	return &crdOperatorActions{
		cli:     cli,
		kubeCli: kubeCli,
	}
}

func (coa *crdOperatorActions) GetTidbClusterOrDie(name, namespace string) *v1alpha1.TidbCluster {
	tc, err := coa.cli.PingcapV1alpha1().TidbClusters(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		slack.NotifyAndPanic(err)
	}
	return tc
}

func (coa *crdOperatorActions) CreateTidbClusterOrDie(tc *v1alpha1.TidbCluster) {
	_, err := coa.cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(tc)
	if err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (coa *crdOperatorActions) UpdateTidbClusterOrDie(tc *v1alpha1.TidbCluster) {
	err := wait.Poll(5*time.Second, 3*time.Minute, func() (done bool, err error) {
		_, err = coa.cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Update(tc)
		if err != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (coa *crdOperatorActions) CheckDisasterToleranceOrDie(tc *v1alpha1.TidbCluster) {
	err := coa.checkDisasterTolerance(tc)
	if err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (coa *crdOperatorActions) checkDisasterTolerance(cluster *v1alpha1.TidbCluster) error {
	pds, err := coa.kubeCli.CoreV1().Pods(cluster.Namespace).List(
		metav1.ListOptions{LabelSelector: labels.SelectorFromSet(
			label.New().Instance(cluster.Name).PD().Labels(),
		).String()})
	if err != nil {
		return err
	}
	err = coa.checkPodsDisasterTolerance(pds.Items)
	if err != nil {
		return err
	}

	tikvs, err := coa.kubeCli.CoreV1().Pods(cluster.Namespace).List(
		metav1.ListOptions{LabelSelector: labels.SelectorFromSet(
			label.New().Instance(cluster.Name).TiKV().Labels(),
		).String()})
	if err != nil {
		return err
	}
	err = coa.checkPodsDisasterTolerance(tikvs.Items)
	if err != nil {
		return err
	}

	tidbs, err := coa.kubeCli.CoreV1().Pods(cluster.Namespace).List(
		metav1.ListOptions{LabelSelector: labels.SelectorFromSet(
			label.New().Instance(cluster.Name).TiDB().Labels(),
		).String()})
	if err != nil {
		return err
	}
	return coa.checkPodsDisasterTolerance(tidbs.Items)
}

func (coa *crdOperatorActions) checkPodsDisasterTolerance(allPods []corev1.Pod) error {
	for _, pod := range allPods {
		if pod.Spec.Affinity == nil {
			return fmt.Errorf("the pod:[%s/%s] has not Affinity", pod.Namespace, pod.Name)
		}
		if pod.Spec.Affinity.PodAntiAffinity == nil {
			return fmt.Errorf("the pod:[%s/%s] has not Affinity.PodAntiAffinity", pod.Namespace, pod.Name)
		}
		if len(pod.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution) == 0 {
			return fmt.Errorf("the pod:[%s/%s] has not PreferredDuringSchedulingIgnoredDuringExecution", pod.Namespace, pod.Name)
		}
		for _, prefer := range pod.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
			if prefer.PodAffinityTerm.TopologyKey != RackLabel {
				return fmt.Errorf("the pod:[%s/%s] topology key is not %s", pod.Namespace, pod.Name, RackLabel)
			}
		}
	}
	return nil
}
