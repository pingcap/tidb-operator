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

package revision

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/third_party/kubernetes/pkg/controller/history"
)

// FakeHistoryClient is a fake implementation of Interface that is useful for testing.
type FakeHistoryClient struct {
	Revisions  []*appsv1.ControllerRevision
	CreateFunc func(parent client.Object, revision *appsv1.ControllerRevision, collisionCount *int32) (*appsv1.ControllerRevision, error)
}

func (f *FakeHistoryClient) CreateControllerRevision(parent client.Object, revision *appsv1.ControllerRevision, collisionCount *int32) (*appsv1.ControllerRevision, error) {
	if f.CreateFunc != nil {
		rev, err := f.CreateFunc(parent, revision, collisionCount)
		if err != nil {
			f.Revisions = append(f.Revisions, rev)
		}
		return rev, err
	}
	return nil, nil
}

func (f *FakeHistoryClient) DeleteControllerRevision(revision *appsv1.ControllerRevision) error {
	for i, r := range f.Revisions {
		if r.Name == revision.Name {
			f.Revisions = append(f.Revisions[:i], f.Revisions[i+1:]...)
			return nil
		}
	}
	return nil
}

func (f *FakeHistoryClient) ListControllerRevisions(_ client.Object, _ labels.Selector) ([]*appsv1.ControllerRevision, error) {
	return f.Revisions, nil
}

func (f *FakeHistoryClient) UpdateControllerRevision(revision *appsv1.ControllerRevision, newRevision int64) (*appsv1.ControllerRevision, error) {
	for _, r := range f.Revisions {
		if r.Name == revision.Name {
			r.Revision = newRevision
			return r, nil
		}
	}
	return nil, nil
}

var _ history.Interface = &FakeHistoryClient{}

func withRevision(currentRev, updateRev string) fake.ChangeFunc[v1alpha1.TiDB, *v1alpha1.TiDB] {
	return func(tidb *v1alpha1.TiDB) *v1alpha1.TiDB {
		tidb.Status.CurrentRevision = currentRev
		tidb.Status.UpdateRevision = updateRev
		return tidb
	}
}

