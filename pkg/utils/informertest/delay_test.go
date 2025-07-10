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

package informertest

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

func TestDelayedListerWatcher(t *testing.T) {
	for i := range 3 {
		t.Run("delay "+strconv.Itoa(i+1)+" seconds", func(tt *testing.T) {
			ch := make(chan watch.Event, 100)
			lw := cache.ListWatch{
				ListFunc: func(metav1.ListOptions) (runtime.Object, error) {
					return &corev1.PodList{}, nil
				},
				WatchFunc: func(metav1.ListOptions) (watch.Interface, error) {
					pw := watch.NewProxyWatcher(ch)
					return pw, nil
				},
			}
			delay := time.Second * time.Duration(i+1)
			dlw := &delayedListerWatcher{
				lw:    &lw,
				delay: delay,
			}

			done := make(chan struct{})
			w, err := dlw.Watch(metav1.ListOptions{})
			require.NoError(t, err)
			go func() {
				select {
				case <-w.ResultChan():
					assert.Fail(tt, "receive result before delay time")
				case <-time.After(delay - time.Millisecond*200):
					select {
					case <-w.ResultChan():
					case <-time.After(time.Millisecond * 500):
						assert.Fail(tt, "cannot receive result after delay time")
					}
				}
				close(done)
			}()

			ch <- watch.Event{}
			<-done
			w.Stop()
			close(ch)
		})
	}
}
