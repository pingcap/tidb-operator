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
	"strings"

	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
)

func IsHealthy[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](f F) bool {
	return scope.From[S](f).IsHealthy()
}

func NamePrefixAndSuffix[
	F client.Object,
](f F) (prefix, suffix string) {
	name := f.GetName()
	index := strings.LastIndexByte(name, '-')
	// TODO(liubo02): validate name to avoid '-' is not found
	if index == -1 {
		panic("cannot get name prefix")
	}
	return name[:index], name[index+1:]
}

// TODO(liubo02): rename to more reasonable one
// TODO(liubo02): move to namer
func PodName[
	S scope.Object[F, T],
	F client.Object,
	T runtime.Object,
](f F) string {
	prefix, suffix := NamePrefixAndSuffix(f)
	return prefix + "-" + scope.Component[S]() + "-" + suffix
}

// TODO(liubo02): move to namer
func TLSClusterSecretName[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](f F) string {
	prefix, _ := NamePrefixAndSuffix(f)
	return prefix + "-" + scope.Component[S]() + "-cluster-secret"
}
