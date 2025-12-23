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

package coreutil

import (
	"maps"

	"k8s.io/apimachinery/pkg/labels"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/compare"
	maputil "github.com/pingcap/tidb-operator/v2/pkg/utils/map"
)

func StatusVersion[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](f F) string {
	return scope.From[S](f).StatusVersion()
}

func Replicas[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](f F) int32 {
	return scope.From[S](f).Replicas()
}

func SetReplicas[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](f F, replicas int32) {
	scope.From[S](f).SetReplicas(replicas)
}

func SetTemplateClusterTLS[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](f F, ca, certKeyPair string) {
	scope.From[S](f).SetTemplateClusterTLS(ca, certKeyPair)
}

func MinReadySeconds[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](f F) int64 {
	return scope.From[S](f).MinReadySeconds()
}

func SchedulePolicies[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](f F) []v1alpha1.SchedulePolicy {
	return scope.From[S](f).SchedulePolicies()
}

// IsGroupHealthyAndUpToDate is defined to check whether all replicas of the group are healthy and up to date
// TODO: simplify it by a condition
func IsGroupHealthyAndUpToDate[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](f F) bool {
	t := scope.From[S](f)
	updateRevision, currentRevision, _ := t.StatusRevision()
	replicas, readyReplicas, updateReplicas, currentReplicas := t.StatusReplicas()
	return t.ObservedGeneration() == t.GetGeneration() &&
		updateRevision == currentRevision &&
		// replicas num is expected, no scale out/in
		t.Replicas() == replicas &&
		// replicas are all ready
		readyReplicas == replicas &&
		// replicas are all ready
		updateReplicas == replicas &&
		currentReplicas == replicas
}

func SetStatusVersion[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](f F) bool {
	obj := scope.From[S](f)
	v := obj.StatusVersion()
	if compare.SetIfNotEmptyAndChanged(&v, obj.Version()) {
		obj.SetStatusVersion(v)
		return true
	}

	return false
}

func SetStatusReplicas[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](f F, newReplicas, newReady, newUpdate, newCurrent int32) bool {
	obj := scope.From[S](f)
	replicas, ready, update, current := obj.StatusReplicas()
	changed := compare.SetIfChanged(&replicas, newReplicas)
	changed = compare.SetIfChanged(&ready, newReady) || changed
	changed = compare.SetIfChanged(&update, newUpdate) || changed
	changed = compare.SetIfChanged(&current, newCurrent) || changed
	if changed {
		obj.SetStatusReplicas(replicas, ready, update, current)
		return changed
	}

	return false
}

func SetStatusRevision[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](f F, newUpdate, newCurrent string, newCollisionCount int32) bool {
	obj := scope.From[S](f)
	update, current, collisionCount := obj.StatusRevision()
	changed := compare.SetIfNotEmptyAndChanged(&update, newUpdate)
	changed = compare.SetIfNotEmptyAndChanged(&current, newCurrent) || changed
	changed = compare.NewAndSetIfNotEmptyAndChanged(&collisionCount, newCollisionCount) || changed
	if changed {
		obj.SetStatusRevision(update, current, collisionCount)
		return changed
	}

	return false
}

func InstanceLabels[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](f F, rev string) map[string]string {
	obj := scope.From[S](f)

	return maputil.Merge(obj.TemplateLabels(), map[string]string{
		v1alpha1.LabelKeyManagedBy:            v1alpha1.LabelValManagedByOperator,
		v1alpha1.LabelKeyComponent:            obj.Component(),
		v1alpha1.LabelKeyCluster:              obj.Cluster(),
		v1alpha1.LabelKeyGroup:                f.GetName(),
		v1alpha1.LabelKeyInstanceRevisionHash: rev,
	})
}

func InstanceAnnotations[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](f F) map[string]string {
	obj := scope.From[S](f)

	return maps.Clone(obj.TemplateAnnotations())
}

func SetStatusSelector[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](f F) bool {
	obj := scope.From[S](f)

	l := obj.StatusSelector()

	changed := compare.SetIfNotEmptyAndChanged(&l, labels.Set{
		v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
		v1alpha1.LabelKeyComponent: obj.Component(),
		v1alpha1.LabelKeyCluster:   obj.Cluster(),
		v1alpha1.LabelKeyGroup:     f.GetName(),
	}.String())

	if changed {
		obj.SetStatusSelector(l)
	}

	return changed
}

func TemplateAnnotations[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](f F) map[string]string {
	obj := scope.From[S](f)

	return obj.TemplateAnnotations()
}

func SetTemplateAnnotations[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](f F, anno map[string]string) {
	obj := scope.From[S](f)

	obj.SetTemplateAnnotations(anno)
}

func HeadlessServiceName[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](f F) string {
	return f.GetName() + "-" + scope.Component[S]() + "-peer"
}
