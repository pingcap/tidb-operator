// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +groupName=core.pingcap.com
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=package
//
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
//
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumes,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch
//
// +kubebuilder:rbac:groups=apps,resources=controllerrevisions,verbs=get;list;watch;create;update;patch;delete
//
// +kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses,verbs=get;list;watch
// +kubebuilder:rbac:groups=storage.k8s.io,resources=volumeattributesclasses,verbs=get;list;watch
//
// +kubebuilder:rbac:groups=core.pingcap.com,resources=clusters,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=core.pingcap.com,resources=clusters/status,verbs=get;update;patch
//
// +kubebuilder:rbac:groups=core.pingcap.com,resources=pdgroups,verbs=get;list;watch;delete;update
// +kubebuilder:rbac:groups=core.pingcap.com,resources=pdgroups/status,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=core.pingcap.com,resources=pds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.pingcap.com,resources=pds/status,verbs=get;list;watch;update;patch
//
// +kubebuilder:rbac:groups=core.pingcap.com,resources=tikvgroups,verbs=get;list;watch;delete;update
// +kubebuilder:rbac:groups=core.pingcap.com,resources=tikvgroups/status,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=core.pingcap.com,resources=tikvs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.pingcap.com,resources=tikvs/status,verbs=get;list;watch;update;patch
//
// +kubebuilder:rbac:groups=core.pingcap.com,resources=tiflashgroups,verbs=get;list;watch;delete;update
// +kubebuilder:rbac:groups=core.pingcap.com,resources=tiflashgroups/status,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=core.pingcap.com,resources=tiflashes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.pingcap.com,resources=tiflashes/status,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=core.pingcap.com,resources=tidbgroups,verbs=get;list;watch;delete;update
// +kubebuilder:rbac:groups=core.pingcap.com,resources=tidbgroups/status,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=core.pingcap.com,resources=tidbs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.pingcap.com,resources=tidbs/status,verbs=get;list;watch;update;patch
//
// +kubebuilder:rbac:resources="",verbs=get,urls=/metrics
//
// Package v1alpha1 is the v1alpha1 version of core tidb operator api
package v1alpha1
