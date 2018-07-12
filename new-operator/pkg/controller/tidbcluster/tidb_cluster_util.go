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

package tidbcluster

import (
	"fmt"
	"math"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/new-operator/pkg/apis/pingcap.com/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultPushgatewayImage = "prom/pushgateway:v0.3.1"
)

func getOwnerRef(tc *v1.TidbCluster) metav1.OwnerReference {
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

// tikv uses GB, TB as unit suffix, but it actually means GiB, TiB
func tikvCapacity(limits *v1.ResourceRequirement) string {
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

func getPushgatewayImage(cluster *v1.TidbCluster) string {
	if img, ok := cluster.Annotations["pushgateway-image"]; ok && img != "" {
		return img
	}
	return defaultPushgatewayImage
}

func pdSvcName(clusterName string) string {
	return fmt.Sprintf("%s-pd", clusterName)
}

func pdClientSvcName(clusterName string) string {
	return fmt.Sprintf("%s-pd-client", clusterName)
}

func tikvSvcName(clusterName string) string {
	return fmt.Sprintf("%s-tikv", clusterName)
}

func pdSetNameFor(tc *v1.TidbCluster) string {
	return fmt.Sprintf("%s-pdset", tc.GetName())
}

func tidbSetNameFor(tc *v1.TidbCluster) string {
	return fmt.Sprintf("%s-tidbset", tc.GetName())
}

func tikvSetNameFor(tc *v1.TidbCluster) string {
	return fmt.Sprintf("%s-tikvset", tc.GetName())
}
