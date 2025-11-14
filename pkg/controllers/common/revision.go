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

package common

import (
	"context"

	"k8s.io/apimachinery/pkg/labels"

	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	revisionutil "github.com/pingcap/tidb-operator/v2/pkg/utils/k8s/revision"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/v2/third_party/kubernetes/pkg/controller/history"
)

type RevisionSetter interface {
	Set(update, current string, collisionCount int32)
}

type RevisionSetterFunc func(update, current string, collisionCount int32)

func (f RevisionSetterFunc) Set(update, current string, collisionCount int32) {
	f(update, current, collisionCount)
}

type Revision[G runtime.Group] interface {
	WithCurrentRevision(CurrentRevisionOption) Revision[G]
	WithCollisionCount(CollisionCountOption) Revision[G]
	WithParent(ParentOption[G]) Revision[G]
	WithLabels(LabelsOption) Revision[G]
	Initializer() RevisionInitializer[G]
}

type RevisionInitializer[G runtime.Group] interface {
	LabelsOption
	ParentOption[G]
	CurrentRevision() string
	CollisionCount() *int32
	RevisionSetter
}

type revision[G runtime.Group] struct {
	RevisionSetter

	parent          ParentOption[G]
	currentRevision CurrentRevisionOption
	collisionCount  CollisionCountOption
	labels          LabelsOption
}

func NewRevision[G runtime.Group](setter RevisionSetter) Revision[G] {
	return &revision[G]{
		RevisionSetter: setter,
	}
}

func (r *revision[G]) WithParent(parent ParentOption[G]) Revision[G] {
	r.parent = parent
	return r
}

func (r *revision[G]) WithCurrentRevision(rev CurrentRevisionOption) Revision[G] {
	r.currentRevision = rev
	return r
}

func (r *revision[G]) WithCollisionCount(collisionCount CollisionCountOption) Revision[G] {
	r.collisionCount = collisionCount
	return r
}

func (r *revision[G]) WithLabels(ls LabelsOption) Revision[G] {
	r.labels = ls
	return r
}

func (r *revision[G]) Initializer() RevisionInitializer[G] {
	return r
}

func (r *revision[G]) Parent() G {
	return r.parent.Parent()
}

func (r *revision[G]) CurrentRevision() string {
	return r.currentRevision.CurrentRevision()
}

func (r *revision[G]) CollisionCount() *int32 {
	return r.collisionCount.CollisionCount()
}

func (r *revision[G]) Labels() map[string]string {
	return r.labels.Labels()
}

func TaskRevision[
	GT runtime.GroupTuple[OG, RG],
	OG client.Object,
	RG runtime.Group,
](state RevisionStateInitializer[RG], c client.Client) task.Task {
	var gt GT
	w := state.RevisionInitializer()
	return task.NameTaskFunc("ContextRevision", func(ctx context.Context) task.Result {
		parent := w.Parent()
		historyCli := history.NewClient(c, parent.Component())

		lbs := w.Labels()
		selector := labels.SelectorFromSet(labels.Set(lbs))

		revisions, err := historyCli.ListControllerRevisions(ctx, gt.To(parent), selector)
		if err != nil {
			return task.Fail().With("cannot list controller revisions: %w", err)
		}
		history.SortControllerRevisions(revisions)

		// Get the current(old) and update(new) ControllerRevisions.
		currentRevision, updateRevision, collisionCount, err := revisionutil.GetCurrentAndUpdate(
			ctx,
			gt.To(parent),
			parent.Component(),
			lbs,
			revisions,
			historyCli,
			w.CurrentRevision(),
			w.CollisionCount(),
		)
		if err != nil {
			return task.Fail().With("cannot get revisions: %w", err)
		}

		w.Set(updateRevision.Name, currentRevision.Name, collisionCount)
		return task.Complete().With("revision is set")
	})
}