func TestTruncateHistory(t *testing.T) {
	tests := []struct {
		name      string
		instances []*v1alpha1.TiDB
		revisions []*appsv1.ControllerRevision
		current   *appsv1.ControllerRevision
		update    *appsv1.ControllerRevision
		limit     *int32
		expected  []*appsv1.ControllerRevision
	}{
		{
			name:      "no revisions to truncate",
			instances: []*v1alpha1.TiDB{},
			revisions: []*appsv1.ControllerRevision{
				fake.FakeObj[appsv1.ControllerRevision]("rev1"),
				fake.FakeObj[appsv1.ControllerRevision]("rev2"),
			},
			current: fake.FakeObj[appsv1.ControllerRevision]("rev1"),
			update:  fake.FakeObj[appsv1.ControllerRevision]("rev2"),
			limit:   ptr.To[int32](2),
			expected: []*appsv1.ControllerRevision{
				fake.FakeObj[appsv1.ControllerRevision]("rev1"),
				fake.FakeObj[appsv1.ControllerRevision]("rev2"),
			},
		},
		{
			name:      "truncate one revision",
			instances: []*v1alpha1.TiDB{},
			revisions: []*appsv1.ControllerRevision{
				fake.FakeObj[appsv1.ControllerRevision]("rev1"),
				fake.FakeObj[appsv1.ControllerRevision]("rev2"),
				fake.FakeObj[appsv1.ControllerRevision]("rev3"),
				fake.FakeObj[appsv1.ControllerRevision]("rev4"),
			},
			current: fake.FakeObj[appsv1.ControllerRevision]("rev4"),
			update:  fake.FakeObj[appsv1.ControllerRevision]("rev4"),
			limit:   ptr.To[int32](2),
			expected: []*appsv1.ControllerRevision{
				fake.FakeObj[appsv1.ControllerRevision]("rev2"),
				fake.FakeObj[appsv1.ControllerRevision]("rev3"),
				fake.FakeObj[appsv1.ControllerRevision]("rev4"),
			},
		},
		{
			name:      "truncate multiple revisions",
			instances: []*v1alpha1.TiDB{},
			revisions: []*appsv1.ControllerRevision{
				fake.FakeObj[appsv1.ControllerRevision]("rev1"),
				fake.FakeObj[appsv1.ControllerRevision]("rev2"),
				fake.FakeObj[appsv1.ControllerRevision]("rev3"),
				fake.FakeObj[appsv1.ControllerRevision]("rev4"),
			},
			current: fake.FakeObj[appsv1.ControllerRevision]("rev3"),
			update:  fake.FakeObj[appsv1.ControllerRevision]("rev4"),
			limit:   ptr.To[int32](0),
			expected: []*appsv1.ControllerRevision{
				fake.FakeObj[appsv1.ControllerRevision]("rev3"),
				fake.FakeObj[appsv1.ControllerRevision]("rev4"),
			},
		},
		{
			name: "complex case",
			instances: []*v1alpha1.TiDB{
				fake.FakeObj("tidb1", withRevision("rev4", "rev4")),
				fake.FakeObj("tidb2", withRevision("rev3", "rev4")),
				fake.FakeObj("tidb3", withRevision("rev3", "rev5")),
				fake.FakeObj("tidb4", withRevision("rev4", "rev5")),
			},
			revisions: []*appsv1.ControllerRevision{
				fake.FakeObj[appsv1.ControllerRevision]("rev1"),
				fake.FakeObj[appsv1.ControllerRevision]("rev2"),
				fake.FakeObj[appsv1.ControllerRevision]("rev3"),
				fake.FakeObj[appsv1.ControllerRevision]("rev4"),
				fake.FakeObj[appsv1.ControllerRevision]("rev5"),
				fake.FakeObj[appsv1.ControllerRevision]("rev6"),
			},
			current: fake.FakeObj[appsv1.ControllerRevision]("rev4"),
			update:  fake.FakeObj[appsv1.ControllerRevision]("rev5"),
			limit:   ptr.To[int32](1),
			expected: []*appsv1.ControllerRevision{
				fake.FakeObj[appsv1.ControllerRevision]("rev3"),
				fake.FakeObj[appsv1.ControllerRevision]("rev4"),
				fake.FakeObj[appsv1.ControllerRevision]("rev5"),
				fake.FakeObj[appsv1.ControllerRevision]("rev6"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := &FakeHistoryClient{
				Revisions: tt.revisions,
			}
			err := TruncateHistory(cli, runtime.FromTiDBSlice(tt.instances), tt.revisions, tt.current.Name, tt.update.Name, tt.limit)
			require.NoError(t, err)

			remainingRevisions, err := cli.ListControllerRevisions(nil, labels.Everything())
			require.NoError(t, err)
			assert.Equal(t, len(tt.expected), len(remainingRevisions))
			m := make(map[string]struct{}, len(remainingRevisions))
			for _, r := range remainingRevisions {
				m[r.Name] = struct{}{}
			}
			for _, r := range tt.expected {
				if _, ok := m[r.Name]; !ok {
					t.Errorf("expected revision %s not found", r.Name)
				}
			}
		})
	}
}

func TestGetCurrentAndUpdate(t *testing.T) {
	pdg := &v1alpha1.PDGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: "basic",
		},
		Spec: v1alpha1.PDGroupSpec{
			Cluster: v1alpha1.ClusterReference{
				Name: "basic",
			},
			Replicas: ptr.To[int32](1),
		},
	}
	rev1, err := newRevision(pdg, "pd", nil, pdg.GVK(), 1, ptr.To[int32](0))
	require.NoError(t, err)

	pdg2 := pdg.DeepCopy()
	pdg2.Spec.Replicas = ptr.To[int32](2)
	rev2, err := newRevision(pdg2, "pd", nil, pdg2.GVK(), 2, ptr.To[int32](0))
	require.NoError(t, err)

	tests := []struct {
		name           string
		group          client.Object
		revisions      []*appsv1.ControllerRevision
		accessor       v1alpha1.ComponentAccessor
		expectedCurRev string
		expectedUpdRev string
		expectedErr    bool
	}{
		{
			name:           "no existing revisions",
			group:          pdg,
			revisions:      []*appsv1.ControllerRevision{},
			accessor:       pdg,
			expectedCurRev: "basic-pd-687bcf9d45",
			expectedUpdRev: "basic-pd-687bcf9d45",
			expectedErr:    false,
		},
		{
			name:           "match the prior revision",
			group:          pdg2,
			revisions:      []*appsv1.ControllerRevision{rev1, rev2},
			accessor:       pdg2,
			expectedCurRev: "basic-pd-5f5f578c9d",
			expectedUpdRev: "basic-pd-5f5f578c9d",
			expectedErr:    false,
		},
		{
			name:           "match an earlier revision",
			group:          pdg,
			revisions:      []*appsv1.ControllerRevision{rev1, rev2},
			accessor:       pdg,
			expectedCurRev: "basic-pd-687bcf9d45",
			expectedUpdRev: "basic-pd-687bcf9d45",
			expectedErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := &FakeHistoryClient{
				Revisions: tt.revisions,
				CreateFunc: func(_ client.Object, revision *appsv1.ControllerRevision, _ *int32) (*appsv1.ControllerRevision, error) {
					return revision, nil
				},
			}
			curRev, updRev, _, err := GetCurrentAndUpdate(context.TODO(), tt.group, "pd", nil, tt.revisions, cli, tt.accessor.CurrentRevision(), tt.accessor.CollisionCount())
			if tt.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCurRev, curRev.Name)
				assert.Equal(t, tt.expectedUpdRev, updRev.Name)
			}
		})
	}
}
