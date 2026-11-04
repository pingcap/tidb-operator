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

	"github.com/go-logr/logr"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
)

// hpaScaleInAnnotatingClient writes the HPA scale-in annotation on the same patch
// that marks a TiProxy offline. A non-HPA offline patch clears it, so a stale
// marker cannot survive into a later manual scale-in. Upgrade and explicit delete
// use Delete, so they do not change the annotation.
type hpaScaleInAnnotatingClient struct {
	client.Client
	hpa bool
}

func (c *hpaScaleInAnnotatingClient) Patch(
	ctx context.Context,
	obj ctrlclient.Object,
	patch ctrlclient.Patch,
	opts ...ctrlclient.PatchOption,
) error {
	next, changed, err := withHPAScaleInAnnotation(patch, obj, c.hpa)
	if err != nil {
		return err
	}
	if changed && c.hpa {
		logr.FromContextOrDiscard(ctx).Info("mark TiProxy offline by HPA scale-in",
			"namespace", obj.GetNamespace(),
			"name", obj.GetName(),
		)
	}
	return c.Client.Patch(ctx, obj, next, opts...)
}

func withHPAScaleInAnnotation(
	patch ctrlclient.Patch,
	obj ctrlclient.Object,
	hpa bool,
) (ctrlclient.Patch, bool, error) {
	if _, ok := obj.(*v1alpha1.TiProxy); !ok {
		return patch, false, nil
	}
	data, err := patch.Data(obj)
	if err != nil {
		return nil, false, err
	}

	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, false, fmt.Errorf("tiproxy offline patch is not a JSON object: %w", err)
	}
	spec, ok := body["spec"].(map[string]any)
	if !ok || spec["offline"] != true {
		return patch, false, nil
	}

	metadata, _ := body["metadata"].(map[string]any)
	if metadata == nil {
		metadata = map[string]any{}
		body["metadata"] = metadata
	}
	annotations, _ := metadata["annotations"].(map[string]any)
	if annotations == nil {
		annotations = map[string]any{}
		metadata["annotations"] = annotations
	}
	if hpa {
		annotations[v1alpha1.AnnoKeyTiProxyHPAScaleIn] = v1alpha1.AnnoValTrue
	} else {
		annotations[v1alpha1.AnnoKeyTiProxyHPAScaleIn] = nil
	}

	out, err := json.Marshal(body)
	if err != nil {
		return nil, false, err
	}
	return client.RawPatch(patch.Type(), out), true, nil
}
