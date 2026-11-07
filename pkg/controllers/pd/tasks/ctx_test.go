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
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	pdv1 "github.com/pingcap/tidb-operator/v2/pkg/timanager/apis/pd/v1"
	pdm "github.com/pingcap/tidb-operator/v2/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

type readyAPIClientManager struct {
	pdm.PDClientManager
	pdClient pdm.PDClient
}

func (m *readyAPIClientManager) Get(string) (pdm.PDClient, bool) { return m.pdClient, true }

func TestContextInfoValidatesNameBeforeReadyAPI(t *testing.T) {
	for _, readyAPI := range []bool{false, true} {
		name := "ready API disabled"
		if readyAPI {
			name = "ready API enabled"
		}
		t.Run(name, func(t *testing.T) {
			cluster := &v1alpha1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "test"}}
			cluster.Spec.FeatureGates = []metav1alpha1.FeatureGate{{Name: metav1alpha1.UsePDReadyAPI, Enabled: ptr.To(readyAPI)}}
			pd := &v1alpha1.PD{ObjectMeta: metav1.ObjectMeta{Name: "invalid", Namespace: "test"}}
			pd.Spec.Version = "v8.5.2"
			st := NewState(client.ObjectKeyFromObject(pd))
			st.SetObject(pd)
			st.SetCluster(cluster)
			state := &ReconcileContext{State: st}
			controller := gomock.NewController(t)
			pc := pdm.NewMockPDClient(controller)
			members := pdm.NewMockMemberCache[pdv1.Member](controller)
			pc.EXPECT().MembersSynced().Return(true)
			pc.EXPECT().Members().Return(members)
			members.EXPECT().Get(pd.Name).Return(&pdv1.Member{Health: true}, nil)
			// An invalid name must never reach Underlay().GetMemberReady.
			result, _ := task.RunTask(t.Context(), TaskContextInfoFromPD(state, &readyAPIClientManager{pdClient: pc}))
			expected := task.SComplete
			if readyAPI {
				expected = task.SFail
			}
			require.Equal(t, expected, result.Status())
		})
	}
}
