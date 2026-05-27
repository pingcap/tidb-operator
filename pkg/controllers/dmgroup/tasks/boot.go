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

package tasks

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func TaskBoot(state State, c client.Client) task.Task {
	return task.NameTaskFunc("Boot", func(ctx context.Context) task.Result {
		var unready []string
		dms := state.InstanceSlice()
		for _, dm := range dms {
			if _, ok := dm.Annotations[v1alpha1.AnnoKeyInitialClusterNum]; !ok {
				continue
			}
			if !coreutil.IsReady[scope.DM](dm) {
				unready = append(unready, dm.Name)
			}
		}
		if len(unready) > 0 {
			return task.Wait().With("wait until dm masters %v are ready", unready)
		}

		for _, dm := range dms {
			if _, ok := dm.Annotations[v1alpha1.AnnoKeyInitialClusterNum]; !ok {
				continue
			}
			if err := delBootAnnotation(ctx, c, dm); err != nil {
				return task.Fail().With("cannot del boot annotation after dm %s is available: %v", dm.Name, err)
			}
		}

		return task.Complete().With("dm-master cluster is bootstrapped")
	})
}

type Patch struct {
	Metadata Metadata `json:"metadata"`
}

type Metadata struct {
	ResourceVersion string             `json:"resourceVersion"`
	Annotations     map[string]*string `json:"annotations"`
}

func delBootAnnotation(ctx context.Context, c client.Client, dm *v1alpha1.DM) error {
	p := Patch{
		Metadata: Metadata{
			ResourceVersion: dm.GetResourceVersion(),
			Annotations: map[string]*string{
				v1alpha1.AnnoKeyInitialClusterNum: nil,
			},
		},
	}
	data, err := json.Marshal(&p)
	if err != nil {
		return fmt.Errorf("invalid patch: %w", err)
	}

	if err := c.Patch(ctx, dm, client.RawPatch(types.MergePatchType, data)); err != nil {
		return fmt.Errorf("cannot del boot annotation for %s/%s: %w", dm.GetNamespace(), dm.GetName(), err)
	}

	return nil
}
