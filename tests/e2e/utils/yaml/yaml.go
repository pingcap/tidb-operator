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

package yaml

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/pingcap/tidb-operator/pkg/client"
)

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

func ApplyFile(ctx context.Context, c client.Client, file string, opts ...Option) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	objs, err := DecodeYAML(f)
	if err != nil {
		return err
	}
	for _, obj := range objs {
		for _, opt := range opts {
			opt.Apply(obj)
		}
		if err := c.Apply(ctx, obj); err != nil {
			return err
		}
	}

	return nil
}

type Option interface {
	Apply(obj *unstructured.Unstructured)
}

type Namespace string

func (n Namespace) Apply(obj *unstructured.Unstructured) {
	obj.SetNamespace(string(n))
}

type IgnoreResources struct{}

func (IgnoreResources) Apply(obj *unstructured.Unstructured) {
	unstructured.RemoveNestedField(obj.Object, "spec", "template", "spec", "resources")
}

func ApplyDir(ctx context.Context, c client.Client, dir string, opts ...Option) error {
	return filepath.WalkDir(dir, func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		return ApplyFile(ctx, c, name, opts...)
	})
}
