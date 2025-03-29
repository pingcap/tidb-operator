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
