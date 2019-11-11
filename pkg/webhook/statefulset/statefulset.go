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
	"strconv"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/webhook/util"
	"k8s.io/api/admission/v1beta1"
	apps "k8s.io/api/apps/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
)

var (
	versionCli   versioned.Interface
	deserializer runtime.Decoder
)

func init() {
	deserializer = util.GetCodec()
}

func AdmitStatefulSets(ar v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {

	name := ar.Request.Name
	namespace := ar.Request.Namespace
	glog.Infof("admit statefulsets [%s/%s]", name, namespace)

	apiVersion := ar.Request.Resource.Version
	setResource := metav1.GroupVersionResource{Group: "apps", Version: apiVersion, Resource: "statefulsets"}
	if ar.Request.Resource.Group != "apps" || ar.Request.Resource.Resource != "statefulsets" {
		err := fmt.Errorf("expect resource to be %s instead of %s", setResource, ar.Request.Resource)
		glog.Error(err)
		return util.ARFail(err)
	}

	if versionCli == nil {
		cfg, err := rest.InClusterConfig()
		if err != nil {
			glog.Errorf("failed to get config: %v", err)
			return util.ARFail(err)
		}

		versionCli, err = versioned.NewForConfig(cfg)
		if err != nil {
			glog.Errorf("failed to create Clientset: %v", err)
			return util.ARFail(err)
		}
	}

	stsObjectMeta, stsPartition, err := getStsAttributes(ar.Request.OldObject.Raw, apiVersion)
	if err != nil {
		err = fmt.Errorf("statefulset %s/%s, decode request failed, err: %v", namespace, name, err)
		glog.Error(err)
		return util.ARFail(err)
	}

	tc, err := versionCli.PingcapV1alpha1().TidbClusters(namespace).Get(stsObjectMeta.Labels[label.InstanceLabelKey], metav1.GetOptions{})
	if err != nil {
		glog.Errorf("fail to fetch tidbcluster info namespace %s clustername(instance) %s err %v", namespace, stsObjectMeta.Labels[label.InstanceLabelKey], err)
		return util.ARFail(err)
	}

	if stsObjectMeta.Labels[label.ComponentLabelKey] == label.TiDBLabelVal {
		protect, ok := tc.Annotations[label.AnnTiDBPartition]

		if ok {
			partition, err := strconv.ParseInt(protect, 10, 32)
			if err != nil {
				glog.Errorf("fail to convert protect to int namespace %s name %s err %v", namespace, name, err)
				return util.ARFail(err)
			}

			if *stsPartition <= int32(partition) && tc.Status.TiDB.Phase == v1alpha1.UpgradePhase {
				glog.Infof("set has been protect by annotations name %s namespace %s", name, namespace)
				return util.ARFail(errors.New("protect by annotation"))
			}
		}
	}

	return util.ARSuccess()
}

func getStsAttributes(data []byte, apiVersion string) (*metav1.ObjectMeta, *int32, error) {
	if apiVersion == "v1" {
		set := apps.StatefulSet{}
		if _, _, err := deserializer.Decode(data, nil, &set); err != nil {
			return nil, nil, err
		}
		return &(set.ObjectMeta), set.Spec.UpdateStrategy.RollingUpdate.Partition, nil
	}

	set := appsv1beta1.StatefulSet{}
	if _, _, err := deserializer.Decode(data, nil, &set); err != nil {
		return nil, nil, err
	}
	return &(set.ObjectMeta), set.Spec.UpdateStrategy.RollingUpdate.Partition, nil
}
