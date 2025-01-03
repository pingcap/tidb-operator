package common

import (
	"context"

	"k8s.io/apimachinery/pkg/labels"

	"github.com/pingcap/tidb-operator/pkg/client"
	revisionutil "github.com/pingcap/tidb-operator/pkg/utils/k8s/revision"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/third_party/kubernetes/pkg/controller/history"
)

type RevisionSetter interface {
	Set(update, current string, collisionCount int32)
}

type RevisionSetterFunc func(update, current string, collisionCount int32)

func (f RevisionSetterFunc) Set(update, current string, collisionCount int32) {
	f(update, current, collisionCount)
}

type Revision interface {
	WithCurrentRevision(CurrentRevisionOption) Revision
	WithCollisionCount(CollisionCountOption) Revision
	WithParent(ParentOption) Revision
	WithLabels(LabelsOption) Revision
	Initializer() RevisionInitializer
}

type RevisionInitializer interface {
	LabelsOption
	ParentOption
	CurrentRevision() string
	CollisionCount() *int32
	RevisionSetter
}

type revision struct {
	RevisionSetter

	parent          ParentOption
	currentRevision CurrentRevisionOption
	collisionCount  CollisionCountOption
	labels          LabelsOption
}

func NewRevision(setter RevisionSetter) Revision {
	return &revision{
		RevisionSetter: setter,
	}
}

func (r *revision) WithParent(parent ParentOption) Revision {
	r.parent = parent
	return r
}

func (r *revision) WithCurrentRevision(rev CurrentRevisionOption) Revision {
	r.currentRevision = rev
	return r
}

func (r *revision) WithCollisionCount(collisionCount CollisionCountOption) Revision {
	r.collisionCount = collisionCount
	return r
}

func (r *revision) WithLabels(ls LabelsOption) Revision {
	r.labels = ls
	return r
}

func (r *revision) Initializer() RevisionInitializer {
	return r
}

func (r *revision) Parent() client.Object {
	return r.parent.Parent()
}

func (r *revision) CurrentRevision() string {
	return r.currentRevision.CurrentRevision()
}

func (r *revision) CollisionCount() *int32 {
	return r.collisionCount.CollisionCount()
}

func (r *revision) Labels() map[string]string {
	return r.labels.Labels()
}

func TaskRevision(state RevisionStateInitializer, c client.Client) task.Task {
	w := state.RevisionInitializer()
	return task.NameTaskFunc("ContextRevision", func(ctx context.Context) task.Result {
		historyCli := history.NewClient(c)
		parent := w.Parent()

		lbs := w.Labels()
		selector := labels.SelectorFromSet(labels.Set(lbs))

		revisions, err := historyCli.ListControllerRevisions(parent, selector)
		if err != nil {
			return task.Fail().With("cannot list controller revisions: %w", err)
		}
		history.SortControllerRevisions(revisions)

		// Get the current(old) and update(new) ControllerRevisions.
		currentRevision, updateRevision, collisionCount, err := revisionutil.GetCurrentAndUpdate(
			ctx,
			parent,
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
