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

package waiter

import (
	"context"
	"testing"
	"time"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/utils/ptr"
)

func TestCheckPodsRollingUpdateOnce(t *testing.T) {
	alive := podInfo{name: "tikv", uid: "original", creationTime: metav1.NewTime(time.Unix(1, 0))}
	deleted := alive
	deleted.deletionTime = metav1.NewTime(time.Unix(2, 0))
	replacement := podInfo{name: "tikv", uid: "replacement", creationTime: metav1.NewTime(time.Unix(3, 0))}
	terminating := alive
	terminating.terminating = true
	cases := []struct {
		name            string
		from, to, surge int
		noRestart       bool
		infos           []podInfo
		wantErr         string
	}{
		{name: "unchanged", from: 1, to: 1, noRestart: true, infos: []podInfo{alive}},
		{name: "unused surge does not require replacement", from: 1, to: 1, surge: 1, noRestart: true, infos: []podInfo{alive}},
		{name: "recreated", from: 1, to: 1, noRestart: true, infos: []podInfo{deleted, replacement}, wantErr: "expect 1 pods info"},
		{name: "deleted without replacement", from: 1, to: 1, noRestart: true, infos: []podInfo{deleted}, wantErr: "should not be deleted"},
		{name: "terminating", from: 1, to: 1, noRestart: true, infos: []podInfo{terminating}, wantErr: "should not be deleted"},
		{name: "empty observation", from: 1, to: 1, noRestart: true, wantErr: "expect 1 pods info"},
		{name: "scale out without rolling", from: 1, to: 2, noRestart: true, infos: []podInfo{alive, replacement}},
		{name: "scale in without rolling", from: 2, to: 1, noRestart: true, infos: []podInfo{alive, deleted}},
		{name: "scale out with unexpected recreation", from: 1, to: 2, noRestart: true, infos: []podInfo{alive, deleted, replacement}, wantErr: "expect 2 pods info"},
		{name: "rolling once", from: 1, to: 1, infos: []podInfo{deleted, replacement}},
		{name: "rolling missing", from: 1, to: 1, infos: []podInfo{alive}, wantErr: "expect 2 pods info"},
		{name: "incomplete scale out", from: 1, to: 5, infos: []podInfo{alive}, wantErr: "pods info"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkPodsRollingUpdateOnce(tc.infos, tc.from, tc.to, tc.surge, tc.noRestart)
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.wantErr)
			}
		})
	}
}

// Only list/watch are used by the pod observer.
type podWatchClient struct {
	client.Client
	pod    corev1.Pod
	stream *watch.RaceFreeFakeWatcher
}

func (c *podWatchClient) List(_ context.Context, list client.ObjectList, _ ...client.ListOption) error {
	pods := list.(*corev1.PodList)
	pods.ResourceVersion = "1"
	pods.Items = []corev1.Pod{c.pod}
	return nil
}

func (c *podWatchClient) Watch(_ context.Context, _ client.ObjectList, _ ...client.ListOption) (watch.Interface, error) {
	return c.stream, nil
}

func TestWaitPodsRollingUpdateOnceSynchronized(t *testing.T) {
	cases := []struct {
		name                string
		noRestart, recreate bool
		wantErr             string
	}{
		{name: "cancel immediately after synchronization", noRestart: true},
		{name: "detect recreation after synchronization", noRestart: true, recreate: true, wantErr: "expect 1 pods info"},
		{name: "rolling update after synchronization", recreate: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stream := watch.NewRaceFreeFake()
			defer stream.Stop()
			original := corev1.Pod{ObjectMeta: metav1.ObjectMeta{
				Name: "tikv", Namespace: "test", UID: "original", ResourceVersion: "1",
				CreationTimestamp: metav1.NewTime(time.Now().Add(-time.Minute)),
			}}
			c := &podWatchClient{pod: original, stream: stream}
			group := &v1alpha1.TiKVGroup{ObjectMeta: metav1.ObjectMeta{Name: "kvg", Namespace: "test"}, Spec: v1alpha1.TiKVGroupSpec{Replicas: ptr.To[int32](1)}}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			synced := make(chan struct{})
			done := make(chan error, 1)
			go func() {
				done <- WaitPodsRollingUpdateOnce[scope.TiKVGroup](ctx, c, group, 1, 0, tc.noRestart, time.Second, synced)
			}()
			select {
			case <-synced:
			case err := <-done:
				t.Fatalf("observer ended before synchronization: %v", err)
			}
			if tc.recreate {
				stream.Delete(&original)
				replacement := original.DeepCopy()
				replacement.UID = "replacement"
				replacement.ResourceVersion = "2"
				replacement.CreationTimestamp = metav1.NewTime(time.Now().Add(time.Second))
				stream.Add(replacement)
			} else {
				cancel()
			}
			err := <-done
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.wantErr)
			}
		})
	}
}
