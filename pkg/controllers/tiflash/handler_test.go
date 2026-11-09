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

package tiflash

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/timanager"
	pdv1 "github.com/pingcap/tidb-operator/v2/pkg/timanager/apis/pd/v1"
)

func TestStoreEventHandlesMalformedInstanceName(t *testing.T) {
	for _, valid := range []bool{false, true} {
		t.Run(map[bool]string{false: "invalid", true: "valid"}[valid], func(t *testing.T) {
			name := "invalid"
			if valid {
				name = "instance-abc"
			}
			instance := &v1alpha1.TiFlash{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "test", Labels: map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyCluster:   "cluster",
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiFlash,
			}}}
			cluster := &v1alpha1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "test"}}
			address := "no-match"
			if valid {
				address = coreutil.InstanceAdvertiseAddress[scope.TiFlash](cluster, instance, coreutil.TiFlashFlashPort(instance))
			}
			store := &pdv1.Store{ObjectMeta: metav1.ObjectMeta{Name: address, Namespace: timanager.PrimaryKey("test", "cluster")}}
			objects := []client.Object{cluster, instance}
			if valid {
				malformed := instance.DeepCopy()
				malformed.Name = "aaa"
				objects = append(objects, malformed)
			}
			r := &Reconciler{Client: client.NewFakeClient(objects...), Logger: logr.Discard()}
			req, err := r.getRequestOfTiFlashStore(context.Background(), store)
			if valid {
				require.NoError(t, err)
				require.Equal(t, name, req.Name)
			} else {
				require.Error(t, err)
			}
		})
	}
}
