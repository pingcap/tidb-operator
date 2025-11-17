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

// Package tracker provides utilities for tracking and managing instances within groups.
// It supports dynamic allocation of instance names and maintains state across namespace boundaries.
// This package is primarily used by updater to to allocate stable names for scaling operations.
package tracker

import (
	"sync"

	"k8s.io/apimachinery/pkg/util/sets"

	maputil "github.com/pingcap/tidb-operator/pkg/utils/map"
	"github.com/pingcap/tidb-operator/pkg/utils/random"
)

// Factory creates and manages Tracker and AllocateFactory instances for different kinds of resources.
// Each kind (e.g., "tidb", "tikv", "pd") maintains its own separate tracking state.
type Factory interface {
	// Tracker returns a Tracker instance for the specified kind of resource
	Tracker(kind string) Tracker
	// AllocateFactory returns an AllocateFactory instance for the specified kind of resource
	AllocateFactory(kind string) AllocateFactory
}

// Tracker tracks changes of instances within groups across namespaces.
// It maintains the relationship between instance and their groups,
// allowing for proper cleanup when instances are deleted.
type Tracker interface {
	// Track registers an instance with the given namespace, name, and group
	Track(ns, name, group string)
	// Untrack removes an instance from tracking when it's deleted
	Untrack(ns, name string)
}

// AllocateFactory creates Allocator to allocate instance names.
type AllocateFactory interface {
	// New creates an Allocator for the given namespace and group, with observed instance names
	New(ns, group string, instanceNames ...string) Allocator
}

// Allocator provides deterministic allocation of instance names based on index.
// It ensures that previously allocated names are reused consistently and handles
// name generation for new instances.
// Allocate can be called multiple times in one reconciliation.
// Allocator should be new again in next reconciliation.
type Allocator interface {
	// Allocate returns the instance name for the given index, creating new names as needed
	Allocate(index int) string
}

// factory is the concrete implementation of Factory interface.
// It maintains a map of trackers, one for each resource kind.
type factory struct {
	// ts maps resource kind to its corresponding tracker instance
	ts map[string]*tracker
}

// New creates a new Factory instance for managing trackers across different resource kinds.
func New() Factory {
	f := &factory{
		ts: make(map[string]*tracker),
	}
	return f
}

// tracker returns the tracker instance for the given kind, creating one if it doesn't exist.
func (f *factory) tracker(kind string) *tracker {
	t, ok := f.ts[kind]
	if !ok {
		t = &tracker{}
		f.ts[kind] = t
	}
	return t
}

// Tracker returns a Tracker interface for the specified resource kind.
func (f *factory) Tracker(kind string) Tracker {
	return f.tracker(kind)
}

// AllocateFactory returns an AllocateFactory interface for the specified resource kind.
// Note that the same tracker instance implements both Tracker and AllocateFactory interfaces.
func (f *factory) AllocateFactory(kind string) AllocateFactory {
	return f.tracker(kind)
}

// tracker is the concrete implementation of both Tracker and AllocateFactory interfaces.
// It manages namespacedTracker instances for each namespace, providing thread-safe access
// through a concurrent map.
type tracker struct {
	// ts is a thread-safe map from namespace to namespacedTracker
	ts maputil.Map[string, *namespacedTracker]
}

// Track registers an instance in the specified namespace with its name and group.
// If the namespace doesn't exist, it creates a new namespacedTracker for it.
func (t *tracker) Track(ns, name, group string) {
	nt, _ := t.ts.LoadOrStore(ns, &namespacedTracker{
		groupToInstance: make(map[string]*nameSet),
	})

	nt.track(name, group)
}

// Untrack removes an instance from tracking in the specified namespace.
// If the namespace doesn't exist, it creates a new namespacedTracker for consistency.
func (t *tracker) Untrack(ns, name string) {
	nt, _ := t.ts.LoadOrStore(ns, &namespacedTracker{
		groupToInstance: make(map[string]*nameSet),
	})

	nt.untrack(name)
}

// New creates an Allocator for the specified namespace and group.
// It takes optional existing instance names to track them before allocation begins.
func (t *tracker) New(ns, group string, instanceNames ...string) Allocator {
	nt, _ := t.ts.LoadOrStore(ns, &namespacedTracker{
		groupToInstance: make(map[string]*nameSet),
	})

	return nt.allocator(group, instanceNames...)
}

// NewNameFunc is a function type that generates new instance names.
type NewNameFunc func() string

// suffixLen defines the length of the random suffix appended to generated instance names.
const suffixLen = 6

// namespacedTracker manages instance tracking within a single namespace.
// It maintains mappings between instance names and their groups, and provides
// thread-safe operations for tracking and name allocation.
type namespacedTracker struct {
	// instances maps instance name to instance metadata
	instances maputil.Map[string, *instance]
	// groupToInstance maps group name to the set of instance names in that group
	groupToInstance map[string]*nameSet

	// l protects all fields of this struct from concurrent access
	l sync.Mutex
}

// instance represents a tracked instance with its name and group association.
type instance struct {
	// group may be empty if instance is unmanaged (not part of any group)
	group string
	// name is the unique identifier for this instance within the namespace
	name string
}

// nameSet manages a collection of instance names within a specific group.
// It tracks both the complete set of names and maintains allocation state
// for deterministic name assignment.
type nameSet struct {
	// names contains all instance names that belong to this specific group.
	// It's updated when Track or allocation operations are called.
	names sets.Set[string]
	// unseen manages names that haven't been observed yet and handles allocation
	// of new names when needed. It's updated when reset or Allocate is called.
	unseen *unseen
}

