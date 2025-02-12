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
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
)

func Cluster[
	S scope.Object[F, T],
	F client.Object,
	T runtime.Object,
](f F) string {
	return scope.From[S](f).Cluster()
}

func Version[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](f F) string {
	return scope.From[S](f).Version()
}

func StatusVersion[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](f F) string {
	return scope.From[S](f).StatusVersion()
}

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
