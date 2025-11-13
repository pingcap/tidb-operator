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
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/pdapi/v1"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func TestTaskContextObject(t *testing.T) {
	cases := []struct {
		desc          string
		state         *fakeState[v1alpha1.PD]
		objs          []client.Object
		unexpectedErr bool

		expectedResult task.Status
		expectedObj    *v1alpha1.PD
	}{
		{
			desc: "success",
			state: &fakeState[v1alpha1.PD]{
				ns:   "aaa",
				name: "aaa",
			},
			objs: []client.Object{
				fake.FakeObj("aaa", fake.SetNamespace[v1alpha1.PD]("aaa")),
			},
			expectedResult: task.SComplete,
			expectedObj:    fake.FakeObj("aaa", fake.SetNamespace[v1alpha1.PD]("aaa")),
		},
		{
			desc: "not found",
			state: &fakeState[v1alpha1.PD]{
				ns:   "aaa",
				name: "aaa",
			},
			expectedResult: task.SComplete,
		},
		{
			desc: "has unexpected error",
			state: &fakeState[v1alpha1.PD]{
				ns:   "aaa",
				name: "aaa",
			},
			objs: []client.Object{
				fake.FakeObj("aaa", fake.SetNamespace[v1alpha1.PD]("aaa")),
			},
			unexpectedErr:  true,
			expectedResult: task.SFail,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			fc := client.NewFakeClient(c.objs...)

			if c.unexpectedErr {
				fc.WithError("*", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}

			res, done := task.RunTask(context.Background(), TaskContextObject[scope.PD](c.state, fc))
			assert.Equal(tt, c.expectedResult, res.Status(), c.desc)
			assert.False(tt, done, c.desc)
			assert.Equal(tt, c.expectedObj, c.state.obj, c.desc)
		})
	}
}

func TestTaskContextCluster(t *testing.T) {
	const ns = "aaa"
	const name = "bbb"
	cases := []struct {
		desc          string
		state         *fakeObjectState[*v1alpha1.PD]
		objs          []client.Object
		unexpectedErr bool

		expectedResult task.Status
		expectedObj    *v1alpha1.Cluster
	}{
		{
			desc: "success",
			state: newFakeObjectState(fake.FakeObj(name, func(obj *v1alpha1.PD) *v1alpha1.PD {
				obj.Namespace = ns
				obj.Spec.Cluster.Name = name
				return obj
			})),
			objs: []client.Object{
				fake.FakeObj(name, fake.SetNamespace[v1alpha1.Cluster](ns)),
			},
			expectedResult: task.SComplete,
			expectedObj:    fake.FakeObj(name, fake.SetNamespace[v1alpha1.Cluster](ns)),
		},
		{
			desc: "not found",
			state: newFakeObjectState(fake.FakeObj(name, func(obj *v1alpha1.PD) *v1alpha1.PD {
				obj.Namespace = ns
				obj.Spec.Cluster.Name = name
				return obj
			})),
			expectedResult: task.SFail,
		},
		{
			desc: "has unexpected error",
			state: newFakeObjectState(fake.FakeObj(name, func(obj *v1alpha1.PD) *v1alpha1.PD {
				obj.Namespace = ns
				obj.Spec.Cluster.Name = name
				return obj
			})),
			objs: []client.Object{
				fake.FakeObj(name, fake.SetNamespace[v1alpha1.Cluster](ns)),
			},
			unexpectedErr:  true,
			expectedResult: task.SFail,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			fc := client.NewFakeClient(c.objs...)

			if c.unexpectedErr {
				fc.WithError("*", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}
			res, done := task.RunTask(context.Background(), TaskContextCluster[scope.PD](c.state, fc))
			assert.Equal(tt, c.expectedResult, res.Status(), c.desc)
			assert.False(tt, done, c.desc)
			assert.Equal(tt, c.expectedObj, c.state.cluster, c.desc)
		})
	}
}

func TestTaskContextPod(t *testing.T) {
	const name = "bbb"
	cases := []struct {
		desc          string
		state         *fakeObjectState[*v1alpha1.PD]
		objs          []client.Object
		unexpectedErr bool

		expectedResult task.Status
		expectedObj    *corev1.Pod
	}{
		{
			desc: "success",
			state: newFakeObjectState(fake.FakeObj(name, func(obj *v1alpha1.PD) *v1alpha1.PD {
				return obj
			})),
			objs: []client.Object{
				fake.FakeObj(name, fake.InstanceOwner[scope.PD, corev1.Pod](fake.FakeObj[v1alpha1.PD](name))),
			},
			expectedResult: task.SComplete,
			expectedObj:    fake.FakeObj(name, fake.InstanceOwner[scope.PD, corev1.Pod](fake.FakeObj[v1alpha1.PD](name))),
		},
		{
			desc: "not found",
			state: newFakeObjectState(fake.FakeObj(name, func(obj *v1alpha1.PD) *v1alpha1.PD {
				return obj
			})),
			objs:           []client.Object{},
			expectedResult: task.SComplete,
		},
		{
			desc: "has unexpected error",
			state: newFakeObjectState(fake.FakeObj(name, func(obj *v1alpha1.PD) *v1alpha1.PD {
				return obj
			})),
			objs: []client.Object{
				fake.FakeObj(name, fake.InstanceOwner[scope.PD, corev1.Pod](fake.FakeObj[v1alpha1.PD](name))),
			},
			unexpectedErr:  true,
			expectedResult: task.SFail,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			fc := client.NewFakeClient(c.objs...)

			if c.unexpectedErr {
				fc.WithError("*", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}

			res, done := task.RunTask(context.Background(), TaskContextPod[scope.PD](c.state, fc))
			assert.Equal(tt, c.expectedResult, res.Status(), c.desc)
			assert.False(tt, done, c.desc)
			assert.Equal(tt, c.expectedObj, c.state.pod, c.desc)
		})
	}
}

