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

package apicall

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
)

type OfflinePatch struct {
	Metadata OfflineMetadata `json:"metadata"`
	Spec     *OfflineSpec    `json:"spec,omitempty"`
}

type OfflineMetadata struct {
	ResourceVersion string             `json:"resourceVersion"`
	Annotations     map[string]*string `json:"annotations,omitempty"`
}

type OfflineSpec struct {
	Offline bool `json:"offline"`
}

// SetOffline set spec.offline for the store
func SetOffline[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](ctx context.Context, c client.Client, obj F) error {
	p := OfflinePatch{
		Metadata: OfflineMetadata{
			ResourceVersion: obj.GetResourceVersion(),
			Annotations: map[string]*string{
				// Always delete defer deletion annotation to ensure
				// the store deletion can be canceled.
				v1alpha1.AnnoKeyDeferDelete: nil,
			},
		},
		Spec: &OfflineSpec{
			Offline: true,
		},
	}

	data, err := json.Marshal(&p)
	if err != nil {
		return fmt.Errorf("failed to set offline, invalid patch: %w", err)
	}

	return c.Patch(ctx, obj, client.RawPatch(types.MergePatchType, data))
}
