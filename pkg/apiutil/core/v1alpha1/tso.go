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

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	corev1 "k8s.io/api/core/v1"
)

func TSOClientPort(tso *v1alpha1.TSO) int32 {
	if tso.Spec.Server.Ports.Client != nil {
		return tso.Spec.Server.Ports.Client.Port
	}
	return v1alpha1.DefaultTSOPortClient
}

func TSOGroupClientPort(tg *v1alpha1.TSOGroup) int32 {
	if tg.Spec.Template.Spec.Server.Ports.Client != nil {
		return tg.Spec.Template.Spec.Server.Ports.Client.Port
	}
	return v1alpha1.DefaultTSOPortClient
}

func TSOAdvertiseClientURLs(tso *v1alpha1.TSO) string {
	ns := tso.Namespace
	if ns == "" {
		ns = corev1.NamespaceDefault
	}
	return fmt.Sprintf("%s.%s.%s:%d", PodName[scope.TSO](tso), tso.Spec.Subdomain, ns, TSOClientPort(tso))
}

func TSOAdvertiseClientURLsWithScheme(tso *v1alpha1.TSO, isTLS bool) string {
	return hostToURL(TSOAdvertiseClientURLs(tso), isTLS)
}

func TSOServiceHost(tg *v1alpha1.TSOGroup) string {
	host := fmt.Sprintf("%s-tso-peer.%s:%d", tg.Name, tg.Namespace, TSOGroupClientPort(tg))

	return host
}

func TSOServiceURL(tg *v1alpha1.TSOGroup, isTLS bool) string {
	return hostToURL(TSOServiceHost(tg), isTLS)
}
