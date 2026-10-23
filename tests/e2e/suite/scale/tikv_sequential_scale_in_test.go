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

package scale

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func TestWaitForSequentialTiKVRemoval(t *testing.T) {
	for _, tt := range []struct {
		name    string
		overlap bool
		deleted bool
	}{
		{name: "sequential"},
		{name: "overlapping", overlap: true},
		{name: "finalized without Removed status", deleted: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			stream := watch.NewRaceFreeFake()
			defer stream.Stop()
			for _, name := range []string{"a", "b", "c"} {
				kv := &v1alpha1.TiKV{ObjectMeta: metav1.ObjectMeta{Name: name}}
				kv.Status.State = v1alpha1.StoreStateRemoving
				stream.Modify(kv.DeepCopy())
				if tt.overlap {
					continue
				}
				if tt.deleted {
					stream.Delete(kv.DeepCopy())
				} else {
					kv.Status.State = v1alpha1.StoreStateRemoved
					stream.Modify(kv.DeepCopy())
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err := waitForSequentialTiKVRemoval(ctx, stream, 3)
			if tt.overlap {
				require.ErrorContains(t, err, "while a was still Removing")
			} else {
				require.NoError(t, err)
			}
		})
	}
}
