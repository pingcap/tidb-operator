// Copyright 2019. PingCAP, Inc.
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

package strategy

import (
	"fmt"
	"sync"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var Registry = NewRegistry()

func init() {
	for _, s := range v1alpha1.Strategies {
		Registry.Register(s)
	}
}

// StrategyRegistry maintain the map of resource GVK to its CreateUpdateStrategy
// TODO: automate the registration of ValidatingAdmissionWebhook and MutatingAdmissionWebhook based on this registry
type StrategyRegistry struct {
	sync.RWMutex
	gvkToStrategy map[metav1.GroupVersionKind]v1alpha1.CreateUpdateStrategy
}

func NewRegistry() StrategyRegistry {
	return StrategyRegistry{
		gvkToStrategy: map[metav1.GroupVersionKind]v1alpha1.CreateUpdateStrategy{},
	}
}

func (r *StrategyRegistry) Register(strategy v1alpha1.CreateUpdateStrategy) {
	r.Lock()
	defer r.Unlock()
	obj := strategy.NewObject()
	gvk, err := controller.InferObjectKind(obj)
	if err != nil {
		// impossible
		panic(fmt.Errorf("Object type %T has not been registered in scheme", obj))
	}
	metaGVK := metav1.GroupVersionKind{
		Group:   gvk.Group,
		Version: gvk.Version,
		Kind:    gvk.Kind,
	}
	r.gvkToStrategy[metaGVK] = strategy
}

func (r *StrategyRegistry) Get(kind metav1.GroupVersionKind) (v1alpha1.CreateUpdateStrategy, bool) {
	r.RLock()
	defer r.RUnlock()
	s, ok := r.gvkToStrategy[kind]
	return s, ok
}
