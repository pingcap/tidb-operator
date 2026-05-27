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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func TestTaskBoot(t *testing.T) {
	cases := []struct {
		desc          string
		dms           []*v1alpha1.DM
		unexpectedErr bool
		wantStatus    task.Status
		wantAnno      bool
	}{
		{
			desc:       "cluster not yet bootstrapped",
			dms:        []*v1alpha1.DM{fakeBootDM("aaa-0", true)},
			wantStatus: task.SComplete,
		},
		{
			desc:       "cluster already bootstrapped",
			dms:        []*v1alpha1.DM{fakeBootDM("aaa-0", false)},
			wantStatus: task.SComplete,
		},
		{
			desc:          "annotation removal fails",
			dms:           []*v1alpha1.DM{fakeBootDM("aaa-0", true)},
			unexpectedErr: true,
			wantStatus:    task.SFail,
			wantAnno:      true,
		},
		{
			desc:       "wait for annotated unready dm",
			dms:        []*v1alpha1.DM{fakeBootDM("aaa-0", true, func(dm *v1alpha1.DM) { dm.Status.Conditions = nil })},
			wantStatus: task.SWait,
			wantAnno:   true,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(t *testing.T) {
			t.Parallel()

			dmg := newTestDMGroup("aaa")
			s := &state{dmg: dmg, dms: c.dms}
			objs := []client.Object{dmg}
			for _, dm := range c.dms {
				objs = append(objs, dm)
			}
			fc := client.NewFakeClient(objs...)
			if c.unexpectedErr {
				fc.WithError("patch", "dms", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}

			res, done := task.RunTask(context.Background(), TaskBoot(s, fc))
			assert.Equal(t, c.wantStatus.String(), res.Status().String(), res.Message())
			assert.False(t, done)

			dm := &v1alpha1.DM{}
			require.NoError(t, fc.Get(context.Background(), client.ObjectKey{Name: "aaa-0"}, dm))
			_, ok := dm.Annotations[v1alpha1.AnnoKeyInitialClusterNum]
			assert.Equal(t, c.wantAnno, ok)
		})
	}
}

func fakeBootDM(name string, annotated bool, opts ...func(*v1alpha1.DM)) *v1alpha1.DM { //nolint:unparam
	dm := fake.FakeObj(name, func(obj *v1alpha1.DM) *v1alpha1.DM {
		obj.ResourceVersion = "1"
		obj.Status.Conditions = []metav1.Condition{{Type: v1alpha1.CondReady, Status: metav1.ConditionTrue}}
		if annotated {
			obj.Annotations = map[string]string{v1alpha1.AnnoKeyInitialClusterNum: "1"}
		}
		return obj
	})
	for _, opt := range opts {
		opt(dm)
	}
	return dm
}
