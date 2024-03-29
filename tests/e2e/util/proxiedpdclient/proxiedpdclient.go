// Copyright 2019 PingCAP, Inc.
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

package proxiedpdclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/pkg/util"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	utilportforward "github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
)

// NewProxiedPDClient creates an PD client which can be used outside of the
// Kubernetes cluster.
//
// Example:
//
//	   pdClient, cancel, err := NewProxiedPDClient(...)
//	   if err != nil {
//			log.Fatal(err)
//		  }
//	   defer cancel()
func NewProxiedPDClient(secretLister corelisterv1.SecretLister, fw utilportforward.PortForward, namespace string, tcName string, tlsEnabled bool) (pdapi.PDClient, context.CancelFunc, error) {
	var tlsConfig *tls.Config
	var err error
	scheme := "http"
	if tlsEnabled {
		scheme = "https"
		tlsConfig, err = pdapi.GetTLSConfig(secretLister, pdapi.Namespace(namespace), util.ClusterClientTLSSecretName(tcName))
		if err != nil {
			return nil, nil, err
		}
	}
	localHost, localPort, cancel, err := portforward.ForwardOnePort(fw, namespace, fmt.Sprintf("svc/%s", controller.PDMemberName(tcName)), uint16(v1alpha1.DefaultPDClientPort))
	if err != nil {
		return nil, nil, err
	}
	u := url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s:%d", localHost, localPort),
	}
	return pdapi.NewPDClient(u.String(), pdapi.DefaultTimeout, tlsConfig), cancel, nil
}

func NewProxiedPDClientFromTidbCluster(fw utilportforward.PortForward, secretLister corelisterv1.SecretLister, tc *v1alpha1.TidbCluster) (pdapi.PDClient, context.CancelFunc, error) {
	return NewProxiedPDClient(secretLister, fw, tc.GetNamespace(), tc.GetName(), tc.IsTLSClusterEnabled())
}
