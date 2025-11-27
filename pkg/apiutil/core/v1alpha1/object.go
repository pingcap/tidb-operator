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
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/compare"
)

func Cluster[
	S scope.Object[F, T],
	F client.Object,
	T runtime.Object,
](f F) string {
	return scope.From[S](f).Cluster()
}

func SetCluster[
	S scope.Object[F, T],
	F client.Object,
	T runtime.Object,
](f F, cluster string) {
	scope.From[S](f).SetCluster(cluster)
}

func IsSynced[
	S scope.Object[F, T],
	F client.Object,
	T runtime.Object,
](f F) bool {
	t := scope.From[S](f)
	return meta.IsStatusConditionTrue(t.Conditions(), v1alpha1.CondSynced)
}

func SetStatusObservedGeneration[
	S scope.Object[F, T],
	F client.Object,
	T runtime.Object,
](f F) bool {
	t := scope.From[S](f)
	gen := t.ObservedGeneration()
	if compare.SetIfChanged(&gen, t.GetGeneration()) {
		t.SetObservedGeneration(gen)
		return true
	}

	return false
}

func SetStatusCondition[
	S scope.Object[F, T],
	F client.Object,
	T runtime.Object,
](f F, conds ...metav1.Condition) bool {
	obj := scope.From[S](f)
	cur := obj.Conditions()
	needUpdate := false
	for _, cond := range conds {
		cond.ObservedGeneration = obj.GetGeneration()
		needUpdate = meta.SetStatusCondition(&cur, cond) || needUpdate
	}
	if needUpdate {
		obj.SetConditions(cur)
		return true
	}
	return false
}

func RemoveStatusCondition[
	S scope.Object[F, T],
	F client.Object,
	T runtime.Object,
](f F, condTypes ...string) bool {
	obj := scope.From[S](f)
	cur := obj.Conditions()
	needUpdate := false
	for _, condType := range condTypes {
		needUpdate = meta.RemoveStatusCondition(&cur, condType) || needUpdate
	}
	if needUpdate {
		obj.SetConditions(cur)
		return true
	}
	return false
}

func FindStatusCondition[
	S scope.Object[F, T],
	F client.Object,
	T runtime.Object,
](f F, condType string) *metav1.Condition {
	obj := scope.From[S](f)
	return meta.FindStatusCondition(obj.Conditions(), condType)
}

func StatusConditions[
	S scope.Object[F, T],
	F client.Object,
	T runtime.Object,
](f F) []metav1.Condition {
	obj := scope.From[S](f)
	return obj.Conditions()
}

func Features[
	S scope.Object[F, T],
	F client.Object,
	T runtime.Object,
](f F) []metav1alpha1.Feature {
	obj := scope.From[S](f)
	return obj.Features()
}

// LongestReadyPeer returns a ready peer who is ready for the longest time.
func LongestReadyPeer[
	S scope.Object[F, T],
	F client.Object,
	T runtime.Object,
](in F, peers []F) F {
	var choosed F
	var lastTime *time.Time
	for _, peer := range peers {
		if peer.GetName() == in.GetName() {
			continue
		}
		cond := meta.FindStatusCondition(StatusConditions[S](peer), v1alpha1.CondReady)
		if cond == nil || cond.Status != metav1.ConditionTrue {
			continue
		}
		if lastTime == nil {
			lastTime = &cond.LastTransitionTime.Time
			choosed = peer
			continue
		}
		if cond.LastTransitionTime.Time.Before(*lastTime) {
			lastTime = &cond.LastTransitionTime.Time
			choosed = peer
		}
	}

	return choosed
}

func SetVersion[
	S scope.Object[F, T],
	F client.Object,
	T runtime.Object,
](f F, version string) {
	scope.From[S](f).SetVersion(version)
}

func Version[
	S scope.Object[F, T],
	F client.Object,
	T runtime.Object,
](in F) string {
	return scope.From[S](in).Version()
}

func SetImage[
	S scope.Object[F, T],
	F client.Object,
	T runtime.Object,
](f F, image string) {
	scope.From[S](f).SetImage(image)
}

func ClusterCertKeyPairSecretName[
	S scope.Object[F, T],
	F client.Object,
	T runtime.Object,
](f F) string {
	return scope.From[S](f).ClusterCertKeyPairSecretName()
}

func ClusterCASecretName[
	S scope.Object[F, T],
	F client.Object,
	T runtime.Object,
](f F) string {
	return scope.From[S](f).ClusterCASecretName()
}
