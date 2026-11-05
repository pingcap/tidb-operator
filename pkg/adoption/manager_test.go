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
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func TestAdoptionScope(t *testing.T) {
	for _, tc := range []struct {
		name, namespace, cluster string
		allowed                  bool
	}{
		{"same scope", "tenant", "cluster", true},
		{"other namespace", "other", "cluster", false},
		{"other cluster", "tenant", "other", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := New(logr.Discard())
			db := &v1alpha1.TiDB{ObjectMeta: metav1.ObjectMeta{Namespace: tc.namespace, Name: "standby", UID: "db"}}
			db.Spec.Cluster.Name = tc.cluster
			db.Spec.Mode = v1alpha1.TiDBModeStandBy
			group := &v1alpha1.TiDBGroup{ObjectMeta: metav1.ObjectMeta{Namespace: "tenant", Name: "normal", UID: "group"}}
			group.Spec.Cluster.Name = "cluster"
			m.Register(db)
			adopted, unlock := m.Adopt(group, 0)
			if !tc.allowed {
				require.Nil(t, adopted)
				return
			}
			require.NotNil(t, adopted)
			again, _ := m.Adopt(group, 0)
			require.Equal(t, adopted, again)
			changed := group.DeepCopy()
			changed.Spec.Cluster.Name = "other"
			stale, _ := m.Adopt(changed, 0)
			require.Nil(t, stale)
			recreated := group.DeepCopy()
			recreated.UID = "new-group"
			stale, _ = m.Adopt(recreated, 0)
			require.Nil(t, stale)
			unlock()
			adopted, unlock = m.Adopt(group, 0)
			require.NotNil(t, adopted)
			unlock()
		})
	}
}
