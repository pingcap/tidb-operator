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

	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
)

type Object interface {
	metav1.Object

	Component() string

	SetCluster(cluster string)
	Cluster() string

	SetConditions([]metav1.Condition)
	Conditions() []metav1.Condition

	SetObservedGeneration(int64)
	ObservedGeneration() int64

	SetVersion(versions string)
	Version() string

	Features() []metav1alpha1.Feature

	// tls secret name for the tidb operator to visit internal components
	ClientCertKeyPairSecretName() string
	ClientCASecretName() string
	ClientInsecureSkipTLSVerify() bool

	// tls secret name for internal communication between components
	ClusterCertKeyPairSecretName() string
	ClusterCASecretName() string
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

func Component[T ObjectSet, O ObjectT[T]]() string {
	// TODO(liubo02): new only once, now it's ok because only used in test
	var o O = new(T)
	return o.Component()
}
