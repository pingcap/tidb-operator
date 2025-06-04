package validation

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func TestCluster(t *testing.T) {
	var cases []Case
	cases = append(cases,
		transferClusterCases(t, FeatureGates(), "spec", "featureGates")...)

	Validate(t, "crd/core.pingcap.com_clusters.yaml", cases)
}

func basicCluster() map[string]any {
	data := []byte(`
apiVersion: core.pingcap.com/v1alpha1
kind: Cluster
metadata:
  name: basic
spec: {}
`)
	obj := map[string]any{}
	if err := yaml.Unmarshal(data, &obj); err != nil {
		panic(err)
	}

	return obj
}

func transferClusterCases(t *testing.T, cases []Case, fields ...string) []Case {
	for i := range cases {
		c := &cases[i]

		current := basicCluster()
		if c.current == nil {
			unstructured.RemoveNestedField(current, fields...)
		} else {
			require.NoError(t, unstructured.SetNestedField(current, c.current, fields...))
		}

		c.current = current

		if c.isCreate {
			c.old = nil
			continue
		}

		old := basicCluster()
		if c.old == nil {
			unstructured.RemoveNestedField(old, fields...)
		} else {
			require.NoError(t, unstructured.SetNestedField(old, c.old, fields...))
		}

		c.old = old
	}

	return cases
}
