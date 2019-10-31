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

package webhook

import (
	"net/http"

	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/webhook/route"
	"github.com/pingcap/tidb-operator/pkg/webhook/util"
	"k8s.io/client-go/kubernetes"
	glog "k8s.io/klog"
)

type WebhookServer struct {
	// kubernetes client interface
	KubeCli kubernetes.Interface
	// operator client interface
	Cli versioned.Interface
	// http server
	Server *http.Server
}

func NewWebHookServer(kubecli kubernetes.Interface, cli versioned.Interface, certFile string, keyFile string) *WebhookServer {

	http.HandleFunc("/statefulsets", route.ServeStatefulSets)

	sCert, err := util.ConfigTLS(certFile, keyFile)

	if err != nil {
		glog.Fatalf("failed to create scert file %v", err)
	}

	server := &http.Server{
		Addr:      ":443",
		TLSConfig: sCert,
	}

	return &WebhookServer{
		KubeCli: kubecli,
		Cli:     cli,
		Server:  server,
	}
}

func (ws *WebhookServer) Run() error {
	return ws.Server.ListenAndServeTLS("", "")
}

func (ws *WebhookServer) Shutdown() error {
	return ws.Server.Shutdown(nil)
}
