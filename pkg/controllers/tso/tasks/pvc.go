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
	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
)

func PVCNewer() common.PVCNewer[*v1alpha1.TSO] {
	return common.PVCNewerFunc[*v1alpha1.TSO](
		func(cluster *v1alpha1.Cluster, tso *v1alpha1.TSO, fg features.Gates) []*corev1.PersistentVolumeClaim {
			pvcs := coreutil.PVCs[scope.TSO](
				cluster,
				tso,
				coreutil.EnableVAC(fg.Enabled(meta.VolumeAttributesClass)),
			)

			return pvcs
		},
	)
}