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
	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	pdapi "github.com/pingcap/tidb-operator/v2/pkg/pdapi/v1"
	pdv1 "github.com/pingcap/tidb-operator/v2/pkg/timanager/apis/pd/v1"
	pdm "github.com/pingcap/tidb-operator/v2/pkg/timanager/pd"
	task "github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"testing"
)

type staleStorePDClientState struct{ c pdm.PDClient }

func (s *staleStorePDClientState) GetPDClient(pdm.PDClientManager) (pdm.PDClient, bool) {
	return s.c, true
}

func TestContextRejectsInvalidStore(t *testing.T) {
	for _, invalid := range []bool{true, false} {
		t.Run(map[bool]string{true: "invalid", false: "valid"}[invalid], func(t *testing.T) {
			ctrl := gomock.NewController(t)
			pc := pdm.NewMockPDClient(ctrl)
			cache := pdm.NewMockStoreCache[pdv1.Store, *pdv1.Store](ctrl)
			pc.EXPECT().Stores().Return(cache)
			cache.EXPECT().Get(gomock.Any()).Return(&pdv1.Store{Invalid: invalid, ID: "1", NodeState: pdv1.NodeStateRemoved}, nil)
			st := NewState(types.NamespacedName{Namespace: "test", Name: "instance-abc"})
			st.SetObject(&v1alpha1.TiKV{ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "instance-abc"}})
			st.SetCluster(&v1alpha1.Cluster{ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "cluster"}})
			st.(*state).IPDClient = &staleStorePDClientState{c: pc}
			rtx := &ReconcileContext{State: st}
			underlay := pdapi.NewMockPDClient(ctrl)
			pc.EXPECT().Underlay().Return(underlay).AnyTimes()
			underlay.EXPECT().GetEvictLeaderScheduler(gomock.Any(), "1").Return("", nil).AnyTimes()
			continued := false
			result, _ := task.RunTask(context.Background(), task.Block(TaskContextInfoFromPD(rtx, nil), task.NameTaskFunc("unsafe follow-up", func(context.Context) task.Result { continued = true; return task.Complete().With("continued") })))
			if invalid {
				require.Equal(t, task.SFail, result.Status())
				require.False(t, continued, "invalid store must stop subsequent tasks")
				require.False(t, rtx.PDSynced)
				require.Nil(t, rtx.Store)
				require.Empty(t, rtx.GetStoreState())
			} else {
				require.Equal(t, task.SComplete, result.Status())
				require.True(t, continued)
				require.True(t, rtx.PDSynced)
				require.NotNil(t, rtx.Store)
				require.Equal(t, pdv1.NodeStateRemoved, rtx.GetStoreState())
			}
		})
	}
}
