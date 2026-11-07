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
	pdapi "github.com/pingcap/tidb-operator/v2/pkg/pdapi/v1"
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
	for _, tc := range []struct {
		name, instanceName string
		readyAPI, deleting bool
		expected           task.Status
	}{
		{name: "ready API disabled", instanceName: "invalid", expected: task.SComplete},
		{name: "ready API enabled", instanceName: "invalid", readyAPI: true, expected: task.SFail},
		{name: "deleting invalid member", instanceName: "invalid", readyAPI: true, deleting: true, expected: task.SComplete},
		{name: "valid member still checked", instanceName: "pd-abc", readyAPI: true, expected: task.SComplete},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cluster := &v1alpha1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "test"}}
			cluster.Spec.FeatureGates = []metav1alpha1.FeatureGate{{Name: metav1alpha1.UsePDReadyAPI, Enabled: ptr.To(tc.readyAPI)}}
			pd := &v1alpha1.PD{ObjectMeta: metav1.ObjectMeta{Name: tc.instanceName, Namespace: "test"}}
			pd.Spec.Version = "v8.5.2"
			if tc.deleting {
				pd.DeletionTimestamp = ptr.To(metav1.Now())
			}
			st := NewState(client.ObjectKeyFromObject(pd))
			st.SetObject(pd)
			st.SetCluster(cluster)
			state := &ReconcileContext{State: st}
			controller := gomock.NewController(t)
			pc := pdm.NewMockPDClient(controller)
			members := pdm.NewMockMemberCache[pdv1.Member](controller)
			pc.EXPECT().MembersSynced().Return(true)
			pc.EXPECT().Members().Return(members)
			members.EXPECT().Get(pd.Name).Return(&pdv1.Member{ID: "123", Health: true}, nil)
			underlay := pdapi.NewMockPDClient(controller)
			if tc.instanceName == "pd-abc" {
				pc.EXPECT().Underlay().Return(underlay)
				underlay.EXPECT().GetMemberReady(gomock.Any(), getPDURL(cluster, pd), pd.Spec.Version).Return(true, nil)
			}
			result, _ := task.RunTask(t.Context(), TaskContextInfoFromPD(state, &readyAPIClientManager{pdClient: pc}))
			require.Equal(t, tc.expected, result.Status())
			if tc.deleting {
				require.Equal(t, "123", state.MemberID)
				pc.EXPECT().Underlay().Return(underlay)
				underlay.EXPECT().DeleteMember(gomock.Any(), pd.Name).Return(nil)
				result, _ = task.RunTask(t.Context(), TaskDeleteMember(state))
				require.Equal(t, task.SComplete, result.Status())
			}
		})
	}
}
