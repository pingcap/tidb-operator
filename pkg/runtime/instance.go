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

//go:generate ${GOBIN}/mockgen -write_command_comment=false -copyright_file ${BOILERPLATE_FILE} -destination instance_mock_generated.go -package=runtime ${GO_MODULE}/pkg/runtime Instance
package runtime

import (
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

type Instance interface {
	Object

	GetTopology() v1alpha1.Topology
	SetTopology(topo v1alpha1.Topology)

	GetUpdateRevision() string
	IsReady() bool
	// IsUpToDate means all resources managed by the instance is up to date
	// NOTE: It does not mean the instance is updated to the newest revision
	// TODO: may be change a more meaningful name?
	IsUpToDate() bool
	IsOffline() bool

	CurrentRevision() string
	SetCurrentRevision(rev string)

	PodOverlay() *v1alpha1.PodOverlay

	Subdomain() string

	// IsStore indicates whether the instance is a store.
	// For TiKV and TiFlash, it returns true, otherwise it returns false.
	IsStore() bool
}

type InstanceT[T InstanceSet] interface {
	Instance

	*T
}

type InstanceSet interface {
	PD | TiDB | TiKV | TiFlash | TiCDC | TiProxy | TSO | Scheduler
}

type InstanceTuple[PT client.Object, PU Instance] interface {
	Tuple[PT, PU]
}

func SetOfflineCondition(s Instance, condition *metav1.Condition) {
	if condition == nil {
		return
	}

	conditions := s.Conditions()

	// Find existing condition
	updated := false
	for i := range conditions {
		if conditions[i].Type == v1alpha1.StoreOfflinedConditionType {
			conditions[i] = *condition
			updated = true
		}
	}

	if !updated {
		conditions = append(conditions, *condition)
	}
	s.SetConditions(conditions)
}

func GetOfflineCondition(s Instance) *metav1.Condition {
	return meta.FindStatusCondition(s.Conditions(), v1alpha1.StoreOfflinedConditionType)
}

func RemoveOfflineCondition(s Instance) {
	conditions := s.Conditions()
	meta.RemoveStatusCondition(&conditions, v1alpha1.StoreOfflinedConditionType)
	s.SetConditions(conditions)
}

func NamePrefixAndSuffix(name string) (prefix, suffix string) {
	index := strings.LastIndexByte(name, '-')
	// TODO(liubo02): validate name to avoid '-' is not found
	if index == -1 {
		panic("cannot get name prefix")
	}
	return name[:index], name[index+1:]
}
