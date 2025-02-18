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

// +groupName=br.pingcap.com
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=package
//
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
//
// +kubebuilder:rbac:groups=br,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=br,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=br,resources=persistentvolumes,verbs=get;list;watch
// +kubebuilder:rbac:groups=br,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=br,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=br,resources=events,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=br,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=br,resources=nodes,verbs=get;list;watch
//
// +kubebuilder:rbac:groups=br.pingcap.com,resources=backup,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=br.pingcap.com,resources=backup/status,verbs=get;update;patch
//
// +kubebuilder:rbac:groups=br.pingcap.com,resources=restore,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=br.pingcap.com,resources=restore/status,verbs=get;update;patch
//
// +kubebuilder:rbac:groups=br.pingcap.com,resources=backupschedules,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=br.pingcap.com,resources=backupschedules/status,verbs=get;update;patch
//
// +kubebuilder:rbac:groups=br.pingcap.com,resources=compactbackup,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=br.pingcap.com,resources=compactbackup/status,verbs=get;update;patch
//
// +kubebuilder:rbac:resources="",verbs=get,urls=/metrics
//
// Package v1alpha1 is the v1alpha1 version of br tidb operator api
package v1alpha1
