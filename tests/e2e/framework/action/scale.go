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

package action

import (
	"context"
	"fmt"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/random"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/data"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework"
)

func MustScale[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](
	ctx context.Context,
	f *framework.Framework,
	obj F,
	replicas int32,
) {
	ginkgo.By(fmt.Sprintf("scale %v to %v", client.ObjectKeyFromObject(obj), replicas))

	err := mergePatch[S](ctx, f, obj, data.WithReplicas[S](replicas))

	gomega.ExpectWithOffset(1, err).To(gomega.Succeed())
}

func MustRollingRestart[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](
	ctx context.Context,
	f *framework.Framework,
	obj F,
) {
	ginkgo.By(fmt.Sprintf("rolling restart %v", client.ObjectKeyFromObject(obj)))

	err := mergePatch[S](ctx, f, obj,
		data.WithTemplateAnnotation[S]("rolling-restart-key", random.Random(5)),
	)

	gomega.ExpectWithOffset(1, err).To(gomega.Succeed())
}

func MustScaleAndRollingRestart[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](
	ctx context.Context,
	f *framework.Framework,
	obj F,
	replicas int32,
) {
	ginkgo.By(fmt.Sprintf("scale %v to %v and rolling restart", client.ObjectKeyFromObject(obj), replicas))

	err := mergePatch[S](ctx, f, obj,
		data.WithReplicas[S](replicas),
		data.WithTemplateAnnotation[S]("rolling-restart-key", random.Random(5)),
	)

	gomega.ExpectWithOffset(1, err).To(gomega.Succeed())
}

func MustUpdate[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](
	ctx context.Context,
	f *framework.Framework,
	obj F,
	patches ...data.GroupPatch[F],
) {
	ginkgo.By(fmt.Sprintf("rolling update %v", client.ObjectKeyFromObject(obj)))

	err := mergePatch[S](ctx, f, obj,
		patches...,
	)

	gomega.ExpectWithOffset(1, err).To(gomega.Succeed())
}

func mergePatch[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](
	ctx context.Context,
	f *framework.Framework,
	obj F,
	patches ...data.GroupPatch[F],
) error {
	mp := client.MergeFrom(obj.DeepCopyObject().(client.Object))
	for _, p := range patches {
		p.Patch(obj)
	}
	return f.Client.Patch(ctx, obj, mp)
}
