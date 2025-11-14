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

package tibrgc

import (
	"context"
	"reflect"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1alpha1br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
)

const (
	clusterNameField = "spec.cluster.name"
)

func (r *Reconciler) makeClusterEventHandler(mgr manager.Manager) (handler.TypedEventHandler[client.Object, reconcile.Request], error) {
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(), &v1alpha1br.TiBRGC{},
		clusterNameField,
		func(obj client.Object) []string {
			br := obj.(*v1alpha1br.TiBRGC)
			return []string{br.Spec.Cluster.Name}
		}); err != nil {
		return nil, err
	}

	return handler.TypedFuncs[client.Object, reconcile.Request]{
		UpdateFunc: func(ctx context.Context, event event.TypedUpdateEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request],
		) {
			logger := r.Logger
			oldObj := event.ObjectOld.(*v1alpha1.Cluster)
			newObj := event.ObjectNew.(*v1alpha1.Cluster)

			if newObj.Status.PD != oldObj.Status.PD {
				logger.Info("pd url is updating", "from", oldObj.Status.PD, "to", newObj.Status.PD)
			} else if !reflect.DeepEqual(oldObj.Spec.SuspendAction, newObj.Spec.SuspendAction) {
				logger.Info("suspend action is updating", "from", oldObj.Spec.SuspendAction, "to", newObj.Spec.SuspendAction)
			} else if oldObj.Spec.Paused != newObj.Spec.Paused {
				logger.Info("cluster paused is updating", "from", oldObj.Spec.Paused, "to", newObj.Spec.Paused)
			} else {
				return
			}

			var brgclist v1alpha1br.TiBRGCList
			if err := r.Client.List(ctx, &brgclist, &client.ListOptions{
				Namespace: newObj.Namespace,
				FieldSelector: fields.SelectorFromSet(map[string]string{
					clusterNameField: newObj.Name,
				}),
			}); err != nil {
				r.Logger.Error(err, "cannot list all tibrgc instances", "ns", newObj.Namespace, "cluster", newObj.Name)
				return
			}

			for i := range brgclist.Items {
				brgc := &brgclist.Items[i]
				queue.Add(reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      brgc.Name,
						Namespace: brgc.Namespace,
					},
				})
			}
		},
	}, nil
}
