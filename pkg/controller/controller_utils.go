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
	"time"

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
	// TestMode defines whether tidb operator run in test mode, test mode is only open when test
	TestMode bool
	// ResyncDuration is the resync time of informer
	ResyncDuration time.Duration
)

const (
	// defaultTiDBSlowLogImage is default image of tidb log tailer
	defaultTiDBLogTailerImage = "busybox:1.26.2"
)

// RequeueError is used to requeue the item, this error type should't be considered as a real error
type RequeueError struct {
	s string
}

func (re *RequeueError) Error() string {
	return re.s
}

// RequeueErrorf returns a RequeueError
func RequeueErrorf(format string, a ...interface{}) error {
	return &RequeueError{fmt.Sprintf(format, a...)}
}

// IsRequeueError returns whether err is a RequeueError
func IsRequeueError(err error) bool {
	_, ok := err.(*RequeueError)
	return ok
}

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
	if limits == nil || limits.Storage == "" {
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

func GetSlowLogTailerImage(cluster *v1alpha1.TidbCluster) string {
	if img := cluster.Spec.TiDB.SlowLogTailer.Image; img != "" {
		return img
	}
	return defaultTiDBLogTailerImage
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

// TiDBPeerMemberName returns tidb peer service name
func TiDBPeerMemberName(clusterName string) string {
	return fmt.Sprintf("%s-tidb-peer", clusterName)
}

// AnnProm adds annotations for prometheus scraping metrics
func AnnProm(port int32) map[string]string {
	return map[string]string{
		"prometheus.io/scrape": "true",
		"prometheus.io/path":   "/metrics",
		"prometheus.io/port":   fmt.Sprintf("%d", port),
	}
}

// MemberConfigMapName returns the default ConfigMap name of the specified member type
func MemberConfigMapName(tc *v1alpha1.TidbCluster, member v1alpha1.MemberType) string {
	nameKey := fmt.Sprintf("%s-%s", tc.Name, member)
	return nameKey + getConfigMapSuffix(tc, member.String(), nameKey)
}

// getConfigMapSuffix return the ConfigMap name suffix
func getConfigMapSuffix(tc *v1alpha1.TidbCluster, component string, name string) string {
	if tc.Annotations == nil {
		return ""
	}
	sha := tc.Annotations[fmt.Sprintf("pingcap.com/%s.%s.sha", component, name)]
	if len(sha) == 0 {
		return ""
	}
	return "-" + sha
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
