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
	"time"

	"github.com/dustin/go-humanize"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	glog "k8s.io/klog"
)

var (
	// controllerKind contains the schema.GroupVersionKind for tidbcluster controller type.
	ControllerKind = v1alpha1.SchemeGroupVersion.WithKind("TidbCluster")

	// BackupControllerKind contains the schema.GroupVersionKind for backup controller type.
	BackupControllerKind = v1alpha1.SchemeGroupVersion.WithKind("Backup")

	// RestoreControllerKind contains the schema.GroupVersionKind for restore controller type.
	RestoreControllerKind = v1alpha1.SchemeGroupVersion.WithKind("Restore")

	// backupScheduleControllerKind contains the schema.GroupVersionKind for backupschedule controller type.
	backupScheduleControllerKind = v1alpha1.SchemeGroupVersion.WithKind("BackupSchedule")

	// DefaultStorageClassName is the default storageClassName
	DefaultStorageClassName string

	// DefaultBackupStorageClassName is the default storageClassName for backup and restore job
	DefaultBackupStorageClassName string

	// TidbBackupManagerImage is the image of tidb backup manager tool
	TidbBackupManagerImage string

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

// IgnoreError is used to ignore this item, this error type should't be considered as a real error, no need to requeue
type IgnoreError struct {
	s string
}

func (re *IgnoreError) Error() string {
	return re.s
}

// IgnoreErrorf returns a IgnoreError
func IgnoreErrorf(format string, a ...interface{}) error {
	return &IgnoreError{fmt.Sprintf(format, a...)}
}

// IsIgnoreError returns whether err is a IgnoreError
func IsIgnoreError(err error) bool {
	_, ok := err.(*IgnoreError)
	return ok
}

// GetOwnerRef returns TidbCluster's OwnerReference
func GetOwnerRef(tc *v1alpha1.TidbCluster) metav1.OwnerReference {
	controller := true
	blockOwnerDeletion := true
	return metav1.OwnerReference{
		APIVersion:         ControllerKind.GroupVersion().String(),
		Kind:               ControllerKind.Kind,
		Name:               tc.GetName(),
		UID:                tc.GetUID(),
		Controller:         &controller,
		BlockOwnerDeletion: &blockOwnerDeletion,
	}
}

// GetBackupOwnerRef returns Backup's OwnerReference
func GetBackupOwnerRef(backup *v1alpha1.Backup) metav1.OwnerReference {
	controller := true
	blockOwnerDeletion := true
	return metav1.OwnerReference{
		APIVersion:         BackupControllerKind.GroupVersion().String(),
		Kind:               BackupControllerKind.Kind,
		Name:               backup.GetName(),
		UID:                backup.GetUID(),
		Controller:         &controller,
		BlockOwnerDeletion: &blockOwnerDeletion,
	}
}

// GetRestoreOwnerRef returns Restore's OwnerReference
func GetRestoreOwnerRef(restore *v1alpha1.Restore) metav1.OwnerReference {
	controller := true
	blockOwnerDeletion := true
	return metav1.OwnerReference{
		APIVersion:         RestoreControllerKind.GroupVersion().String(),
		Kind:               RestoreControllerKind.Kind,
		Name:               restore.GetName(),
		UID:                restore.GetUID(),
		Controller:         &controller,
		BlockOwnerDeletion: &blockOwnerDeletion,
	}
}

// GetBackupScheduleOwnerRef returns BackupSchedule's OwnerReference
func GetBackupScheduleOwnerRef(bs *v1alpha1.BackupSchedule) metav1.OwnerReference {
	controller := true
	blockOwnerDeletion := true
	return metav1.OwnerReference{
		APIVersion:         backupScheduleControllerKind.GroupVersion().String(),
		Kind:               backupScheduleControllerKind.Kind,
		Name:               bs.GetName(),
		UID:                bs.GetUID(),
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

// TiKVCapacity returns string resource requirement. In tikv-server, KB/MB/GB
// equal to MiB/GiB/TiB, so we cannot use resource.String() directly.
// Minimum unit we use is MiB, capacity less than 1MiB is ignored.
// https://github.com/tikv/tikv/blob/v3.0.3/components/tikv_util/src/config.rs#L155-L168
// For backward compatibility with old TiKV versions, we should use GB/MB
// rather than GiB/MiB, see https://github.com/tikv/tikv/blob/v2.1.16/src/util/config.rs#L359.
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
	if i%humanize.GiByte == 0 {
		return fmt.Sprintf("%dGB", i/humanize.GiByte)
	}
	return fmt.Sprintf("%dMB", i/humanize.MiByte)
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

// PumpMemberName returns pump member name
func PumpMemberName(clusterName string) string {
	return fmt.Sprintf("%s-pump", clusterName)
}

// For backward compatibility, pump peer member name do not has -peer suffix
// PumpPeerMemberName returns pump peer service name
func PumpPeerMemberName(clusterName string) string {
	return fmt.Sprintf("%s-pump", clusterName)
}

// AnnProm adds annotations for prometheus scraping metrics
func AnnProm(port int32) map[string]string {
	return map[string]string{
		"prometheus.io/scrape": "true",
		"prometheus.io/path":   "/metrics",
		"prometheus.io/port":   fmt.Sprintf("%d", port),
	}
}

func ParseStorageRequest(req *v1alpha1.ResourceRequirement) (*corev1.ResourceRequirements, error) {
	if req == nil {
		return nil, fmt.Errorf("storage request is nil")
	}
	size := req.Storage
	q, err := resource.ParseQuantity(size)
	if err != nil {
		return nil, fmt.Errorf("cant' parse storage size: %s", size)
	}
	return &corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceStorage: q,
		},
	}, nil
}

// MemberConfigMapName returns the default ConfigMap name of the specified member type
// Deprecated
// TODO: remove after helm get totally abandoned
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

// Int32Ptr returns a pointer to an int32
func Int32Ptr(i int32) *int32 {
	return &i
}

// RequestTracker is used by unit test for mocking request error
type RequestTracker struct {
	requests int
	err      error
	after    int
}

func (rt *RequestTracker) ErrorReady() bool {
	return rt.err != nil && rt.requests >= rt.after
}

func (rt *RequestTracker) Inc() {
	rt.requests++
}

func (rt *RequestTracker) Reset() {
	rt.err = nil
	rt.after = 0
}

func (rt *RequestTracker) SetError(err error) *RequestTracker {
	rt.err = err
	return rt
}

func (rt *RequestTracker) SetAfter(after int) *RequestTracker {
	rt.after = after
	return rt
}

func (rt *RequestTracker) SetRequests(requests int) *RequestTracker {
	rt.requests = requests
	return rt
}

func (rt *RequestTracker) GetError() error {
	return rt.err
}

// MergeConfigMap is a MergeFn for generic configmap.
// parameter types are set to runtime.Object to match the signature of MergeFn
func MergeConfigMap(existing, desired runtime.Object) error {
	existingCm := existing.(*corev1.ConfigMap)
	desiredCm := desired.(*corev1.ConfigMap)

	existingCm.Data = desiredCm.Data
	existingCm.Labels = desiredCm.Labels
	for k, v := range desiredCm.Annotations {
		existingCm.Annotations[k] = v
	}
	return nil
}