// unseen tracks instance names that have been allocated but not yet observed in the system.
// It provides deterministic name allocation and handles generation of new names.
type unseen struct {
	// items maintains the ordered list of unseen names for deterministic allocation
	items []string
	// set provides O(1) lookup for membership testing
	set sets.Set[string]

	// newNameFunc generates new instance names when allocation exceeds available names
	newNameFunc NewNameFunc
}

// reset updates the unseen set by removing observed names and cleaning up
// names that are no longer tracked. It maintains the items slice for allocation.
func (u *unseen) reset(tracked sets.Set[string], observed ...string) {
	for _, name := range observed {
		u.set.Delete(name)
	}
	for name := range u.set {
		if !tracked.Has(name) {
			u.set.Delete(name)
		}
	}

	u.items = sets.List(u.set)
}

// Allocate returns the instance name for the given index.
// If the index is within existing items, returns the existing name.
// Otherwise, generates a new name using newNameFunc and adds it to the tracking.
func (u *unseen) Allocate(index int) string {
	if index < len(u.items) {
		return u.items[index]
	}

	name := u.newNameFunc()
	u.items = append(u.items, name)
	u.set.Insert(name)
	return name
}

// observe updates the unseen state by marking the given names as observed.
// This affects future allocations by removing these names from the unseen pool.
func (s *nameSet) observe(names ...string) {
	s.unseen.reset(s.names, names...)
}

// NewNameFunc creates a function that generates unique instance names for the given group.
// The generated names follow the pattern "{group}-{randomSuffix}" where randomSuffix
// is a random string of suffixLen characters. It attempts up to 100 times to generate
// a unique name before panicking.
func (t *namespacedTracker) NewNameFunc(g string) NewNameFunc {
	return func() string {
		for range 100 {
			name := g + "-" + random.Random(suffixLen)

			if t.addIfNotExists(name, g) {
				return name
			}
		}

		panic("cannot new a name")
	}
}

// track registers an instance name within this namespace.
// It's a thread-safe wrapper around the add method.
func (t *namespacedTracker) track(name, g string) {
	t.add(name, g)
}

// untrack removes an instance name from this namespace.
// It's a thread-safe wrapper around the del method.
func (t *namespacedTracker) untrack(name string) {
	t.del(name)
}

// allocator creates an Allocator for the given group, optionally observing existing names.
// It ensures the group's nameSet exists and updates its observed state before returning
// the unseen allocator.
func (t *namespacedTracker) allocator(g string, names ...string) Allocator {
	t.l.Lock()
	defer t.l.Unlock()

	s := t.getOrNewNameSet(g)
	s.observe(names...)

	return s.unseen
}

// getOrNewNameSet returns the nameSet for the given group, creating it if it doesn't exist.
// This method must be called with the mutex held.
func (t *namespacedTracker) getOrNewNameSet(g string) *nameSet {
	s, ok := t.groupToInstance[g]
	if !ok {
		s = &nameSet{
			names: sets.New[string](),
			unseen: &unseen{
				set:         sets.New[string](),
				newNameFunc: t.NewNameFunc(g),
			},
		}
		t.groupToInstance[g] = s
	}
	return s
}

// addNameToGroup adds an instance name to its group's name set.
// If the group is empty (unmanaged instance), this is a no-op.
// This method must be called with the mutex held.
func (t *namespacedTracker) addNameToGroup(name, g string) {
	// unmanaged instance
	if g == "" {
		return
	}

	t.l.Lock()
	defer t.l.Unlock()

	s := t.getOrNewNameSet(g)
	s.names.Insert(name)
}

// delNameFromGroup removes an instance name from its group's name set.
// If the group is empty (unmanaged instance) or doesn't exist, this is a no-op.
// This method must be called with the mutex held.
func (t *namespacedTracker) delNameFromGroup(name, g string) {
	// unmanaged instance
	if g == "" {
		return
	}

	t.l.Lock()
	defer t.l.Unlock()

	s, ok := t.groupToInstance[g]
	if !ok {
		return
	}
	s.names.Delete(name)
}

// addIfNotExists adds an instance if it doesn't already exist.
// Returns true if the instance was added, false if it already existed.
// This method must be called with the mutex held.
func (t *namespacedTracker) addIfNotExists(name, g string) bool {
	_, loaded := t.instances.LoadOrStore(name, &instance{
		group: g,
		name:  name,
	})
	if loaded {
		return false
	}
	t.addNameToGroup(name, g)
	return true
}

// add registers an instance, handling both new instances and group changes.
// If the instance exists in a different group, it moves the instance to the new group.
// This method must be called with the mutex held.
func (t *namespacedTracker) add(name, g string) {
	in, loaded := t.instances.LoadOrStore(name, &instance{
		group: g,
		name:  name,
	})
	if !loaded {
		t.addNameToGroup(name, g)
		return
	}
	if in.group == g {
		return
	}
	t.delNameFromGroup(name, in.group)
	t.addNameToGroup(name, g)
}

// del removes an instance from tracking.
// It removes the instance from both the instances map and its group's name set.
// This method must be called with the mutex held.
func (t *namespacedTracker) del(name string) {
	in, loaded := t.instances.LoadAndDelete(name)
	if !loaded {
		return
	}
	t.delNameFromGroup(name, in.group)
}
