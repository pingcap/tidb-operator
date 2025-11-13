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

package common

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/pdapi/v1"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	pdv1 "github.com/pingcap/tidb-operator/v2/pkg/timanager/apis/pd/v1"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func newFakeStore(offline bool, cond *metav1.Condition) *v1alpha1.TiKV {
	return fake.FakeObj("aaa", func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
		obj.Spec.Offline = ptr.To(offline)
		obj.Status.Conditions = []metav1.Condition{}
		if cond != nil {
			obj.Status.Conditions = append(obj.Status.Conditions, *cond)
		}
		return obj
	})
}

func TestTaskOfflineStore(t *testing.T) {
	cases := []struct {
		desc          string
		obj           *v1alpha1.TiKV
		setupPDClient func(pc *pdapi.MockPDClient)
		storeID       string
		state         pdv1.NodeState
		hasErr        bool
		isWaitErr     bool
	}{
		{
			desc:   "spec.offline == false, empty store id",
			obj:    newFakeStore(false, nil),
			hasErr: false,
		},
		{
			desc:      "spec.offline == true, empty store id",
			obj:       newFakeStore(true, nil),
			hasErr:    true,
			isWaitErr: true,
		},
		{
			desc:    "spec.offline == false, state == Preparing",
			obj:     newFakeStore(false, nil),
			storeID: "aaa",
			state:   pdv1.NodeStatePreparing,
		},
		{
			desc:    "spec.offline == false, state == Serving",
			obj:     newFakeStore(false, nil),
			storeID: "aaa",
			state:   pdv1.NodeStateServing,
		},
		{
			desc: "spec.offline == true, state == Serving",
			obj:  newFakeStore(true, nil),
			setupPDClient: func(pc *pdapi.MockPDClient) {
				pc.EXPECT().DeleteStore(context.Background(), "aaa").Return(nil)
			},
			storeID:   "aaa",
			state:     pdv1.NodeStateServing,
			hasErr:    true,
			isWaitErr: true,
		},
		{
			desc: "spec.offline == true, state == Serving, fail to delete store",
			obj:  newFakeStore(true, nil),
			setupPDClient: func(pc *pdapi.MockPDClient) {
				pc.EXPECT().DeleteStore(context.Background(), "aaa").Return(errors.New("unexpected"))
			},
			storeID: "aaa",
			state:   pdv1.NodeStateServing,
			hasErr:  true,
		},
		{
			desc: "spec.offline == false, state == Removing",
			obj:  newFakeStore(false, nil),
			setupPDClient: func(pc *pdapi.MockPDClient) {
				pc.EXPECT().CancelDeleteStore(context.Background(), "aaa").Return(nil)
			},
			storeID:   "aaa",
			state:     pdv1.NodeStateRemoving,
			hasErr:    true,
			isWaitErr: true,
		},
		{
			desc: "spec.offline == false, state == Removing, fail to cancel",
			obj:  newFakeStore(false, nil),
			setupPDClient: func(pc *pdapi.MockPDClient) {
				pc.EXPECT().CancelDeleteStore(context.Background(), "aaa").Return(errors.New("unexpected"))
			},
			storeID: "aaa",
			state:   pdv1.NodeStateRemoving,
			hasErr:  true,
		},
		{
			desc:      "spec.offline == true, state == Removing",
			obj:       newFakeStore(true, nil),
			storeID:   "aaa",
			state:     pdv1.NodeStateRemoving,
			hasErr:    true,
			isWaitErr: true,
		},
		{
			desc:    "spec.offline == true, state == Removed",
			obj:     newFakeStore(true, nil),
			storeID: "aaa",
			state:   pdv1.NodeStateRemoved,
		},
		{
			desc:    "spec.offline == false, state == Removed",
			obj:     newFakeStore(false, nil),
			storeID: "aaa",
			state:   pdv1.NodeStateRemoved,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			ctx := context.Background()
			ctrl := gomock.NewController(tt)
			pc := pdapi.NewMockPDClient(ctrl)
			if c.setupPDClient != nil {
				c.setupPDClient(pc)
			}
			err := TaskOfflineStore[scope.TiKV](ctx, pc, c.obj, c.storeID, c.state)
			if c.hasErr {
				require.Error(tt, err)
				assert.Equal(tt, c.isWaitErr, task.IsWaitError(err))
			} else {
				assert.NoError(tt, err)
			}
		})
	}
}

