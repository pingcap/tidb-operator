// Copyright 2018 PingCAP, Inc.
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

package controller

import (
	"fmt"
	"strings"

	"github.com/pingcap/tidb-operator/new-operator/pkg/apis/pingcap.com/v1"
	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	appslisters "k8s.io/client-go/listers/apps/v1beta1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
)

// StatefulSetControlInterface defines the interface that uses to create, update, and delete StatefulSets,
type StatefulSetControlInterface interface {
	// CreateStatefulSet creates a StatefulSet in a TidbCluster.
	CreateStatefulSet(*v1.TidbCluster, *apps.StatefulSet) error
	// UpdateStatefulSet updates a StatefulSet in a TidbCluster.
	UpdateStatefulSet(*v1.TidbCluster, *apps.StatefulSet) error
	// DeleteStatefulSet deletes a StatefulSet in a TidbCluster.
	DeleteStatefulSet(*v1.TidbCluster, *apps.StatefulSet) error
}

type realStatefulSetControl struct {
	kubeCli   kubernetes.Interface
	setLister appslisters.StatefulSetLister
	recorder  record.EventRecorder
}

// NewRealStatefuSetControl returns a StatefulSetControlInterface
func NewRealStatefuSetControl(kubeCli kubernetes.Interface, setLister appslisters.StatefulSetLister, recorder record.EventRecorder) StatefulSetControlInterface {
	return &realStatefulSetControl{kubeCli, setLister, recorder}
}

// CreateStatefulSet create a StatefulSet in a TidbCluster.
func (sc *realStatefulSetControl) CreateStatefulSet(tc *v1.TidbCluster, set *apps.StatefulSet) error {
	_, err := sc.kubeCli.AppsV1beta1().StatefulSets(tc.Namespace).Create(set)
	// sink already exists errors
	if apierrors.IsAlreadyExists(err) {
		return err
	}
	sc.recordStatefulSetEvent("create", tc, set, err)
	return err
}

// UpdateStatefulSet update a StatefulSet in a TidbCluster.
func (sc *realStatefulSetControl) UpdateStatefulSet(tc *v1.TidbCluster, set *apps.StatefulSet) error {
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		// TODO: verify if StatefulSet identity(name, namespace, labels) matches TidbCluster
		_, updateErr := sc.kubeCli.AppsV1beta1().StatefulSets(tc.Namespace).Update(set)
		if updateErr == nil {
			return nil
		}
		if updated, err := sc.setLister.StatefulSets(tc.Namespace).Get(set.Name); err != nil {
			// make a copy so we don't mutate the shared cache
			set = updated.DeepCopy()
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated StatefulSet %s/%s from lister: %v", tc.Namespace, set.Name, err))
		}
		return updateErr
	})
	sc.recordStatefulSetEvent("update", tc, set, err)
	return err
}

// DeleteStatefulSet delete a StatefulSet in a TidbCluster.
func (sc *realStatefulSetControl) DeleteStatefulSet(tc *v1.TidbCluster, set *apps.StatefulSet) error {
	err := sc.kubeCli.AppsV1beta1().StatefulSets(tc.Namespace).Delete(set.Name, nil)
	sc.recordStatefulSetEvent("delete", tc, set, err)
	return err
}

func (sc *realStatefulSetControl) recordStatefulSetEvent(verb string, tc *v1.TidbCluster, set *apps.StatefulSet, err error) {
	tcName := tc.Name
	setName := set.Name
	if err == nil {
		reason := fmt.Sprintf("Successful%s", strings.Title(verb))
		message := fmt.Sprintf("%s StatefulSet %s in TidbCluster %s successful",
			strings.ToLower(verb), setName, tcName)
		sc.recorder.Event(tc, corev1.EventTypeNormal, reason, message)
	} else {
		reason := fmt.Sprintf("Failed%s", strings.Title(verb))
		message := fmt.Sprintf("%s StatefulSet %s in TidbCluster %s failed error: %s",
			strings.ToLower(verb), setName, tcName, err)
		sc.recorder.Event(tc, corev1.EventTypeWarning, reason, message)
	}
}

var _ StatefulSetControlInterface = &realStatefulSetControl{}
