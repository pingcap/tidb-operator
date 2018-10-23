// Copyright 2018 PingCAP, Inc.
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

package predicates

import (
	apiv1 "k8s.io/api/core/v1"
)

// Predicate is an interface as extender-implemented predicate functions
type Predicate interface {
	// Name return the predicate name
	Name() string

	// Filter function receives a *volume.TiDBVolume, an *apiv1.Pod and an *apiv1.PersistentVolumeClaim,
	// should remove or add a Priority to every volumes,
	// and return whether it is fusing after this predicate, and an error.
	// Implementations must treat the *apiv1.Pod and *apiv1.PersistentVolumeClaim parameter as read-only and not modify it.
	Filter(*apiv1.Pod) (bool, error)
}
