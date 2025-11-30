// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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

func TaskBoot(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("Boot", func(ctx context.Context) task.Result {
		pdg := state.PDGroup()
		if !state.IsBootstrapped && !pdg.Spec.Bootstrapped {
			return task.Wait().With("skip the task and wait until the pd svc is available")
		}
		if !pdg.Spec.Bootstrapped {
			pdg.Spec.Bootstrapped = true
			if err := c.Update(ctx, pdg); err != nil {
				return task.Fail().With("pd cluster is available but not marked as bootstrapped: %w", err)
			}
		}

		var unready []string
		pds := state.InstanceSlice()
		for _, pd := range pds {
			if _, ok := pd.Annotations[v1alpha1.AnnoKeyInitialClusterNum]; !ok {
				continue
			}
			if !coreutil.IsReady[scope.PD](pd) {
				unready = append(unready, pd.Name)
			}
		}
		if len(unready) > 0 {
			return task.Wait().With("wait until pds %v are ready", unready)
		}

		for _, pd := range pds {
			if _, ok := pd.Annotations[v1alpha1.AnnoKeyInitialClusterNum]; !ok {
				continue
			}
			if err := delBootAnnotation(ctx, c, pd); err != nil {
				return task.Fail().With("cannot del boot annotation after pd %s is available: %v", pd.Name, err)
			}
		}

		return task.Complete().With("pd cluster is bootstrapped")
	})
}

type Patch struct {
	Metadata Metadata `json:"metadata"`
}

type Metadata struct {
	ResourceVersion string             `json:"resourceVersion"`
	Annotations     map[string]*string `json:"annotations"`
}

// We have to del boot annotation after PD is bootstrapped.
// If a PD instance has boot annotation, it will try to generate a bootstrap configmap to init a cluster.
// If the boot annotation is not removed, PD controller cannot sync cm and will keep to check
// whether initial number of pd instances are as expected.
// Removing this annotation for a bootstrapped PD cluster is safe
// because the bootstrap config fields are only worked when bootstrapping.
func delBootAnnotation(ctx context.Context, c client.Client, pd *v1alpha1.PD) error {
	p := Patch{
		Metadata: Metadata{
			ResourceVersion: pd.GetResourceVersion(),
			Annotations: map[string]*string{
				v1alpha1.AnnoKeyInitialClusterNum: nil,
			},
		},
	}
	data, err := json.Marshal(&p)
	if err != nil {
		return fmt.Errorf("invalid patch: %w", err)
	}

	if err := c.Patch(ctx, pd, client.RawPatch(types.MergePatchType, data)); err != nil {
		return fmt.Errorf("cannot del boot annotation for %s/%s: %w", pd.GetNamespace(), pd.GetName(), err)
	}

	return nil
}
