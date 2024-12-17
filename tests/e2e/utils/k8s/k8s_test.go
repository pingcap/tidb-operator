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

package k8s

import (
	"math/rand"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func makeTestEvent(podName string, typ watch.EventType, tim time.Time) watch.Event {
	return watch.Event{
		Type: typ,
		Object: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              podName,
				CreationTimestamp: metav1.Time{Time: tim},
				DeletionTimestamp: &metav1.Time{Time: tim},
			},
		},
	}
}

const fixedUnixTime = 1257894000

func getTestTime(s int) time.Time {
	return time.Unix(fixedUnixTime+int64(s), 0)
}

func shuffleEvents(events []watch.Event) []watch.Event {
	for i := range events {
		//nolint:gosec // no need to use cryptographically secure random number generator
		j := rand.Intn(i + 1)
		events[i], events[j] = events[j], events[i]
	}
	return events
}

func TestCheckRollingRestartLogic(t *testing.T) {
	tests := []struct {
		name                string
		eventsBeforeShuffle []watch.Event
		want                bool
	}{
		{
			name: "happy path",
			eventsBeforeShuffle: []watch.Event{
				makeTestEvent("pod1", watch.Added, getTestTime(1)),
				makeTestEvent("pod2", watch.Added, getTestTime(2)),
				makeTestEvent("pod3", watch.Added, getTestTime(3)),

				makeTestEvent("pod2", watch.Deleted, getTestTime(4)),
				makeTestEvent("pod2", watch.Added, getTestTime(5)),
				makeTestEvent("pod3", watch.Deleted, getTestTime(6)),
				makeTestEvent("pod3", watch.Added, getTestTime(7)),
				makeTestEvent("pod1", watch.Deleted, getTestTime(8)),
				makeTestEvent("pod1", watch.Added, getTestTime(9)),
			},
			want: true,
		},
		{
			name:                "Empty events",
			eventsBeforeShuffle: []watch.Event{},
			want:                false,
		},
		{
			name: "only add events",
			eventsBeforeShuffle: []watch.Event{
				makeTestEvent("pod1", watch.Added, getTestTime(1)),
				makeTestEvent("pod2", watch.Added, getTestTime(2)),
				makeTestEvent("pod3", watch.Added, getTestTime(3)),
			},
			want: false,
		},
		{
			name: "Alternating delete and add events",
			eventsBeforeShuffle: []watch.Event{
				makeTestEvent("pod1", watch.Added, getTestTime(1)),
				makeTestEvent("pod1", watch.Deleted, getTestTime(2)),
				makeTestEvent("pod2", watch.Added, getTestTime(3)),
				makeTestEvent("pod2", watch.Deleted, getTestTime(4)),
			},
			want: false,
		},
		{
			name: "two pods are deleted at the same time",
			eventsBeforeShuffle: []watch.Event{
				makeTestEvent("pod2", watch.Added, getTestTime(1)),
				makeTestEvent("pod1", watch.Added, getTestTime(2)),
				makeTestEvent("pod3", watch.Added, getTestTime(3)),
				makeTestEvent("pod1", watch.Deleted, getTestTime(4)),
				makeTestEvent("pod3", watch.Deleted, getTestTime(4)),
				makeTestEvent("pod1", watch.Added, getTestTime(5)),
				makeTestEvent("pod3", watch.Added, getTestTime(5)),
				makeTestEvent("pod2", watch.Deleted, getTestTime(6)),
				makeTestEvent("pod2", watch.Added, getTestTime(7)),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CheckRollingRestartLogic(shuffleEvents(tt.eventsBeforeShuffle)); got != tt.want {
				t.Errorf("CheckRollingRestartLogic() = %v, want %v", got, tt.want)
			}
		})
	}
}
