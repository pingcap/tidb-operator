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

package features

import (
	"sync"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
)

var defaultFeatureGates = NewFeatureGates()

func Enabled(ns, name string, feat meta.Feature) bool {
	return defaultFeatureGates.Enabled(ns, name, feat)
}

func Register(c *v1alpha1.Cluster) {
	defaultFeatureGates.Register(c)
}

func Deregister(ns, name string) {
	defaultFeatureGates.Deregister(ns, name)
}

type FeatureGates interface {
	Enabled(ns, name string, feat meta.Feature) bool

	Register(c *v1alpha1.Cluster)
	Deregister(ns, name string)
}

type gates struct {
	lock sync.Mutex

	features map[string]featureSet
}

type featureSet struct {
	generation int64
	uid        types.UID
	set        sets.Set[meta.Feature]
}

func NewFeatureGates() FeatureGates {
	return &gates{
		features: map[string]featureSet{},
	}
}

func (g *gates) Enabled(ns, name string, feat meta.Feature) bool {
	g.lock.Lock()
	defer g.lock.Unlock()

	key := types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}.String()

	fs, ok := g.features[key]
	if !ok {
		// It should not happened
		// Dangerous to visit an uninitialized feature gates
		panic("feature gates is not initialized")
	}

	return fs.set.Has(feat)
}

func (g *gates) Register(c *v1alpha1.Cluster) {
	g.lock.Lock()
	defer g.lock.Unlock()

	key := types.NamespacedName{
		Namespace: c.Namespace,
		Name:      c.Name,
	}.String()

	fs, ok := g.features[key]
	if !ok || fs.uid != c.UID || fs.generation != c.Generation {
		s := sets.New[meta.Feature]()
		for _, feat := range c.Spec.FeatureGates {
			s.Insert(feat.Name)
		}

		g.features[key] = featureSet{
			generation: c.Generation,
			uid:        c.UID,
			set:        s,
		}
	}
}

func (g *gates) Deregister(ns, name string) {
	g.lock.Lock()
	defer g.lock.Unlock()

	key := types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}.String()

	delete(g.features, key)
}
