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

package coreutil

import (
	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func SchedulingClientPort(s *v1alpha1.Scheduling) int32 {
	if s.Spec.Server.Ports.Client != nil {
		return s.Spec.Server.Ports.Client.Port
	}
	return v1alpha1.DefaultSchedulingPortClient
}

func SchedulingGroupClientPort(sg *v1alpha1.SchedulingGroup) int32 {
	if sg.Spec.Template.Spec.Server.Ports.Client != nil {
		return sg.Spec.Template.Spec.Server.Ports.Client.Port
	}
	return v1alpha1.DefaultSchedulingPortClient
}
