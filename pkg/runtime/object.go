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

package runtime

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type object interface {
	metav1.Object

	Cluster() string
	Component() string
	To() client.Object
	Conditions() []metav1.Condition
}

type Object interface {
	object

	*PDGroup | *TiDBGroup | *TiKVGroup | *TiFlashGroup | *PD | *TiDB | *TiKV | *TiFlash
}
