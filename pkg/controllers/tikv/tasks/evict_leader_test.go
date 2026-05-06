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

package tasks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	pdapi "github.com/pingcap/tidb-operator/v2/pkg/pdapi/v1"
	stateutil "github.com/pingcap/tidb-operator/v2/pkg/state"
	pdv1 "github.com/pingcap/tidb-operator/v2/pkg/timanager/apis/pd/v1"
	pdm "github.com/pingcap/tidb-operator/v2/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func TestTaskEvictLeader(t *testing.T) {
	cases := []struct {
		desc           string
		state          *ReconcileContext
		expectBegin    bool
		expectEnd      bool
		expectEvicting bool
		expectedStatus task.Status
	}{
		{
			desc: "begin evict leader when requested",
			state: &ReconcileContext{
				State: &state{
					tikv: fake.FakeObj[v1alpha1.TiKV]("aaa-xxx"),
					pod:  fake.FakeObj[corev1.Pod]("aaa-tikv-xxx"),
				},
				ShouldEvictLeader: true,
				PDSynced:          true,
				Store: &pdv1.Store{
					ID: "1",
				},
			},
			expectBegin:    true,
			expectEvicting: true,
			expectedStatus: task.SComplete,
		},
		{
			desc: "end evict leader when annotation is absent",
			state: &ReconcileContext{
				State: &state{
					tikv: fake.FakeObj[v1alpha1.TiKV]("aaa-xxx"),
					pod:  fake.FakeObj[corev1.Pod]("aaa-tikv-xxx"),
				},
				PDSynced:       true,
				LeaderEvicting: true,
				Store: &pdv1.Store{
					ID: "1",
				},
			},
			expectEnd:      true,
			expectEvicting: false,
			expectedStatus: task.SComplete,
		},
		{
			desc: "sync leaders evicted condition when store is absent",
			state: &ReconcileContext{
				State: &state{
					tikv: fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
						obj.Generation = 3
						return obj
					}),
					pod: fake.FakeObj[corev1.Pod]("aaa-tikv-xxx"),
				},
				PDSynced:       true,
				LeaderEvicting: true,
				Store:          nil,
			},
			expectEvicting: true,
			expectedStatus: task.SComplete,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			ctrl := gomock.NewController(tt)
			mockPDClient := pdm.NewMockPDClient(ctrl)
			mockUnderlay := pdapi.NewMockPDClient(ctrl)
			mockPDClient.EXPECT().Underlay().Return(mockUnderlay).AnyTimes()
			if c.expectBegin {
				mockUnderlay.EXPECT().BeginEvictLeader(gomock.Any(), "1").Return(nil)
			}
			if c.expectEnd {
				mockUnderlay.EXPECT().EndEvictLeader(gomock.Any(), "1").Return(nil)
			}

			s := c.state.State.(*state)
			s.IPDClient = &stubPDClientState{client: mockPDClient}

			res, done := task.RunTask(context.Background(), TaskEvictLeader(c.state, nil))
			assert.Equal(tt, c.expectedStatus, res.Status())
			assert.False(tt, done)
			assert.Equal(tt, c.expectEvicting, c.state.LeaderEvicting)
			if c.state.Store == nil {
				cond := findCondition(c.state.TiKV().Status.Conditions, v1alpha1.TiKVCondLeadersEvicted)
				require.NotNil(tt, cond)
				assert.Equal(tt, metav1.ConditionTrue, cond.Status)
				assert.Equal(tt, v1alpha1.ReasonStoreNotExist, cond.Reason)
			}
		})
	}
}

type stubPDClientState struct {
	client pdm.PDClient
}

func (s *stubPDClientState) GetPDClient(pdm.PDClientManager) (pdm.PDClient, bool) {
	return s.client, true
}

var _ stateutil.IPDClient = (*stubPDClientState)(nil)

type stubPDClientUnavailableState struct{}

func (s *stubPDClientUnavailableState) GetPDClient(pdm.PDClientManager) (pdm.PDClient, bool) {
	return nil, false
}

var _ stateutil.IPDClient = (*stubPDClientUnavailableState)(nil)

func findCondition(conds []metav1.Condition, typ string) *metav1.Condition {
	for i := range conds {
		if conds[i].Type == typ {
			return &conds[i]
		}
	}
	return nil
}
