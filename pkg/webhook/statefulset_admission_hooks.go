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

package webhook

import (
	"sync"

	asappsv1alpha1 "github.com/pingcap/advanced-statefulset/pkg/apis/apps/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/webhook/statefulset"
	"github.com/pingcap/tidb-operator/pkg/webhook/util"
	admission "k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

type StatefulSetAdmissionHook struct {
	lock        sync.RWMutex
	initialized bool
	stsAC       *statefulset.StatefulSetAdmissionControl
}

func (a *StatefulSetAdmissionHook) ValidatingResource() (plural schema.GroupVersionResource, singular string) {
	return schema.GroupVersionResource{
			Group:    "admission.tidb.pingcap.com",
			Version:  "v1alpha1",
			Resource: "statefulsetadmissionreviews",
		},
		"StatefulSetAdmissionReview"
}

func (a *StatefulSetAdmissionHook) Validate(ar *admission.AdmissionRequest) *admission.AdmissionResponse {
	if !a.initialized {
		return &admission.AdmissionResponse{
			Allowed: false,
		}
	}
	expectedGroup := "apps"
	if features.DefaultFeatureGate.Enabled(features.AdvancedStatefulSet) {
		expectedGroup = asappsv1alpha1.GroupName
	}
	if "StatefulSet" != ar.Kind.Kind || expectedGroup != ar.Kind.Kind {
		klog.Infof("success to %v %s[%s/%s]", ar.Operation, ar.Kind.Kind, ar.Name, ar.Namespace)
		return util.ARSuccess()
	}
	return a.stsAC.AdmitStatefulSets(ar)
}

func (a *StatefulSetAdmissionHook) Initialize(cfg *rest.Config, stopCh <-chan struct{}) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	cli, err := versioned.NewForConfig(cfg)
	if err != nil {
		return err
	}
	a.stsAC = statefulset.NewStatefulSetAdmissionControl(cli)
	a.initialized = true

	klog.Info("statefulset admission webhook initialized successfully")
	return nil
}
