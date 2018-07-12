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
	"math"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/new-operator/pkg/apis/pingcap.com/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// controllerKind contains the schema.GroupVersionKind for this controller type.
var controllerKind = v1.SchemeGroupVersion.WithKind("TidbCluster")

const (
	defaultPushgatewayImage = "prom/pushgateway:v0.3.1"
)

// GetOwnerRef returns TidbCluster's OwnerReference
func GetOwnerRef(tc *v1.TidbCluster) metav1.OwnerReference {
	controller := true
	blockOwnerDeletion := true
	return metav1.OwnerReference{
		APIVersion:         controllerKind.GroupVersion().String(),
		Kind:               controllerKind.Kind,
		Name:               tc.GetName(),
		UID:                tc.GetUID(),
		Controller:         &controller,
		BlockOwnerDeletion: &blockOwnerDeletion,
	}
}

// TiKVCapacity returns string resource requirement,
// tikv uses GB, TB as unit suffix, but it actually means GiB, TiB
func TiKVCapacity(limits *v1.ResourceRequirement) string {
	defaultArgs := "0"
	if limits == nil {
		return defaultArgs
	}
	q, err := resource.ParseQuantity(limits.Storage)
	if err != nil {
		glog.Errorf("failed to parse quantity %s: %v", limits.Storage, err)
		return defaultArgs
	}
	i, b := q.AsInt64()
	if !b {
		glog.Errorf("quantity %s can't be converted to int64", q.String())
		return defaultArgs
	}
	return fmt.Sprintf("%dGB", int(float64(i)/math.Pow(2, 30)))
}

// GetPushgatewayImage returns TidbCluster's pushgateway image
func GetPushgatewayImage(cluster *v1.TidbCluster) string {
	if img, ok := cluster.Annotations["pushgateway-image"]; ok && img != "" {
		return img
	}
	return defaultPushgatewayImage
}

// PDSvcName returns pd service name
func PDSvcName(clusterName string) string {
	return fmt.Sprintf("%s-pd", clusterName)
}

// PDClientSvcName returns pd client service name
func PDClientSvcName(clusterName string) string {
	return fmt.Sprintf("%s-pd-client", clusterName)
}

// TiKVSvcName returns tikv service name
func TiKVSvcName(clusterName string) string {
	return fmt.Sprintf("%s-tikv", clusterName)
}

// PDSetName returns pd's statefulset name
func PDSetName(clusterName string) string {
	return fmt.Sprintf("%s-pd", clusterName)
}

// TiKVSetName returns pd's statefulset name
func TiKVSetName(clusterName string) string {
	return fmt.Sprintf("%s-tikv", clusterName)
}

// TiDBSetName returns pd's statefulset name
func TiDBSetName(clusterName string) string {
	return fmt.Sprintf("%s-tidb", clusterName)
}
