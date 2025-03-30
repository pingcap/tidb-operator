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

package client

import (
	"github.com/go-json-experiment/json"
	jsonv1 "github.com/go-json-experiment/json/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// This function tries to del all null creationTimestamps so that the serialized data will not contain any null
// creationTimestamps.
func convertToUnstructured(gvk schema.GroupVersionKind, obj client.Object) (*unstructured.Unstructured, error) {
	// TODO(liubo02): change to use json v2 of golang
	// We skip all creationTimestamps in crd because of the controller-gen tools.
	// However, the creationTimestamp field always encoded to null and cannot be applied directly
	// because it's not defined in schema.
	// We use this json encoder to skip creationTimestamp
	data, err := json.Marshal(obj, jsonv1.OmitEmptyWithLegacyDefinition(true), json.OmitZeroStructFields(true))
	if err != nil {
		return nil, err
	}

	m := map[string]any{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	m["apiVersion"] = gvk.GroupVersion().String()
	m["kind"] = gvk.Kind

	return &unstructured.Unstructured{
		Object: m,
	}, nil
}
