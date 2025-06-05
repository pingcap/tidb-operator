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
	"k8s.io/apimachinery/pkg/util/sets"

	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
)

type Gates interface {
	Enabled(feat meta.Feature) bool
}

type gates struct {
	s sets.Set[meta.Feature]
}

func (g *gates) Enabled(feat meta.Feature) bool {
	return g.s.Has(feat)
}

func New[
	S scope.Object[F, T],
	F client.Object,
	T runtime.Object,
](obj F) Gates {
	return NewFromFeatures(coreutil.Features[S](obj))
}

func NewFromFeatures(fs []meta.Feature) Gates {
	return &gates{
		s: sets.New(fs...),
	}
}
