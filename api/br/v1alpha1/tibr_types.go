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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=br
// +kubebuilder:resource:shortName="tibr"
// +kubebuilder:selectablefield:JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="ScheduleType",type=string,JSONPath=`.spec.autoSchedule.type`
// +kubebuilder:printcolumn:name="ScheduleAt",type=string,JSONPath=`.spec.autoSchedule.at`
// +kubebuilder:printcolumn:name="Synced",type=string,JSONPath=`.status.conditions[?(@.type=="Synced")].status`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// TiBR is the Schema for the tibr API.
// 1. Allows user to define auto backup schedule.
// 2. Provides http server for user to query backup records, execute restore and query restore progress.
type TiBR struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TiBRSpec   `json:"spec,omitempty"`
	Status TiBRStatus `json:"status,omitempty"`
}

// TiBRSpec defines the desired state of TiBR.
type TiBRSpec struct {
	// Cluster is a reference of tidb cluster
	Cluster v1alpha1.ClusterReference `json:"cluster"`
	// AutoSchedule define auto backup strategy for TiBR, it's optional
	AutoSchedule *TiBRAutoSchedule `json:"autoSchedule,omitempty"`
	// Image is image of br service, default is pingcap/tikv
	Image  *string             `json:"image,omitempty"`
	Config v1alpha1.ConfigFile `json:"config,omitempty"`
	// Overlay defines a k8s native resource template patch
	// Notice: there will be two containers in the pod, one for auto backup, one as api server,if you want to overlay them please use the right name
	Overlay *v1alpha1.Overlay `json:"overlay,omitempty"`
}

const (
	ContainerAutoBackup = "auto-backup" // the container name for auto backup
	ContainerAPIServer  = "api-server"  // the container name for api server
)

// TiBRStatus defines the observed state of TiBR.
type TiBRStatus struct {
	v1alpha1.CommonStatus `json:",inline"`
}

const (
	TiBRCondReady  = "Ready"
	TiBRCondSynced = "Synced"

	// TODO: need to make it more specific
	TiBRReasonNotReady = "NotReady"
	TiBRReasonReady    = "Ready"
	TiBRReasonUnSynced = "UnSynced"
	TiBRReasonSynced   = "Synced"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// TiBRList contains a list of TiBR.
type TiBRList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TiBR `json:"items"`
}

type TiBRAutoScheduleType string

const (
	TiBRAutoScheduleTypePerDay    TiBRAutoScheduleType = "per-day"
	TiBRAutoScheduleTypePerHour   TiBRAutoScheduleType = "per-hour"
	TiBRAutoScheduleTypePerMinute TiBRAutoScheduleType = "per-minute"
)

// +kubebuilder:validation:XValidation:rule="(self.type== 'per-day' && self.at >= 0 && self.at <= 23)|| (self.type == 'per-hour' && self.at >= 0 && self.at <= 59) || (self.type == 'per-minute' && self.at == 0)"
// +kubebuilder:validation:Message="if type is per-day, at is the hour of day; if type is per-hour, at is the minute of hour; if type is per-minute, it should be 0"
type TiBRAutoSchedule struct {
	// Type defines the schedule type, such as per-day, per-hour, per-minute
	// +kubebuilder:validation:Enum=per-day;per-hour;per-minute
	Type TiBRAutoScheduleType `json:"type"`
	// FIXME: it's hard to control the time in current auto backup runtime binary, this parameter is not actually works currently.
	// At defines the schedule time of backup
	At uint32 `json:"at"`
}
