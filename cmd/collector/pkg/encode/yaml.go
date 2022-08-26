package encode

import (
	"gopkg.in/yaml.v3"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const YAML Encoding = "yaml"

type YAMLEncoder struct{}

var _ Encoder = (*YAMLEncoder)(nil)

func (e *YAMLEncoder) Encode(obj client.Object) ([]byte, error) {
	return yaml.Marshal(obj)
}

func (e *YAMLEncoder) Extension() string {
	return "yaml"
}

func NewYAMLEncoder() *YAMLEncoder {
	return &YAMLEncoder{}
}
