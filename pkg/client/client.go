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
	"bytes"
	"context"
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/managedfields"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/kube-openapi/pkg/util/proto"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/structured-merge-diff/v4/typed"

	"github.com/pingcap/tidb-operator/pkg/scheme"
)

const (
	DefaultFieldManager = "tidb-operator"
)

type Client interface {
	client.WithWatch
	Apply(ctx context.Context, obj client.Object) error
	ApplyWithResult(ctx context.Context, obj client.Object) (ApplyResult, error)
}

type ApplyResult int

const (
	ApplyResultUpdated ApplyResult = iota
	ApplyResultUnchanged
	ApplyResultCreated
)

type applier struct {
	client.WithWatch
	parser GVKParser
}

func (p *applier) Apply(ctx context.Context, obj client.Object) error {
	_, err := p.ApplyWithResult(ctx, obj)
	return err
}

func (p *applier) ApplyWithResult(ctx context.Context, obj client.Object) (ApplyResult, error) {
	gvks, _, err := scheme.Scheme.ObjectKinds(obj)
	if err != nil {
		return ApplyResultUnchanged, fmt.Errorf("cannot get gvks of the obj %T: %w", obj, err)
	}
	if len(gvks) == 0 {
		return ApplyResultUnchanged, fmt.Errorf("cannot get gvk of obj %T", obj)
	}

	expected, ok := obj.DeepCopyObject().(client.Object)
	if !ok {
		panic("it's unreachable")
	}

	hasCreated := true
	if err := p.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
		if !errors.IsNotFound(err) {
			return ApplyResultUnchanged, err
		}

		hasCreated = false
	}

	if hasCreated {
		lastApplied := newObject(obj)

		if err := p.Extract(obj, DefaultFieldManager, gvks[0], lastApplied, ""); err != nil {
			return ApplyResultUnchanged, fmt.Errorf("cannot extract last applied patch: %w", err)
		}

		// ignore name, namespace and gvk
		lastApplied.SetName(obj.GetName())
		lastApplied.SetNamespace(obj.GetNamespace())
		lastApplied.GetObjectKind().SetGroupVersionKind(expected.GetObjectKind().GroupVersionKind())

		if equality.Semantic.DeepEqual(expected, lastApplied) {
			return ApplyResultUnchanged, nil
		}
	}

	if err := p.Patch(ctx, obj, &applyPatch{
		expected: expected,
		gvk:      gvks[0],
	}, &client.PatchOptions{
		FieldManager: DefaultFieldManager,
	}); err != nil {
		return ApplyResultUnchanged, fmt.Errorf("cannot apply patch: %w", err)
	}

	if hasCreated {
		return ApplyResultUpdated, nil
	}

	return ApplyResultCreated, nil
}

func (p *applier) Extract(current client.Object, fieldManager string, gvk schema.GroupVersionKind, patch any, subresource string) error {
	tpd := p.parser.Type(gvk)
	if tpd == nil {
		return fmt.Errorf("can't find specified type: %s", gvk)
	}
	if err := managedfields.ExtractInto(current, *tpd, fieldManager, patch, subresource); err != nil {
		return err
	}
	return nil
}

type applyPatch struct {
	expected client.Object
	gvk      schema.GroupVersionKind
}

func (*applyPatch) Type() types.PatchType {
	return types.ApplyPatchType
}

func (p *applyPatch) Data(client.Object) ([]byte, error) {
	encoder := scheme.Codecs.EncoderForVersion(
		json.NewSerializerWithOptions(
			json.DefaultMetaFactory,
			scheme.Scheme,
			scheme.Scheme,
			json.SerializerOptions{
				Yaml:   true,
				Pretty: false,
				Strict: true,
			}),
		p.gvk.GroupVersion(),
	)
	buf := bytes.Buffer{}
	if err := encoder.Encode(p.expected, &buf); err != nil {
		return nil, fmt.Errorf("failed to encode patch: %w", err)
	}
	return buf.Bytes(), nil
}

type GVKParser interface {
	Type(gvk schema.GroupVersionKind) *typed.ParseableType
}

func New(cfg *rest.Config, opts client.Options) (Client, error) {
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}

	doc, err := dc.OpenAPISchema()
	if err != nil {
		return nil, err
	}

	models, err := proto.NewOpenAPIData(doc)
	if err != nil {
		return nil, err
	}

	parser, err := managedfields.NewGVKParser(models, false)
	if err != nil {
		return nil, err
	}

	c, err := client.NewWithWatch(cfg, opts)
	if err != nil {
		return nil, err
	}

	return &applier{
		WithWatch: c,
		parser:    parser,
	}, nil
}

func newObject(x client.Object) client.Object {
	if x == nil {
		return nil
	}
	res := reflect.ValueOf(x).Elem()
	n := reflect.New(res.Type())
	return n.Interface().(client.Object)
}
