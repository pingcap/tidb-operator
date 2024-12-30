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

	"github.com/pingcap/tidb-operator/pkg/client"
)

type Object interface {
	metav1.Object

	Cluster() string
	Component() string
	Conditions() []metav1.Condition
	SetConditions([]metav1.Condition)

	ObservedGeneration() int64
	SetObservedGeneration(int64)
}

type ObjectT[T ObjectSet] interface {
	Object

	*T
}

type ObjectSet interface {
	GroupSet | InstanceSet
}

type Tuple[T any, U any] interface {
	From(T) U
	FromSlice([]T) []U
	To(U) T
	ToSlice([]U) []T
}

type ObjectTuple[PT client.Object, PU Object] interface {
	Tuple[PT, PU]
}
