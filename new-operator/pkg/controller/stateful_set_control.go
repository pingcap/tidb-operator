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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/listers/apps/v1beta1"
	"k8s.io/client-go/tools/record"
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

type defaultStatefulSetControl struct {
	kubeCli   kubernetes.Interface
	setLister v1beta1.StatefulSetLister
	recorder  record.EventRecorder
}

// NewRealStatefuSetControl returns a defaultStatefulSetControl
func NewRealStatefuSetControl(kubeCli kubernetes.Interface,
	setLister v1beta1.StatefulSetLister,
	recorder record.EventRecorder) *defaultStatefulSetControl {
	return &defaultStatefulSetControl{
		kubeCli,
		setLister,
		recorder,
	}
}

// CreateStatefulSet create a StatefulSet in a TidbCluster.
func (dss *defaultStatefulSetControl) CreateStatefulSet(tc *v1.TidbCluster, set *apps.StatefulSet) error {
	set, err := dss.kubeCli.AppsV1beta1().StatefulSets(tc.Namespace).Create(set)
	dss.recordStatefulSetEvent("create", tc, set, err)
	return err
}

// UpdateStatefulSet update a StatefulSet in a TidbCluster.
func (dss *defaultStatefulSetControl) UpdateStatefulSet(tc *v1.TidbCluster, set *apps.StatefulSet) error {
	return nil
}

// DeleteStatefulSet delete a StatefulSet in a TidbCluster.
func (dss *defaultStatefulSetControl) DeleteStatefulSet(tc *v1.TidbCluster, set *apps.StatefulSet) error {
	return nil
}

func (dss *defaultStatefulSetControl) recordStatefulSetEvent(verb string, tc *v1.TidbCluster, set *apps.StatefulSet, err error) {
	tcName := tc.Name
	setName := set.Name
	if err == nil {
		reason := fmt.Sprintf("Successful%s", strings.Title(verb))
		message := fmt.Sprintf("%s StatefulSet %s in TidbCluster %s successful",
			strings.ToLower(verb), setName, tcName)
		dss.recorder.Event(tc, corev1.EventTypeNormal, reason, message)
	} else {
		reason := fmt.Sprintf("Failed%s", strings.Title(verb))
		message := fmt.Sprintf("%s StatefulSet %s in TidbCluster %s failed error: %s",
			strings.ToLower(verb), setName, tcName, err)
		dss.recorder.Event(tc, corev1.EventTypeWarning, reason, message)
	}
}
