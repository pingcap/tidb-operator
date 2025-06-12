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

package data

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
)

func NewSchedulingGroup(ns string, patches ...GroupPatch[*runtime.SchedulingGroup]) *v1alpha1.SchedulingGroup {
	sg := &runtime.SchedulingGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      defaultSchedulingGroupName,
		},
		Spec: v1alpha1.SchedulingGroupSpec{
			Cluster:  v1alpha1.ClusterReference{Name: defaultClusterName},
			Replicas: ptr.To[int32](1),
			Template: v1alpha1.SchedulingTemplate{
				Spec: v1alpha1.SchedulingTemplateSpec{
					Version: defaultVersion,
					Image:   ptr.To(defaultImageRegistry + "pd"),
				},
			},
		},
	}
	for _, p := range patches {
		p(sg)
	}

	return runtime.ToSchedulingGroup(sg)
}