func TestTaskSuspendPod(t *testing.T) {
	now := metav1.Now()
	cases := []struct {
		desc          string
		state         *fakeState[corev1.Pod]
		objs          []client.Object
		unexpectedErr bool

		expectedResult task.Status
		expectedObj    *corev1.Pod
	}{
		{
			desc:           "pod is nil",
			state:          &fakeState[corev1.Pod]{},
			expectedResult: task.SComplete,
		},
		{
			desc: "pod is deleting",
			state: &fakeState[corev1.Pod]{
				obj: fake.FakeObj("aaa", fake.DeleteTimestamp[corev1.Pod](&now)),
			},
			expectedResult: task.SComplete,
			expectedObj:    fake.FakeObj("aaa", fake.DeleteTimestamp[corev1.Pod](&now)),
		},
		{
			// means pod has been fully deleted after it is fetched previously
			desc: "pod is deleted",
			state: &fakeState[corev1.Pod]{
				obj: fake.FakeObj[corev1.Pod]("aaa"),
			},
			expectedResult: task.SComplete,
			expectedObj:    fake.FakeObj[corev1.Pod]("aaa"),
		},
		{
			desc: "delete pod",
			state: &fakeState[corev1.Pod]{
				obj: fake.FakeObj[corev1.Pod]("aaa"),
			},
			objs: []client.Object{
				fake.FakeObj[corev1.Pod]("aaa"),
			},
			expectedResult: task.SRetry,
			expectedObj:    fake.FakeObj[corev1.Pod]("aaa"),
		},
		{
			desc: "delete pod with unexpected err",
			state: &fakeState[corev1.Pod]{
				obj: fake.FakeObj[corev1.Pod]("aaa"),
			},
			objs: []client.Object{
				fake.FakeObj[corev1.Pod]("aaa"),
			},
			unexpectedErr:  true,
			expectedResult: task.SFail,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			fc := client.NewFakeClient(c.objs...)

			if c.unexpectedErr {
				fc.WithError("*", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}

			s := &fakePodState{s: c.state}
			res, done := task.RunTask(context.Background(), TaskSuspendPod(s, fc))
			assert.Equal(tt, c.expectedResult, res.Status(), c.desc)
			assert.False(tt, done, c.desc)
			if !c.unexpectedErr {
				assert.Equal(tt, c.expectedObj, c.state.obj, c.desc)
			}
		})
	}
}

