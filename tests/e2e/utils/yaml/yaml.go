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
