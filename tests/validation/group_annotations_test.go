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

package validation

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupTemplateReservedAnnotations(t *testing.T) {
	groups, err := filepath.Glob("crd/core.pingcap.com_*groups.yaml")
	require.NoError(t, err)
	require.NotEmpty(t, groups)

	for _, group := range groups {
		t.Run(filepath.Base(group), func(t *testing.T) {
			validateCELStaticCosts(t, group, "spec", "template", "metadata")
			schema := structuralSchemaFromCRD(t, group, "v1alpha1")
			metadata := schema.Properties["spec"].Properties["template"].Properties["metadata"]
			validate := validateFunc(&metadata)

			empty := map[string]any{}
			custom := map[string]any{"annotations": map[string]any{"example.com/custom": "value"}}
			require.Empty(t, validate(t.Context(), nil, empty, nil))
			require.Empty(t, validate(t.Context(), nil, custom, nil))

			for _, key := range []string{
				"core.pingcap.com/defer-delete",
				"pd.core.pingcap.com/initial-cluster-num",
			} {
				reserved := map[string]any{"annotations": map[string]any{key: ""}}
				require.Len(t, validate(t.Context(), nil, reserved, nil), 1, key)
				require.Len(t, validate(t.Context(), nil, reserved, custom), 1, key)
				require.Empty(t, validate(t.Context(), nil, custom, reserved), key)
			}
		})
	}
}
