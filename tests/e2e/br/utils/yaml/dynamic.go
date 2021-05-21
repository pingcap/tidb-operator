package yaml

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
)

type Interface interface {
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
