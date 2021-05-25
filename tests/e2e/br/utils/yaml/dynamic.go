// Copyright 2021 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package yaml

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
)

// Interface is prepared for dealing with custom kubernetes resource in yaml data format.
// It is defined to avoid to import too many third-party clients.
type Interface interface {
	// Create creates object in yaml format
	Create(yaml []byte) error
}

func New(dc dynamic.Interface, restMapper meta.RESTMapper) Interface {
	return &client{
		restMapper: restMapper,
		dc:         dc,
	}
}

type client struct {
	restMapper meta.RESTMapper
	dc         dynamic.Interface
}

func (c *client) Create(yamlBytes []byte) error {
	jsonBytes, err := yaml.ToJSON(yamlBytes)
	if err != nil {
		return fmt.Errorf("can't convert yaml to json: %v, current yaml: %v", err, string(yamlBytes))
	}
	obj, gvk, err := unstructured.UnstructuredJSONScheme.Decode(jsonBytes, nil, nil)
	if err != nil {
		return err
	}

	mapping, err := c.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}

	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return fmt.Errorf("can't convert object to unstructured")
	}

	ns := u.GetNamespace()

	if _, err := c.dc.Resource(mapping.Resource).Namespace(ns).Create(u, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}
