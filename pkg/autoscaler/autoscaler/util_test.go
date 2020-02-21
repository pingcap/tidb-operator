// Copyright 2020 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package autoscaler

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"k8s.io/utils/pointer"
)

func newTidbClusterAutoScaler() *v1alpha1.TidbClusterAutoScaler {
	tac := &v1alpha1.TidbClusterAutoScaler{}
	tac.Annotations = map[string]string{}
	tac.Spec.TiKV = &v1alpha1.TikvAutoScalerSpec{}
	tac.Spec.TiDB = &v1alpha1.TidbAutoScalerSpec{}
	tac.Spec.TiKV.ScaleOutThreshold = pointer.Int32Ptr(2)
	tac.Spec.TiKV.ScaleInThreshold = pointer.Int32Ptr(2)
	tac.Spec.TiDB.ScaleOutThreshold = pointer.Int32Ptr(2)
	tac.Spec.TiDB.ScaleInThreshold = pointer.Int32Ptr(2)
	defaultTAC(tac)
	return tac
}

func newTidbCluster() *v1alpha1.TidbCluster {
	tc := &v1alpha1.TidbCluster{}
	tc.Annotations = map[string]string{}
	tc.Name = "tc"
	tc.Namespace = "ns"
	return tc
}
