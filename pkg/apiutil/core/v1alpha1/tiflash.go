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
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
)

func TiFlashGroupFlashPort(fg *v1alpha1.TiFlashGroup) int32 {
	if fg.Spec.Template.Spec.Server.Ports.Flash != nil {
		return fg.Spec.Template.Spec.Server.Ports.Flash.Port
	}
	return v1alpha1.DefaultTiFlashPortFlash
}

func TiFlashGroupProxyPort(fg *v1alpha1.TiFlashGroup) int32 {
	if fg.Spec.Template.Spec.Server.Ports.Proxy != nil {
		return fg.Spec.Template.Spec.Server.Ports.Proxy.Port
	}
	return v1alpha1.DefaultTiFlashPortProxy
}

func TiFlashGroupMetricsPort(fg *v1alpha1.TiFlashGroup) int32 {
	if fg.Spec.Template.Spec.Server.Ports.Metrics != nil {
		return fg.Spec.Template.Spec.Server.Ports.Metrics.Port
	}
	return v1alpha1.DefaultTiFlashPortMetrics
}

func TiFlashGroupProxyStatusPort(fg *v1alpha1.TiFlashGroup) int32 {
	if fg.Spec.Template.Spec.Server.Ports.ProxyStatus != nil {
		return fg.Spec.Template.Spec.Server.Ports.ProxyStatus.Port
	}
	return v1alpha1.DefaultTiFlashPortProxyStatus
}

func TiFlashFlashPort(f *v1alpha1.TiFlash) int32 {
	if f.Spec.Server.Ports.Flash != nil {
		return f.Spec.Server.Ports.Flash.Port
	}
	return v1alpha1.DefaultTiFlashPortFlash
}

func TiFlashProxyPort(f *v1alpha1.TiFlash) int32 {
	if f.Spec.Server.Ports.Proxy != nil {
		return f.Spec.Server.Ports.Proxy.Port
	}
	return v1alpha1.DefaultTiFlashPortProxy
}

func TiFlashMetricsPort(f *v1alpha1.TiFlash) int32 {
	if f.Spec.Server.Ports.Metrics != nil {
		return f.Spec.Server.Ports.Metrics.Port
	}
	return v1alpha1.DefaultTiFlashPortMetrics
}

func TiFlashProxyStatusPort(f *v1alpha1.TiFlash) int32 {
	if f.Spec.Server.Ports.ProxyStatus != nil {
		return f.Spec.Server.Ports.ProxyStatus.Port
	}
	return v1alpha1.DefaultTiFlashPortProxyStatus
}

func TiFlashProxyStatusURL(f *v1alpha1.TiFlash, isTLS bool) string {
	ns := f.Namespace
	if ns == "" {
		ns = corev1.NamespaceDefault
	}
	host := fmt.Sprintf("%s.%s.%s:%d", PodName[scope.TiFlash](f), f.Spec.Subdomain, ns, TiFlashProxyStatusPort(f))

	return hostToURL(host, isTLS)
}