func TestTaskServerLabels(t *testing.T) {
	cases := []struct {
		desc          string
		state         *fakeServerLabelsState
		objs          []client.Object
		pdConfig      *pdapi.PDConfigFromAPI
		pdErr         error
		setLabelsErr  error
		unexpectedErr bool

		expectedResult task.Status
		expectedLabels map[string]string
	}{
		{
			desc: "instance is not healthy",
			state: &fakeServerLabelsState{
				healthy: false,
			},
			expectedResult: task.SComplete,
		},
		{
			desc: "pod is nil",
			state: &fakeServerLabelsState{
				healthy: true,
			},
			expectedResult: task.SComplete,
		},
		{
			desc: "pod is terminating",
			state: &fakeServerLabelsState{
				healthy: true,
				pod: fake.FakeObj("test-pod", func(obj *corev1.Pod) *corev1.Pod {
					obj.DeletionTimestamp = &metav1.Time{Time: time.Now()}
					return obj
				}),
			},
			expectedResult: task.SComplete,
		},
		{
			desc: "pod is not scheduled",
			state: &fakeServerLabelsState{
				healthy: true,
				pod: fake.FakeObj("test-pod", func(obj *corev1.Pod) *corev1.Pod {
					obj.Spec.NodeName = ""
					return obj
				}),
			},
			expectedResult: task.SFail,
		},
		{
			desc: "failed to get node",
			state: &fakeServerLabelsState{
				healthy: true,
				pod: fake.FakeObj("test-pod", func(obj *corev1.Pod) *corev1.Pod {
					obj.Spec.NodeName = "test-node"
					return obj
				}),
			},
			unexpectedErr:  true,
			expectedResult: task.SFail,
		},
		{
			desc: "failed to get pd config",
			state: &fakeServerLabelsState{
				healthy: true,
				pod: fake.FakeObj("test-pod", func(obj *corev1.Pod) *corev1.Pod {
					obj.Spec.NodeName = "test-node"
					return obj
				}),
			},
			objs: []client.Object{
				fake.FakeObj("test-node", func(obj *corev1.Node) *corev1.Node {
					obj.Name = "test-node"
					return obj
				}),
			},
			pdErr:          fmt.Errorf("failed to get pd config"),
			expectedResult: task.SFail,
		},
		{
			desc: "zone label not found",
			state: &fakeServerLabelsState{
				healthy: true,
				pod: fake.FakeObj("test-pod", func(obj *corev1.Pod) *corev1.Pod {
					obj.Spec.NodeName = "test-node"
					return obj
				}),
				serverLabels: map[string]string{
					"foo": "bar",
				},
			},
			objs: []client.Object{
				fake.FakeObj("test-node", func(obj *corev1.Node) *corev1.Node {
					obj.Name = "test-node"
					return obj
				}),
			},
			pdConfig: &pdapi.PDConfigFromAPI{
				Replication: &pdapi.PDReplicationConfig{
					LocationLabels: []string{"other-label"},
				},
			},
			expectedResult: task.SComplete,
			expectedLabels: map[string]string{
				"foo": "bar",
			},
		},
		{
			desc: "failed to set labels",
			state: &fakeServerLabelsState{
				healthy: true,
				pod: fake.FakeObj("test-pod", func(obj *corev1.Pod) *corev1.Pod {
					obj.Spec.NodeName = "test-node"
					return obj
				}),
				serverLabels: map[string]string{
					"existing-label": "value",
				},
			},
			objs: []client.Object{
				fake.FakeObj("test-node", func(obj *corev1.Node) *corev1.Node {
					obj.Name = "test-node"
					obj.Labels = map[string]string{
						"zone": "zone1",
					}
					return obj
				}),
			},
			pdConfig: &pdapi.PDConfigFromAPI{
				Replication: &pdapi.PDReplicationConfig{
					LocationLabels: []string{"zone"},
				},
			},
			setLabelsErr:   fmt.Errorf("failed to set labels"),
			expectedResult: task.SFail,
		},
		{
			desc: "success",
			state: &fakeServerLabelsState{
				healthy: true,
				pod: fake.FakeObj("test-pod", func(obj *corev1.Pod) *corev1.Pod {
					obj.Spec.NodeName = "test-node"
					return obj
				}),
				serverLabels: map[string]string{
					"existing-label": "value",
				},
			},
			objs: []client.Object{
				fake.FakeObj("test-node", func(obj *corev1.Node) *corev1.Node {
					obj.Name = "test-node"
					obj.Labels = map[string]string{
						"zone": "zone1",
					}
					return obj
				}),
			},
			pdConfig: &pdapi.PDConfigFromAPI{
				Replication: &pdapi.PDReplicationConfig{
					LocationLabels: []string{"zone"},
				},
			},
			expectedResult: task.SComplete,
			expectedLabels: map[string]string{
				"existing-label": "value",
				"zone":           "zone1",
			},
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			fc := client.NewFakeClient(c.objs...)
			if c.unexpectedErr {
				fc.WithError("*", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}

			pdClient := pdapi.NewMockPDClient(gomock.NewController(tt))
			if c.pdConfig != nil {
				pdClient.EXPECT().GetConfig(gomock.Any()).Return(c.pdConfig, nil)
			} else if c.pdErr != nil {
				pdClient.EXPECT().GetConfig(gomock.Any()).Return(nil, c.pdErr)
			} else {
				pdClient.EXPECT().GetConfig(gomock.Any()).Return(&pdapi.PDConfigFromAPI{
					Replication: &pdapi.PDReplicationConfig{
						LocationLabels: []string{},
					},
				}, nil).AnyTimes()
			}
			c.state.SetPDClient(pdClient)
			setLabelsCalled := false
			setLabelsFunc := func(ctx context.Context, labels map[string]string) error {
				setLabelsCalled = true
				if c.setLabelsErr != nil {
					return c.setLabelsErr
				}
				c.state.serverLabels = labels
				return nil
			}

			res, done := task.RunTask(context.Background(), TaskServerLabels[scope.TiDB](c.state, fc, setLabelsFunc))
			assert.Equal(tt, c.expectedResult, res.Status(), c.desc)
			assert.False(tt, done, c.desc)

			if c.expectedLabels != nil {
				assert.True(tt, setLabelsCalled, c.desc)
				assert.Equal(tt, c.expectedLabels, c.state.serverLabels, c.desc)
			}
		})
	}
}
