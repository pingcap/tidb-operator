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

package data

import (
	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
)

type (
	ClusterPatch func(obj *v1alpha1.Cluster)
	// GroupPatch[G runtime.Group] func(obj G)
)

type GroupPatchFunc[
	F client.Object,
] func(obj F)

type GroupPatch[F client.Object] interface {
	Patch(obj F)
}

func (f GroupPatchFunc[F]) Patch(obj F) {
	f(obj)
}

const (
	defaultClusterName              = "tc"
	defaultPDGroupName              = "pdg"
	defaultTiDBGroupName            = "dbg"
	defaultTiKVGroupName            = "kvg"
	defaultTiFlashGroupName         = "fg"
	defaultTiCDCGroupName           = "cg"
	defaultTSOGroupName             = "tg"
	defaultSchedulingGroupName      = "sg"
	defaultResourceManagerGroupName = "rmg"
	defaultTiProxyGroupName         = "pg"
	defaultTiKVWorkerGroupName      = "wg"

	defaultVersion        = "v8.5.2"
	defaultTiProxyVersion = "v1.3.0"

	defaultImageRegistry = "gcr.io/pingcap-public/dbaas/"
	defaultHelperImage   = "gcr.io/pingcap-public/dbaas/busybox:1.36.0"
)

const (
	// TODO(liubo02): extract to namer
	DefaultTiDBServiceName    = defaultTiDBGroupName + "-tidb"
	DefaultTiProxyServiceName = defaultTiProxyGroupName + "-tiproxy"
	DefaultTiProxyServicePort = 6000
)

func WithReplicas[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](replicas int32) GroupPatch[F] {
	return GroupPatchFunc[F](func(obj F) {
		coreutil.SetReplicas[S](obj, replicas)
	})
}

func WithName[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](name string) GroupPatch[F] {
	return GroupPatchFunc[F](func(obj F) {
		obj.SetName(name)
	})
}

func WithVersion[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](version string) GroupPatch[F] {
	return GroupPatchFunc[F](func(obj F) {
		coreutil.SetVersion[S](obj, version)
	})
}

func WithImage[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](image string) GroupPatch[F] {
	return GroupPatchFunc[F](func(obj F) {
		coreutil.SetImage[S](obj, image)
	})
}

func WithCluster[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](cluster string) GroupPatch[F] {
	return GroupPatchFunc[F](func(obj F) {
		coreutil.SetCluster[S](obj, cluster)
	})
}

func WithClusterTLS[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](ca, certKeyPair string) GroupPatch[F] {
	return GroupPatchFunc[F](func(obj F) {
		coreutil.SetTemplateClusterTLS[S](obj, ca, certKeyPair)
	})
}

func WithClusterTLSSuffix[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](ca, certKeyPair string) GroupPatch[F] {
	return GroupPatchFunc[F](func(obj F) {
		// Add ns prefix of ca because the bundle is a cluster scope resource
		ca = obj.GetNamespace() + "-" + ca
		// Default generated certs in test contain group names in dnsNames,
		// so cannot be reused by different groups
		certKeyPair = obj.GetName() + "-" + scope.Component[S]() + "-" + certKeyPair

		coreutil.SetTemplateClusterTLS[S](obj, ca, certKeyPair)
	})
}

func WithTemplateAnnotation[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](k, v string) GroupPatch[F] {
	return GroupPatchFunc[F](func(obj F) {
		anno := coreutil.TemplateAnnotations[S](obj)
		if anno == nil {
			anno = map[string]string{}
		}

		anno[k] = v
		coreutil.SetTemplateAnnotations[S](obj, anno)
	})
}
