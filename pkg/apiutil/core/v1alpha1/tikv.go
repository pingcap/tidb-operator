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

import "github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"

func TiKVGroupClientPort(kvg *v1alpha1.TiKVGroup) int32 {
	if kvg.Spec.Template.Spec.Server.Ports.Client != nil {
		return kvg.Spec.Template.Spec.Server.Ports.Client.Port
	}
	return v1alpha1.DefaultTiKVPortClient
}

func TiKVGroupStatusPort(kvg *v1alpha1.TiKVGroup) int32 {
	if kvg.Spec.Template.Spec.Server.Ports.Status != nil {
		return kvg.Spec.Template.Spec.Server.Ports.Status.Port
	}
	return v1alpha1.DefaultTiKVPortStatus
}

func TiKVClientPort(kv *v1alpha1.TiKV) int32 {
	if kv.Spec.Server.Ports.Client != nil {
		return kv.Spec.Server.Ports.Client.Port
	}
	return v1alpha1.DefaultTiKVPortClient
}

func TiKVStatusPort(kv *v1alpha1.TiKV) int32 {
	if kv.Spec.Server.Ports.Status != nil {
		return kv.Spec.Server.Ports.Status.Port
	}
	return v1alpha1.DefaultTiKVPortStatus
}
