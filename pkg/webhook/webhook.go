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
	"time"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/webhook/route"
	"github.com/pingcap/tidb-operator/pkg/webhook/util"
	admissionV1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type WebhookServer struct {
	// kubernetes client interface
	KubeCli kubernetes.Interface
	// operator client interface
	Cli versioned.Interface
	// cert context ca.crt key server.crt
	Context *util.CertContext
	// configuration name
	ConfigName string
	// http server
	Server *http.Server
}

func NewWebHookServer(kubecli kubernetes.Interface, cli versioned.Interface, context *util.CertContext) *WebhookServer {

	http.HandleFunc("/statefulsets", route.ServeStatefulSets)

	sCert, err := util.ConfigTLS(context.Cert, context.Key)

	if err != nil {
		glog.Fatalf("failed to create scert file %v", err)
	}

	server := &http.Server{
		Addr:      ":443",
		TLSConfig: sCert,
	}

	return &WebhookServer{
		KubeCli:    kubecli,
		Cli:        cli,
		Context:    context,
		ConfigName: "validating-webhook-configuration",
		Server:     server,
	}
}

func (ws *WebhookServer) Run() error {
	return ws.Server.ListenAndServeTLS("", "")
}

func (ws *WebhookServer) Shutdown() error {
	return ws.Server.Shutdown(nil)
}

func strPtr(s string) *string { return &s }

func (ws *WebhookServer) RegisterWebhook(namespace string, svcName string) error {

	policyFail := admissionV1beta1.Fail

	_, err := ws.KubeCli.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Create(&admissionV1beta1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: ws.ConfigName,
		},
		Webhooks: []admissionV1beta1.Webhook{
			{
				Name: "admit-statefulset-webhook.k8s.io",
				Rules: []admissionV1beta1.RuleWithOperations{{
					Operations: []admissionV1beta1.OperationType{admissionV1beta1.Update},
					Rule: admissionV1beta1.Rule{
						APIGroups:   []string{"apps"},
						APIVersions: []string{"v1beta1"},
						Resources:   []string{"statefulsets"},
					},
				}},
				ClientConfig: admissionV1beta1.WebhookClientConfig{
					Service: &admissionV1beta1.ServiceReference{
						Namespace: namespace,
						Name:      svcName,
						Path:      strPtr("/statefulsets"),
					},
					CABundle: ws.Context.SigningCert,
				},
				FailurePolicy: &policyFail,
			},
		},
	})

	if err != nil {
		glog.Errorf("registering webhook config %s with namespace %s error %v", ws.ConfigName, namespace, err)
		return err
	}

	// The webhook configuration is honored in 10s.
	time.Sleep(10 * time.Second)

	return nil
}

func (ws *WebhookServer) UnregisterWebhook() error {
	err := ws.KubeCli.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Delete(ws.ConfigName, nil)
	if err != nil && !errors.IsNotFound(err) {
		glog.Errorf("failed to delete webhook config %v", err)
	}
	return nil
}
