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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/pkg/webhook/pod"
	"github.com/pingcap/tidb-operator/pkg/webhook/util"
	corev1 "k8s.io/api/core/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	eventv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	glog "k8s.io/klog"
)

var (
	podAdmissionControl *pod.PodAdmissionControl
)

type WebhookServer struct {
	// http server
	server *http.Server
}

func NewWebHookServer(kubeCli kubernetes.Interface, operatorCli versioned.Interface, informerFactory informers.SharedInformerFactory, kubeInformerFactory kubeinformers.SharedInformerFactory, certFile, keyFile string) *WebhookServer {

	sCert, err := util.ConfigTLS(certFile, keyFile)

	if err != nil {
		glog.Fatalf("failed to create scert file %v", err)
	}

	server := &http.Server{
		Addr:      ":443",
		TLSConfig: sCert,
	}

	// init pdControl
	pdControl := pdapi.NewDefaultPDControl(kubeCli)

	// init recorder
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&eventv1.EventSinkImpl{
		Interface: eventv1.New(kubeCli.CoreV1().RESTClient()).Events("")})
	recorder := eventBroadcaster.NewRecorder(v1alpha1.Scheme, corev1.EventSource{Component: "tidbcluster"})

	podAdmissionControl = pod.NewPodAdmissionControl(kubeCli, operatorCli, pdControl, informerFactory, kubeInformerFactory, recorder)

	http.HandleFunc("/statefulsets", ServeStatefulSets)
	http.HandleFunc("/pods", ServePods)

	return &WebhookServer{
		server: server,
	}
}

func (ws *WebhookServer) Run() error {
	glog.Info("webhook sever start running")
	return ws.server.ListenAndServeTLS("", "")
}

func (ws *WebhookServer) Shutdown() error {
	return ws.server.Shutdown(nil)
}
