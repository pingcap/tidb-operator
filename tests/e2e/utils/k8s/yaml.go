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

package k8s

import (
	"context"
	"errors"
	"io"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
)

// YAMLApplier applies a list of unstructured objects to the K8s cluster.
type YAMLApplier struct {
	DynamicClient dynamic.Interface
	Mapper        *restmapper.DeferredDiscoveryRESTMapper
}

// NewYAMLApplier creates a new YAMLApplier.
func NewYAMLApplier(dynamicClient dynamic.Interface, mapper *restmapper.DeferredDiscoveryRESTMapper) *YAMLApplier {
	return &YAMLApplier{
		DynamicClient: dynamicClient,
		Mapper:        mapper,
	}
}

// Apply applies a list of unstructured objects to the K8s cluster.
func (a *YAMLApplier) Apply(ctx context.Context, r io.Reader) error {
	objs, err := DecodeYAML(r)
	if err != nil {
		return err
	}
	return ApplyUnstructured(ctx, objs, a.DynamicClient, a.Mapper)
}

// DecodeYAML decodes a YAML file into a list of unstructured objects.
func DecodeYAML(r io.Reader) ([]*unstructured.Unstructured, error) {
	var objects []*unstructured.Unstructured
	//nolint:mnd // refactor to a constant if needed
	decoder := yaml.NewYAMLOrJSONDecoder(r, 4096)
	for {
		obj := &unstructured.Unstructured{}
		if err := decoder.Decode(obj); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		objects = append(objects, obj)
	}
	return objects, nil
}

// ApplyUnstructured applies a list of unstructured objects to the K8s cluster.
func ApplyUnstructured(ctx context.Context, objects []*unstructured.Unstructured,
	dynamicClient dynamic.Interface, mapper *restmapper.DeferredDiscoveryRESTMapper) error {
	for _, obj := range objects {
		gvk := obj.GroupVersionKind()
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return err
		}
		gvr := mapping.Resource

		_, err = dynamicClient.Resource(gvr).Namespace(obj.GetNamespace()).Apply(
			ctx, obj.GetName(), obj, metav1.ApplyOptions{FieldManager: "operator-e2e"})
		if err != nil {
			return err
		}
	}
	return nil
}
