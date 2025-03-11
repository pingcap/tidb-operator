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
	"io"
	"reflect"

	"github.com/go-json-experiment/json"
	jsonv1 "github.com/go-json-experiment/json/v1"
	openapi_v3 "github.com/google/gnostic-models/openapiv3"
	"google.golang.org/protobuf/proto"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/managedfields"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/openapi"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/structured-merge-diff/v4/typed"
	"sigs.k8s.io/yaml"

	"github.com/pingcap/tidb-operator/pkg/scheme"
	"github.com/pingcap/tidb-operator/pkg/utils/kubefeat"
	forkedproto "github.com/pingcap/tidb-operator/third_party/kube-openapi/pkg/util/proto"
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
		patchEncoder{},
		p.gvk.GroupVersion(),
	)
	buf := bytes.Buffer{}
	if err := encoder.Encode(p.expected, &buf); err != nil {
		return nil, fmt.Errorf("failed to encode patch: %w", err)
	}
	return buf.Bytes(), nil
}

type patchEncoder struct{}

func (patchEncoder) Encode(obj kuberuntime.Object, w io.Writer) error {
	json, err := json.Marshal(obj, jsonv1.OmitEmptyWithLegacyDefinition(true), json.OmitZeroStructFields(true))
	if err != nil {
		return err
	}
	data, err := yaml.JSONToYAML(json)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func (patchEncoder) Identifier() kuberuntime.Identifier {
	return kuberuntime.Identifier("custom")
}

type GVKParser interface {
	Type(gvk schema.GroupVersionKind) *typed.ParseableType
}

type gvkParser struct {
	parsers map[schema.GroupVersion]GVKParser
}

func (p *gvkParser) Type(gvk schema.GroupVersionKind) *typed.ParseableType {
	parser, ok := p.parsers[gvk.GroupVersion()]
	if !ok {
		return nil
	}
	return parser.Type(gvk)
}

func gvToAPIPath(gv schema.GroupVersion) string {
	var resourcePath string
	if gv.Group == "" {
		resourcePath = fmt.Sprintf("api/%s", gv.Version)
	} else {
		resourcePath = fmt.Sprintf("apis/%s/%s", gv.Group, gv.Version)
	}
	return resourcePath
}

func New(cfg *rest.Config, opts client.Options) (Client, error) {
	kubefeat.MustInitFeatureGates(cfg)

	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("cannot new discovery client: %w", err)
	}

	oc := dc.OpenAPIV3()
	paths, err := oc.Paths()
	if err != nil {
		return nil, err
	}
	gvs := scheme.GroupVersions()

	parser, err := NewGVKParser(gvs, paths)
	if err != nil {
		return nil, fmt.Errorf("cannot new gvk parser: %w", err)
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

func NewGVKParser(gvs []schema.GroupVersion, paths map[string]openapi.GroupVersion) (GVKParser, error) {
	parser := &gvkParser{
		parsers: map[schema.GroupVersion]GVKParser{},
	}
	for _, gv := range gvs {
		path := gvToAPIPath(gv)
		gvc, ok := paths[path]
		if !ok {
			return nil, fmt.Errorf("cannot find openapi doc of gv %v", gv.String())
		}
		if err := parser.addGroupVersion(gv, gvc); err != nil {
			return nil, err
		}
	}

	return parser, nil
}

func (p *gvkParser) addGroupVersion(gv schema.GroupVersion, gvc openapi.GroupVersion) error {
	bs, err := gvc.Schema(openapi.ContentTypeOpenAPIV3PB)
	if err != nil {
		return err
	}
	var doc openapi_v3.Document
	if err2 := proto.Unmarshal(bs, &doc); err2 != nil {
		return err2
	}
	models, err := forkedproto.NewOpenAPIV3Data(&doc)
	if err != nil {
		return err
	}
	parser, err := managedfields.NewGVKParser(models, false)
	if err != nil {
		return err
	}
	p.parsers[gv] = parser
	return nil
}

func newObject(x client.Object) client.Object {
	if x == nil {
		return nil
	}
	res := reflect.ValueOf(x).Elem()
	n := reflect.New(res.Type())
	return n.Interface().(client.Object)
}
