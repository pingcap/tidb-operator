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
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
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

// MergeFn knows how to merge a desired object into the current object
type MergeFn func(current, desired runtime.Object) error

// GetFn knows how to get the current object
type GetFn func(ns, name string) (runtime.Object, error)

// CreateFn knows how to create the object
type CreateFn func(obj runtime.Object) (runtime.Object, error)

// UpdateFn knows how to update the object
type UpdateFn func(obj runtime.Object) (runtime.Object, error)

type Applier struct {
	Merge  MergeFn
	Get    GetFn
	Create CreateFn
	Update UpdateFn
}

// ApplyHelper is a template method for applying the desired spec of object to apiserver
func ApplyHelper(desired runtime.Object, c *Applier) (runtime.Object, error) {

	meta, ok := desired.(metav1.Object)
	if !ok {
		return nil, fmt.Errorf("given object is not a metav1.Object")
	}
	kind := desired.GetObjectKind().GroupVersionKind().Kind
	ns := meta.GetNamespace()
	name := meta.GetName()

	// 1. try to create and see if there is any conflicts
	created, err := c.Create(desired)
	if errors.IsAlreadyExists(err) {

		// 2. if there is an existing one, get it and patch on it
		currentTemp, err := c.Get(ns, name)
		if err != nil {
			return nil, RequeueErrorf("error getting %s %s/%s during apply: %v", kind, ns, name, err)
		}
		current := currentTemp.DeepCopyObject()
		// 3. if caller wants to become the controller of the object, try adopting
		controller := metav1.GetControllerOf(meta)
		currentMeta, ok := current.(metav1.Object)
		if !ok {
			// it is a bug of the caller to return a runtime.Object which is no a metav1.Object in 'Get' call
			return nil, fmt.Errorf("the existing object is not a metav1.Object")
		}
		if controller != nil {
			err := TryAdoptObject(kind, currentMeta, *controller)
			if err != nil {
				return nil, err
			}
		}
		// 4. merge other changes in the desired object to current one
		err = c.Merge(current, desired)
		if err != nil {
			return nil, err
		}
		// 5. check if current one is mutated after merge to determine whether should we call update
		if !apiequality.Semantic.DeepEqual(currentTemp, current) {
			updated, err := c.Update(current)
			if err != nil {
				return nil, RequeueErrorf("error updating %s %s/%s during apply: %v", kind, ns, name, err)
			}
			return updated, nil
		}

		// no changes to apply, return current one
		return current, nil
	}

	// object do not exist, check the creation result
	if err != nil {
		return nil, RequeueErrorf("error creating %s %s/%s during apply: %v", kind, ns, name, err)
	}
	return created, nil
}

// TryAdoptObject tries to adopt the object in stand of a controller
func TryAdoptObject(kind string, obj metav1.Object, controller metav1.OwnerReference) error {
	oldController := metav1.GetControllerOf(obj)

	// object has no controller, adopt it (non-controller owners are trivial)
	if oldController == nil {
		ownerRefs := obj.GetOwnerReferences()
		if ownerRefs == nil {
			ownerRefs = []metav1.OwnerReference{}
		}
		ownerRefs = append(ownerRefs, controller)
		obj.SetOwnerReferences(ownerRefs)
		return nil
	}

	if oldController.UID != controller.UID {
		// object is actively controlled by other controllers, this is an exception we are unable to handle here
		return fmt.Errorf("error adopt %s/%s, object has is controlled by other controllers", kind, obj.GetName())
	}
	return nil
}
