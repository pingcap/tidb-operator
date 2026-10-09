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

package validation

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const backupScheduleCRD = "crd/br.pingcap.com_backupschedules.yaml"

func TestBackupSchedule(t *testing.T) {
	cases := transferBackupScheduleCases(t, ClusterReference(), "spec", "cluster")
	cases = append(cases,
		Case{
			desc:     "cluster is required",
			isCreate: true,
			current:  backupScheduleWithoutCluster(),
			wantErrs: []string{
				`spec.cluster: Required value`,
			},
		},
		Case{
			desc:     "valid schedule",
			isCreate: true,
			current:  basicBackupSchedule(),
		},
		Case{
			desc:     "empty schedule",
			isCreate: true,
			current:  backupScheduleWithField(t, "", "spec", "schedule"),
			wantErrs: []string{`spec.schedule: Invalid value: "": spec.schedule in body should be at least 1 chars long`},
		},
		Case{
			desc:     "negative maxBackups",
			isCreate: true,
			current:  backupScheduleWithField(t, int64(-1), "spec", "maxBackups"),
			wantErrs: []string{`spec.maxBackups: Invalid value: -1: spec.maxBackups in body should be greater than or equal to 0`},
		},
		Case{
			desc:     "matching nested cluster",
			isCreate: true,
			current:  backupScheduleWithField(t, map[string]any{"cluster": "example"}, "spec", "backupTemplate", "br"),
		},
		Case{
			desc:     "empty nested cluster namespace",
			isCreate: true,
			current: backupScheduleWithField(t, map[string]any{
				"cluster":          "example",
				"clusterNamespace": "",
			}, "spec", "backupTemplate", "br"),
		},
		Case{
			desc:     "nested cluster must match authoritative target",
			isCreate: true,
			current:  backupScheduleWithField(t, map[string]any{"cluster": "other"}, "spec", "backupTemplate", "br"),
			wantErrs: []string{`spec.backupTemplate.br.cluster: Invalid value: "object": backupTemplate.br.cluster must equal cluster.name`},
		},
		Case{
			desc:     "nested same namespace must still be empty",
			isCreate: true,
			current: backupScheduleWithField(t, map[string]any{
				"cluster":          "example",
				"clusterNamespace": "backups",
			}, "spec", "backupTemplate", "br"),
			wantErrs: []string{`spec.backupTemplate.br.clusterNamespace: Invalid value: "object": backupTemplate.br.clusterNamespace must be empty`},
		},
		storedBackupScheduleMigrationCase(),
		storedBackupScheduleMigrationWithNestedTargetCase(t),
		storedBackupScheduleMigrationWithConflictingTargetCase(t),
		storedBackupScheduleMigrationWithNamespaceCase(t),
		storedBackupScheduleWithoutMigrationCase(t),
	)

	Validate(t, backupScheduleCRD, cases)
}

func TestBackupScheduleConditionsAreListMap(t *testing.T) {
	schema := structuralSchemaFromCRD(t, backupScheduleCRD, "v1alpha1")
	status, ok := schema.Properties["status"]
	require.True(t, ok)
	conditions, ok := status.Properties["conditions"]
	require.True(t, ok)
	require.NotNil(t, conditions.XListType)
	assert.Equal(t, "map", *conditions.XListType)
	assert.Equal(t, []string{"type"}, conditions.XListMapKeys)
	_, ok = status.Properties["lastScheduleTime"]
	assert.True(t, ok)
}

func TestBackupScheduleDiscoveryFields(t *testing.T) {
	data, err := os.ReadFile(backupScheduleCRD)
	require.NoError(t, err)

	var crd apiextensionsv1.CustomResourceDefinition
	require.NoError(t, yaml.Unmarshal(data, &crd))
	require.Len(t, crd.Spec.Versions, 1)
	version := crd.Spec.Versions[0]
	assert.Equal(t, []apiextensionsv1.SelectableField{{JSONPath: ".spec.cluster.name"}}, version.SelectableFields)
	require.NotEmpty(t, version.AdditionalPrinterColumns)
	assert.Equal(t, "Cluster", version.AdditionalPrinterColumns[0].Name)
	assert.Equal(t, ".spec.cluster.name", version.AdditionalPrinterColumns[0].JSONPath)
	assert.Equal(t, "string", version.AdditionalPrinterColumns[0].Type)
}

