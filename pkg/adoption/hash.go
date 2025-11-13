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

package adoption

import (
	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/reloadable"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/hasher"
)

type HashableTiDB struct {
	Features []metav1alpha1.Feature
	Template *v1alpha1.TiDBTemplate
}

func hashTemplate(features []metav1alpha1.Feature, template *v1alpha1.TiDBTemplate) string {
	template = reloadable.ConvertTiDBTemplate(template)
	// Ignore template labels and annotations
	template.Annotations = nil
	template.Labels = nil
	// Ignore template mode and keyspace
	// Now can only adopt standby instances
	template.Spec.Mode = ""
	template.Spec.Keyspace = ""
	// Ignore update strategy
	// Always to ensure that config is same
	template.Spec.UpdateStrategy.Config = ""

	hashable := &HashableTiDB{
		Features: features,
		Template: template,
	}

	return hasher.Hash(hashable)
}

func hashTiDBGroup(g *v1alpha1.TiDBGroup) string {
	return hashTemplate(g.Spec.Features, &g.Spec.Template)
}

func hashTiDB(db *v1alpha1.TiDB) string {
	return hashTemplate(db.Spec.Features, reloadable.TemplateFromTiDB(db))
}
