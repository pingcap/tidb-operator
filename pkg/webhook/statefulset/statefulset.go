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

package statefulset

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/openshift/generic-admission-server/pkg/apiserver"
	asapps "github.com/pingcap/advanced-statefulset/client/apis/apps/v1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/webhook/util"
	admission "k8s.io/api/admission/v1beta1"
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

var (
	deserializer runtime.Decoder = util.Codecs.UniversalDeserializer()
)

type StatefulSetAdmissionControl struct {
	lock        sync.RWMutex
	initialized bool
	// operator client interface
	operatorCli versioned.Interface
}

var _ apiserver.ValidatingAdmissionHook = &StatefulSetAdmissionControl{}

func NewStatefulSetAdmissionControl() *StatefulSetAdmissionControl {
	return &StatefulSetAdmissionControl{}
}

func (sc *StatefulSetAdmissionControl) ValidatingResource() (plural schema.GroupVersionResource, singular string) {
	return schema.GroupVersionResource{
			Group:    "admission.tidb.pingcap.com",
			Version:  "v1alpha1",
			Resource: "statefulsetvalidations",
		},
		"statefulsetvalidation"
}

func (sc *StatefulSetAdmissionControl) Validate(ar *admission.AdmissionRequest) *admission.AdmissionResponse {
	sc.lock.RLock()
	defer sc.lock.RUnlock()
	if !sc.initialized {
		return &admission.AdmissionResponse{
			Allowed: false,
		}
	}

	name := ar.Name
	namespace := ar.Namespace
	expectedGroup := "apps"
	if features.DefaultFeatureGate.Enabled(features.AdvancedStatefulSet) {
		expectedGroup = asapps.GroupName
	}
	apiVersion := ar.Resource.Version
	setResource := metav1.GroupVersionResource{Group: expectedGroup, Version: apiVersion, Resource: "statefulsets"}

	klog.Infof("admit %s [%s/%s]", setResource, namespace, name)

	stsObjectMeta, stsPartition, err := getStsAttributes(ar.Object.Raw)
	if err != nil {
		err = fmt.Errorf("statefulset %s/%s, decode request failed, err: %v", namespace, name, err)
		klog.Error(err)
		return util.ARFail(err)
	}

	l := label.Label(stsObjectMeta.Labels)

	if !(l.IsTiDB() || l.IsTiKV()) {
		// If it is not statefulset of tikv and tidb, return quickly.
		return util.ARSuccess()
	}

	controllerRef := metav1.GetControllerOf(stsObjectMeta)
	if controllerRef == nil || controllerRef.Kind != controller.ControllerKind.Kind {
		// In this case, we can't tell if this statefulset is controlled by tidb-operator,
		// so we don't block this statefulset upgrade, return directly.
		klog.Warningf("statefulset %s/%s has tidb or tikv component label but doesn't have owner reference or the owner reference is not TidbCluster", namespace, name)
		return util.ARSuccess()
	}

	tcName := controllerRef.Name
	tc, err := sc.operatorCli.PingcapV1alpha1().TidbClusters(namespace).Get(tcName, metav1.GetOptions{})
	if err != nil {
		err := fmt.Errorf("get tidbcluster %s/%s failed, statefulset %s, err %v", namespace, tcName, name, err)
		klog.Errorf(err.Error())
		return util.ARFail(err)
	}

	annKey := label.AnnTiDBPartition
	if l.IsTiKV() {
		annKey = label.AnnTiKVPartition
	}
	partitionStr := tc.Annotations[annKey]

	if len(partitionStr) == 0 {
		return util.ARSuccess()
	}

	partition, err := strconv.ParseInt(partitionStr, 10, 32)
	if err != nil {
		err := fmt.Errorf("statefulset %s/%s, convert partition str %s to int failed, err: %v", namespace, name, partitionStr, err)
		klog.Errorf(err.Error())
		return util.ARFail(err)
	}

	if stsPartition != nil {
		// only [partition from tc, INT32_MAX] are allowed
		if *stsPartition < int32(partition) {
			klog.Infof("statefulset %s/%s has been protected by partition annotation %q", namespace, name, partitionStr)
			return util.ARFail(fmt.Errorf("protected by partition annotation (%s = %s) on the tidb cluster %s/%s", annKey, partitionStr, namespace, tcName))
		}
		klog.Infof("admit statefulset %s/%s update partition to %d, protect partition is %d", namespace, name, *stsPartition, partition)
	}
	return util.ARSuccess()
}

// Initialize implements AdmissionHook.Initialize interface. It's is called as
// a post-start hook.
func (a *StatefulSetAdmissionControl) Initialize(cfg *rest.Config, stopCh <-chan struct{}) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	cli, err := versioned.NewForConfig(cfg)
	if err != nil {
		return err
	}

	a.operatorCli = cli

	a.initialized = true
	return nil
}

func getStsAttributes(data []byte) (*metav1.ObjectMeta, *int32, error) {
	if !features.DefaultFeatureGate.Enabled(features.AdvancedStatefulSet) {
		set := apps.StatefulSet{}
		if _, _, err := deserializer.Decode(data, nil, &set); err != nil {
			return nil, nil, err
		}
		if set.Spec.UpdateStrategy.RollingUpdate != nil {
			return &(set.ObjectMeta), set.Spec.UpdateStrategy.RollingUpdate.Partition, nil
		}
		return &(set.ObjectMeta), nil, nil
	}
	set := asapps.StatefulSet{}
	if _, _, err := deserializer.Decode(data, nil, &set); err != nil {
		return nil, nil, err
	}
	if set.Spec.UpdateStrategy.RollingUpdate != nil {
		return &(set.ObjectMeta), set.Spec.UpdateStrategy.RollingUpdate.Partition, nil
	}
	return &(set.ObjectMeta), nil, nil
}
