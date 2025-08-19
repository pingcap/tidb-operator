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

package coreutil

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
)

func ReplicationWorkerGRPCPort(rw *v1alpha1.ReplicationWorker) int32 {
	if rw.Spec.Server.Ports.GRPC != nil {
		return rw.Spec.Server.Ports.GRPC.Port
	}
	return v1alpha1.DefaultReplicationWorkerPortGRPC
}

func ReplicationWorkerGroupGRPCPort(rwg *v1alpha1.ReplicationWorkerGroup) int32 {
	if rwg.Spec.Template.Spec.Server.Ports.GRPC != nil {
		return rwg.Spec.Template.Spec.Server.Ports.GRPC.Port
	}
	return v1alpha1.DefaultReplicationWorkerPortGRPC
}

func ReplicationWorkerAdvertiseURL(rw *v1alpha1.ReplicationWorker) string {
	ns := rw.Namespace
	if ns == "" {
		ns = corev1.NamespaceDefault
	}
	return fmt.Sprintf("%s.%s.%s:%d", PodName[scope.ReplicationWorker](rw), rw.Spec.Subdomain, ns, ReplicationWorkerGRPCPort(rw))
}
