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
	"context"
	"fmt"
	"sync"
	"sync/atomic"

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

func Use(ctx context.Context, c *v1alpha1.Cluster) error {
	return defaultFeatureGates.Use(ctx, c)
}

func Deregister(ns, name string) {
	defaultFeatureGates.Deregister(ns, name)
}

// FeatureGates is defined to make sure all controllers use a same version of feature gates.
// When the cluster is changed, some controllers may still use the cached cluster
// All controllers must call `Use` before really using feature gates.
type FeatureGates interface {
	// Check whether the feature is enabled for cluster ns/name
	Enabled(ns, name string, feat meta.Feature) bool

	// Register the feature gates of this cluster, it should be called in the cluster controller
	Register(c *v1alpha1.Cluster)
	Deregister(ns, name string)

	// Controllers will call this function to lock the version of feature gates until ctx is done.
	// Only when the outdated feature gates are used by 0 users, The newest one can be used by others.
	// When the cluster is changed, all controllers will reconcile and try to use the new version of feature gates.
	// But only when feature gates are not changed or the outdated one is not used by any controllers,
	// this function can succeed.
	// If the newest feature gates are not registered or the outdated feature gates are still used,
	// error will be returned and controllers should try again later.
	Use(ctx context.Context, c *v1alpha1.Cluster) error
}

type gates struct {
	lock sync.RWMutex

	features map[string]*featureSet
	outdated map[string]*featureSet
}

type featureSet struct {
	generation int64
	uid        types.UID
	set        sets.Set[meta.Feature]
	count      atomic.Int32
}

func NewFeatureGates() FeatureGates {
	return &gates{
		features: map[string]*featureSet{},
		outdated: map[string]*featureSet{},
	}
}

func (g *gates) Enabled(ns, name string, feat meta.Feature) bool {
	g.lock.RLock()
	defer g.lock.RUnlock()

	key := types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}.String()

	// always use outdated feature set if exists
	ofs, ok := g.outdated[key]
	if ok && ofs.count.Load() != 0 {
		return ofs.set.Has(feat)
	}

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
	// init feature set
	if !ok {
		s := sets.New[meta.Feature]()
		for _, feat := range c.Spec.FeatureGates {
			s.Insert(feat.Name)
		}

		g.features[key] = &featureSet{
			generation: c.Generation,
			uid:        c.UID,
			set:        s,
		}

		return
	}
	// feature gates have been registered but are changed
	if isFeatureGatesChanged(c, fs) {
		// outdated is not found, record it in outdated
		if ofs, ok := g.outdated[key]; !ok || ofs.count.Load() == 0 {
			g.outdated[key] = fs
		}

		s := sets.New[meta.Feature]()
		for _, feat := range c.Spec.FeatureGates {
			s.Insert(feat.Name)
		}

		g.features[key] = &featureSet{
			generation: c.Generation,
			uid:        c.UID,
			set:        s,
		}
	} else {
		// always update generation and uid
		fs.generation = c.Generation
		fs.uid = c.UID
	}
}

func (g *gates) Use(ctx context.Context, c *v1alpha1.Cluster) error {
	g.lock.Lock()
	defer g.lock.Unlock()

	key := types.NamespacedName{
		Namespace: c.Namespace,
		Name:      c.Name,
	}.String()

	ofs, ok := g.outdated[key]
	if ok {
		if count := ofs.count.Load(); count != 0 {
			return fmt.Errorf("outdated feature gates are still in use by %v", count)
		}
	}

	fs, ok := g.features[key]
	if !ok {
		return fmt.Errorf("feature gates of %s are not registered", key)
	}

	if isFeatureGatesChanged(c, fs) {
		add, del := diff(c, fs)
		return fmt.Errorf("feature gates of %s are not up to date, try to enable: %v, disable: %v",
			key,
			add.UnsortedList(),
			del.UnsortedList(),
		)
	}

	fs.count.Add(1)
	go g.unuse(ctx, fs)

	return nil
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

func (g *gates) unuse(ctx context.Context, fs *featureSet) {
	<-ctx.Done()
	fs.count.Add(-1)
}

func isFeatureGatesChanged(c *v1alpha1.Cluster, fs *featureSet) bool {
	// uid and generation is not changed
	if fs.generation == c.Generation && fs.uid == c.UID {
		return false
	}

	s := sets.New[meta.Feature]()
	for _, feat := range c.Spec.FeatureGates {
		s.Insert(feat.Name)
	}

	return !fs.set.Equal(s)
}

func diff(c *v1alpha1.Cluster, fs *featureSet) (add sets.Set[meta.Feature], del sets.Set[meta.Feature]) {
	s := sets.New[meta.Feature]()
	for _, feat := range c.Spec.FeatureGates {
		s.Insert(feat.Name)
	}
	return s.Difference(fs.set), fs.set.Difference(s)
}
