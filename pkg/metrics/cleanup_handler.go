// Copyright 2026 PingCAP, Inc.
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

package metrics

import (
	"context"
	"fmt"

	toolscache "k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

// abnormalInstanceKinds lists every instance CR that writes to the
// AbnormalInstance gauge via TaskInstanceConditionSynced / Ready. Keep this in
// sync with the controllers that wire those tasks into their builder.
var abnormalInstanceKinds = []client.Object{
	&v1alpha1.PD{},
	&v1alpha1.TiDB{},
	&v1alpha1.TiKV{},
	&v1alpha1.TiFlash{},
	&v1alpha1.TiCDC{},
	&v1alpha1.TSO{},
	&v1alpha1.Scheduling{},
	&v1alpha1.TiProxy{},
	&v1alpha1.TiKVWorker{},
}

// RegisterAbnormalInstanceCleanup attaches a watch-level delete handler on
// every instance CR so the AbnormalInstance gauge is cleared the moment the
// object disappears from the API server, even when the deletion bypasses the
// operator's finalizer (e.g. a manual `patch metadata.finalizers=null` used to
// unwedge a stuck instance). Without this, such paths skip
// TaskInstanceFinalizerDel entirely and leave the gauge at its last value
// forever, causing false-positive `metric == 1 for: <duration>` alerts on a
// non-existent instance and unbounded label-cardinality growth.
//
// This complements, rather than replaces, the finalizer-time cleanup: the
// finalizer path covers graceful deletes and short-circuits the observe loop
// before it can re-populate the gauge, while this handler covers the paths
// that never reach the finalizer at all.
func RegisterAbnormalInstanceCleanup(ctx context.Context, mgr ctrl.Manager) error {
	for _, obj := range abnormalInstanceKinds {
		inf, err := mgr.GetCache().GetInformer(ctx, obj)
		if err != nil {
			return fmt.Errorf("get informer for %T: %w", obj, err)
		}
		if _, err := inf.AddEventHandler(toolscache.ResourceEventHandlerFuncs{
			DeleteFunc: clearOnDelete,
		}); err != nil {
			return fmt.Errorf("add delete handler for %T: %w", obj, err)
		}
	}
	return nil
}

func clearOnDelete(o interface{}) {
	if t, ok := o.(toolscache.DeletedFinalStateUnknown); ok {
		o = t.Obj
	}
	co, ok := o.(client.Object)
	if !ok {
		return
	}
	ClearInstanceConditionMetrics(co)
}
