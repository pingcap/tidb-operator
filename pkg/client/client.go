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

	"github.com/go-logr/logr"
	openapi_v3 "github.com/google/gnostic-models/openapiv3"
	"google.golang.org/protobuf/proto"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	serializerjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/managedfields"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/openapi"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"
	"sigs.k8s.io/structured-merge-diff/v4/typed"

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

func (r ApplyResult) String() string {
	switch r {
	case ApplyResultUpdated:
		return "updated"
	case ApplyResultUnchanged:
		return "unchanged"
	case ApplyResultCreated:
		return "created"
	}

	return "unknown"
}

type applier struct {
	client.WithWatch
	parser GVKParser
}

func (p *applier) Apply(ctx context.Context, obj client.Object) error {
	logger := logr.FromContextOrDiscard(ctx)
	res, err := p.ApplyWithResult(ctx, obj)
	if err == nil {
		logger.Info("apply success", "kind", reflect.TypeOf(obj), "namespace", obj.GetNamespace(), "name", obj.GetName(), "result", res)
	}
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

	expected, err := convertToUnstructured(gvks[0], obj)
	if err != nil {
		return ApplyResultUnchanged, err
	}

	hasCreated := true
	if err := p.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
		if !errors.IsNotFound(err) {
			return ApplyResultUnchanged, err
		}

		hasCreated = false
	}

	if hasCreated {
		current, err := convertToUnstructured(gvks[0], obj)
		if err != nil {
			return ApplyResultUnchanged, err
		}
		lastApplied, err := p.Extract(current, DefaultFieldManager, gvks[0], "")
		if err != nil {
			return ApplyResultUnchanged, fmt.Errorf("cannot extract last applied patch: %w", err)
		}

		if equality.Semantic.DeepDerivative(expected.Object, lastApplied) {
			return ApplyResultUnchanged, nil
		}
	}

	if err := p.Patch(ctx, obj, &applyPatch{
		expected: expected,
		gvk:      gvks[0],
	}, &client.PatchOptions{
		FieldManager: DefaultFieldManager,
		Force:        ptr.To(true),
	}); err != nil {
		return ApplyResultUnchanged, fmt.Errorf("cannot apply patch: %w", err)
	}

	if hasCreated {
		return ApplyResultUpdated, nil
	}

	return ApplyResultCreated, nil
}

func (p *applier) Extract(current *unstructured.Unstructured, fieldManager string, gvk schema.GroupVersionKind, subresource string) (map[string]any, error) {
	tpd := p.parser.Type(gvk)
	if tpd == nil {
		return nil, fmt.Errorf("can't find specified type: %s", gvk)
	}
	typedObj, err := tpd.FromUnstructured(current.Object)
	if err != nil {
		return nil, fmt.Errorf("error converting obj to typed: %w", err)
	}

	var fe *metav1.ManagedFieldsEntry
	objManagedFields := current.GetManagedFields()
	for i := range objManagedFields {
		mf := &objManagedFields[i]
		if mf.Manager == fieldManager && mf.Operation == metav1.ManagedFieldsOperationApply && mf.Subresource == subresource {
			fe = mf
		}
	}

	if fe == nil {
		return nil, fmt.Errorf("cannot find managed fields: %w", err)
	}

	fieldset := &fieldpath.Set{}
	if err = fieldset.FromJSON(bytes.NewReader(fe.FieldsV1.Raw)); err != nil {
		return nil, fmt.Errorf("error marshaling FieldsV1 to JSON: %w", err)
	}

	u := typedObj.ExtractItems(fieldset.Leaves()).AsValue().Unstructured()
	m, ok := u.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unable to convert managed fields for %s to unstructured, expected map, got %T", fieldManager, u)
	}

	m["apiVersion"] = gvk.GroupVersion().String()
	m["kind"] = gvk.Kind
	mu, ok := m["metadata"]
	if !ok {
		mu = map[string]any{}
	}
	metadata, ok := mu.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unable to convert metadata to map[string]any, expected map, got %T", mu)
	}
	metadata["name"] = current.GetName()
	if current.GetNamespace() != "" {
		metadata["namespace"] = current.GetNamespace()
	}
	m["metadata"] = metadata

	return m, nil
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
		serializerjson.NewSerializerWithOptions(
			serializerjson.DefaultMetaFactory,
			scheme.Scheme,
			scheme.Scheme,
			serializerjson.SerializerOptions{
				Yaml:   false,
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
