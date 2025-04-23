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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils"
)

func Cluster[
	S scope.Object[F, T],
	F client.Object,
	T runtime.Object,
](f F) string {
	return scope.From[S](f).Cluster()
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
	if utils.SetIfChanged(&gen, t.GetGeneration()) {
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
