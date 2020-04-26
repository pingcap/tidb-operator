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
	"errors"
	"fmt"
	"k8s.io/klog"
	"strconv"

	asappsv1 "github.com/pingcap/advanced-statefulset/pkg/apis/apps/v1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/webhook/util"
	admission "k8s.io/api/admission/v1beta1"
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	deserializer runtime.Decoder
)

func init() {
	deserializer = util.GetCodec()
}

type StatefulSetAdmissionControl struct {
	// operator client interface
	operatorCli versioned.Interface
}

func NewStatefulSetAdmissionControl(operatorCli versioned.Interface) *StatefulSetAdmissionControl {
	return &StatefulSetAdmissionControl{
		operatorCli: operatorCli,
	}
}

func (sc *StatefulSetAdmissionControl) AdmitStatefulSets(ar *admission.AdmissionRequest) *admission.AdmissionResponse {

	name := ar.Name
	namespace := ar.Namespace
	expectedGroup := "apps"
	if features.DefaultFeatureGate.Enabled(features.AdvancedStatefulSet) {
		expectedGroup = asappsv1.GroupName
	}
	apiVersion := ar.Resource.Version
	setResource := metav1.GroupVersionResource{Group: expectedGroup, Version: apiVersion, Resource: "statefulsets"}

	klog.Infof("admit %s [%s/%s]", setResource, namespace, name)

	stsObjectMeta, stsPartition, err := getStsAttributes(ar.OldObject.Raw)
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

	var partitionStr string
	partitionStr = tc.Annotations[label.AnnTiDBPartition]
	if l.IsTiKV() {
		partitionStr = tc.Annotations[label.AnnTiKVPartition]
	}

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
		if *stsPartition > 0 && *stsPartition <= int32(partition) {
			klog.Infof("statefulset %s/%s has been protect by partition %s annotations", namespace, name, partitionStr)
			return util.ARFail(errors.New("protect by partition annotation"))
		}
		klog.Infof("admit statefulset %s/%s update partition to %d, protect partition is %d", namespace, name, *stsPartition, partition)
	}
	return util.ARSuccess()
}

func getStsAttributes(data []byte) (*metav1.ObjectMeta, *int32, error) {
	set := apps.StatefulSet{}
	if _, _, err := deserializer.Decode(data, nil, &set); err != nil {
		return nil, nil, err
	}
	if set.Spec.UpdateStrategy.RollingUpdate != nil {
		return &(set.ObjectMeta), set.Spec.UpdateStrategy.RollingUpdate.Partition, nil
	}
	return &(set.ObjectMeta), nil, nil
}
