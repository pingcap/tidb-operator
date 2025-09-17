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

package adoption

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

type instance struct {
	hash     string
	instance *v1alpha1.TiDB
	owner    *v1alpha1.TiDBGroup
}

func (in *instance) isLocked() bool {
	return in.owner != nil
}

func (in *instance) lock(owner *v1alpha1.TiDBGroup) {
	in.owner = owner
}

func (in *instance) unlock() {
	in.owner = nil
}

func (in *instance) ownerKey() string {
	if in.owner == nil {
		return ""
	}
	return client.ObjectKeyFromObject(in.owner).String()
}

func (in *instance) key() string {
	return client.ObjectKeyFromObject(in.instance).String()
}
