/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package proto

import (
	"fmt"
	"sort"

	"k8s.io/kube-openapi/pkg/util/proto"
)

func newSchemaError(path *proto.Path, format string, a ...interface{}) error {
	err := fmt.Sprintf(format, a...)
	if path.Len() == 0 {
		return fmt.Errorf("SchemaError: %v", err)
	}
	return fmt.Errorf("SchemaError(%v): %v", path, err)
}

// Definitions is an implementation of `Models`. It looks for
// models in an openapi Schema.
type Definitions struct {
	models map[string]proto.Schema
}

var _ proto.Models = &Definitions{}

// LookupModel is public through the interface of Models. It
// returns a visitable schema from the given model name.
func (d *Definitions) LookupModel(model string) proto.Schema {
	return d.models[model]
}

func (d *Definitions) ListModels() []string {
	models := []string{}

	for model := range d.models {
		models = append(models, model)
	}

	sort.Strings(models)
	return models
}

type Ref struct {
	proto.BaseSchema

	reference   string
	definitions *Definitions
}

var _ proto.Reference = &Ref{}

func (r *Ref) Reference() string {
	return r.reference
}

func (r *Ref) SubSchema() proto.Schema {
	return r.definitions.models[r.reference]
}

func (r *Ref) Accept(v proto.SchemaVisitor) {
	v.VisitReference(r)
}

func (r *Ref) GetName() string {
	return fmt.Sprintf("Reference to %q", r.reference)
}
