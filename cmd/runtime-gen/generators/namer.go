package generators

import (
	"strings"

	"k8s.io/gengo/v2/types"
)

type NameFunc func(t *types.Type) string

func (f NameFunc) Name(t *types.Type) string {
	return f(t)
}

func GroupToInstanceName(t *types.Type) string {
	return strings.TrimSuffix(t.Name.Name, "Group")
}
