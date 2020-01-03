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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Pod
// +k8s:openapi-gen=true
// +resource:path=pods
type Pod struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PodSpec   `json:"spec,omitempty"`
	Status PodStatus `json:"status,omitempty"`
}

type PodStatus struct {
	Phase string `json:"phase,omitempty"`
}

// PodSpec v1beta1 can run a group of containers and could have tolerations
// +genregister:unversioned=false
type PodSpec struct {
	Containers   []ContainerSpec   `json:"container,omitempty"`
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	Tolerations  []string          `json:"tolerations,omitempty"`
	HostName     string            `json:"hostName,omitempty"`
}

// +genregister:unversioned=false
type ContainerSpec struct {
	Image           string `json:"image"`
	ImagePullPolicy string `json:"imagePullPolicy"`
}
