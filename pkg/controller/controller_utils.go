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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// controllerKind contains the schema.GroupVersionKind for this controller type.
	controllerKind = v1alpha1.SchemeGroupVersion.WithKind("TidbCluster")
	// DefaultStorageClassName is the default storageClassName
	DefaultStorageClassName string
	// ClusterScoped controls whether operator should manage kubernetes cluster wide TiDB clusters
	ClusterScoped bool
)

const (
	// defaultPushgatewayImage is default image of pushgateway
	defaultPushgatewayImage = "prom/pushgateway:v0.3.1"
)

// GetOwnerRef returns TidbCluster's OwnerReference
func GetOwnerRef(tc *v1alpha1.TidbCluster) metav1.OwnerReference {
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

// GetServiceType returns member's service type
func GetServiceType(services []v1alpha1.Service, serviceName string) corev1.ServiceType {
	for _, svc := range services {
		if svc.Name == serviceName {
			switch svc.Type {
			case "NodePort":
				return corev1.ServiceTypeNodePort
			case "LoadBalancer":
				return corev1.ServiceTypeLoadBalancer
			default:
				return corev1.ServiceTypeClusterIP
			}
		}
	}
	return corev1.ServiceTypeClusterIP
}

// TiKVCapacity returns string resource requirement,
// tikv uses GB, TB as unit suffix, but it actually means GiB, TiB
func TiKVCapacity(limits *v1alpha1.ResourceRequirement) string {
	defaultArgs := "0"
	if limits == nil || limits.Storage == ""{
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

// DefaultPushGatewayRequest for the TiKV sidecar
func DefaultPushGatewayRequest() corev1.ResourceRequirements {
	rr := corev1.ResourceRequirements{}
	rr.Requests = make(map[corev1.ResourceName]resource.Quantity)
	rr.Limits = make(map[corev1.ResourceName]resource.Quantity)
	rr.Requests[corev1.ResourceCPU] = resource.MustParse("50m")
	rr.Requests[corev1.ResourceMemory] = resource.MustParse("50Mi")
	rr.Limits[corev1.ResourceCPU] = resource.MustParse("100m")
	rr.Limits[corev1.ResourceMemory] = resource.MustParse("100Mi")
	return rr
}

// GetPushgatewayImage returns TidbCluster's pushgateway image
func GetPushgatewayImage(cluster *v1alpha1.TidbCluster) string {
	if img, ok := cluster.Annotations["pushgateway-image"]; ok && img != "" {
		return img
	}
	return defaultPushgatewayImage
}

// PDMemberName returns pd member name
func PDMemberName(clusterName string) string {
	return fmt.Sprintf("%s-pd", clusterName)
}

// PDPeerMemberName returns pd peer service name
func PDPeerMemberName(clusterName string) string {
	return fmt.Sprintf("%s-pd-peer", clusterName)
}

// TiKVMemberName returns tikv member name
func TiKVMemberName(clusterName string) string {
	return fmt.Sprintf("%s-tikv", clusterName)
}

// TiKVPeerMemberName returns tikv peer service name
func TiKVPeerMemberName(clusterName string) string {
	return fmt.Sprintf("%s-tikv-peer", clusterName)
}

// TiDBMemberName returns tidb member name
func TiDBMemberName(clusterName string) string {
	return fmt.Sprintf("%s-tidb", clusterName)
}

// PriTiDBMemberName returns privileged tidb member name
func PriTiDBMemberName(clusterName string) string {
	return fmt.Sprintf("%s-privileged-tidb", clusterName)
}

// MonitorMemberName returns monitor member name
func MonitorMemberName(clusterName string) string {
	return fmt.Sprintf("%s-monitor", clusterName)
}

// AnnProm adds annotations for prometheus scraping metrics
// port is determined by container port in pod spec with name metrics
func AnnProm() map[string]string {
	ann := make(map[string]string)
	ann["prometheus.io/scrape"] = "true"
	ann["prometheus.io/path"] = "/metrics"
	return ann
}

// setIfNotEmpty set the value into map when value in not empty
func setIfNotEmpty(container map[string]string, key, value string) {
	if value != "" {
		container[key] = value
	}
}

// requestTracker is used by unit test for mocking request error
type requestTracker struct {
	requests int
	err      error
	after    int
}

func (rt *requestTracker) errorReady() bool {
	return rt.err != nil && rt.requests >= rt.after
}

func (rt *requestTracker) inc() {
	rt.requests++
}

func (rt *requestTracker) reset() {
	rt.err = nil
	rt.after = 0
}