func TestTaskInstanceConditionOffline(t *testing.T) {
	cases := []struct {
		desc       string
		obj        *v1alpha1.TiKV
		storeState pdv1.NodeState

		expectedStatusChanged bool
		expectedStatus        task.Status
		expectedObj           *v1alpha1.TiKV
	}{
		{
			desc:           "spec.offline == false, empty state",
			obj:            newFakeStore(false, nil),
			expectedStatus: task.SComplete,
			expectedObj:    newFakeStore(false, nil),
		},
		{
			desc:                  "spec.offline == true, empty state",
			obj:                   newFakeStore(true, nil),
			expectedStatusChanged: true,
			expectedStatus:        task.SComplete,
			expectedObj:           newFakeStore(true, coreutil.Offlined()),
		},
		{
			desc:       "spec.offline == false, state == Preparing",
			obj:        newFakeStore(false, nil),
			storeState: pdv1.NodeStatePreparing,

			expectedStatus: task.SComplete,
			expectedObj:    newFakeStore(false, nil),
		},
		{
			desc:       "spec.offline == false, state == Preparing, previous condition exists",
			obj:        newFakeStore(false, coreutil.NotOfflined(v1alpha1.ReasonOfflineProcessing)),
			storeState: pdv1.NodeStatePreparing,

			expectedStatusChanged: true,
			expectedStatus:        task.SComplete,
			expectedObj:           newFakeStore(false, nil),
		},
		{
			desc:       "spec.offline == true, state == Serving",
			obj:        newFakeStore(true, nil),
			storeState: pdv1.NodeStateServing,

			expectedStatusChanged: true,
			expectedStatus:        task.SWait,
			expectedObj:           newFakeStore(true, coreutil.NotOfflined(v1alpha1.ReasonOfflineProcessing)),
		},
		{
			desc:       "spec.offline == false, state == Removing",
			obj:        newFakeStore(false, nil),
			storeState: pdv1.NodeStateRemoving,

			expectedStatusChanged: true,
			expectedStatus:        task.SWait,
			expectedObj:           newFakeStore(false, coreutil.NotOfflined(v1alpha1.ReasonOfflineCanceling)),
		},
		{
			desc:       "spec.offline == true, state == Removing",
			obj:        newFakeStore(true, nil),
			storeState: pdv1.NodeStateRemoving,

			expectedStatusChanged: true,
			expectedStatus:        task.SWait,
			expectedObj:           newFakeStore(true, coreutil.NotOfflined(v1alpha1.ReasonOfflineProcessing)),
		},
		{
			desc:       "spec.offline == false, state == Removed",
			obj:        newFakeStore(false, nil),
			storeState: pdv1.NodeStateRemoved,

			expectedStatusChanged: true,
			expectedStatus:        task.SComplete,
			expectedObj:           newFakeStore(false, coreutil.Offlined()),
		},
		{
			desc:       "spec.offline == true, state == Removed",
			obj:        newFakeStore(true, nil),
			storeState: pdv1.NodeStateRemoved,

			expectedStatusChanged: true,
			expectedStatus:        task.SComplete,
			expectedObj:           newFakeStore(true, coreutil.Offlined()),
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			ctrl := gomock.NewController(tt)
			state := NewMockInstanceCondOfflineUpdater[*v1alpha1.TiKV](ctrl)
			state.EXPECT().Object().Return(c.obj)
			state.EXPECT().GetStoreState().Return(c.storeState)
			if c.expectedStatusChanged {
				state.EXPECT().SetStatusChanged()
			}

			ctx := context.Background()
			res, done := task.RunTask(ctx, TaskInstanceConditionOffline[scope.TiKV](state))
			assert.Equal(tt, c.expectedStatus.String(), res.Status().String(), c.desc)
			assert.False(tt, done, c.desc)
			// ignore time
			for i := range c.obj.Status.Conditions {
				cond := &c.obj.Status.Conditions[i]
				cond.LastTransitionTime = metav1.Time{}
			}
			assert.Equal(tt, c.expectedObj, c.obj, c.desc)
		})
	}
}