func storedBackupScheduleMigrationCase() Case {
	return Case{
		desc:    "stored object can add required cluster",
		current: basicBackupSchedule(),
		old:     backupScheduleWithoutCluster(),
	}
}

func storedBackupScheduleMigrationWithNestedTargetCase(t *testing.T) Case {
	old := backupScheduleWithField(t, map[string]any{
		"cluster":          "example",
		"clusterNamespace": "legacy-namespace",
	}, "spec", "backupTemplate", "br")
	unstructured.RemoveNestedField(old, "spec", "cluster")
	current := backupScheduleWithField(t, map[string]any{
		"cluster": "example",
	}, "spec", "backupTemplate", "br")

	return Case{
		desc:    "stored object can migrate target fields together",
		current: current,
		old:     old,
	}
}

func storedBackupScheduleMigrationWithConflictingTargetCase(t *testing.T) Case {
	old := backupScheduleWithField(t, map[string]any{
		"cluster": "legacy",
	}, "spec", "backupTemplate", "br")
	unstructured.RemoveNestedField(old, "spec", "cluster")
	current := backupScheduleWithField(t, map[string]any{
		"cluster": "legacy",
	}, "spec", "backupTemplate", "br")

	return Case{
		desc:    "stored object must align nested cluster during migration",
		current: current,
		old:     old,
		wantErrs: []string{
			`spec.backupTemplate.br.cluster: Invalid value: "object": backupTemplate.br.cluster must equal cluster.name`,
		},
	}
}

func storedBackupScheduleMigrationWithNamespaceCase(t *testing.T) Case {
	old := backupScheduleWithField(t, map[string]any{
		"cluster":          "example",
		"clusterNamespace": "legacy-namespace",
	}, "spec", "backupTemplate", "br")
	unstructured.RemoveNestedField(old, "spec", "cluster")
	current := backupScheduleWithField(t, map[string]any{
		"cluster":          "example",
		"clusterNamespace": "legacy-namespace",
	}, "spec", "backupTemplate", "br")

	return Case{
		desc:    "stored object must clear nested namespace during migration",
		current: current,
		old:     old,
		wantErrs: []string{
			`spec.backupTemplate.br.clusterNamespace: Invalid value: "object": backupTemplate.br.clusterNamespace must be empty`,
		},
	}
}

func storedBackupScheduleWithoutMigrationCase(t *testing.T) Case {
	old := backupScheduleWithoutCluster()
	current := backupScheduleWithoutCluster()
	require.NoError(t, unstructured.SetNestedField(current, "0 1 * * *", "spec", "schedule"))

	return Case{
		desc:    "stored object must add required cluster before other updates",
		current: current,
		old:     old,
		wantErrs: []string{
			`spec.cluster: Required value`,
		},
	}
}

func transferBackupScheduleCases(t *testing.T, cases []Case, fields ...string) []Case {
	for i := range cases {
		c := &cases[i]
		c.current = Patch(t, c.mode, basicBackupSchedule(), c.current, fields...)

		if c.isCreate {
			c.old = nil
			continue
		}

		c.old = Patch(t, c.mode, basicBackupSchedule(), c.old, fields...)
	}

	return cases
}

func backupScheduleWithField(t *testing.T, value any, fields ...string) map[string]any {
	obj := basicBackupSchedule()
	require.NoError(t, unstructured.SetNestedField(obj, value, fields...))
	return obj
}

func backupScheduleWithoutCluster() map[string]any {
	obj := basicBackupSchedule()
	unstructured.RemoveNestedField(obj, "spec", "cluster")
	return obj
}

func basicBackupSchedule() map[string]any {
	data := []byte(`
apiVersion: br.pingcap.com/v1alpha1
kind: BackupSchedule
metadata:
  name: hourly
  namespace: backups
spec:
  cluster:
    name: example
  schedule: "0 * * * *"
  backupTemplate:
    s3:
      provider: aws
      bucket: backups
`)
	obj := map[string]any{}
	if err := yaml.Unmarshal(data, &obj); err != nil {
		panic(err)
	}
	return obj
}
