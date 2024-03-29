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

package main

import (
	"crypto/tls"
	goflag "flag"
	"io/ioutil"
	"net/http"

	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/tests/pkg/webhook"
	"github.com/pingcap/tidb-operator/tests/third_party/k8s/log"
	flag "github.com/spf13/pflag"
	"k8s.io/apiserver/pkg/server/healthz"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/component-base/logs"
)

var (
	optKubeconfig      string
	optWatchNamespaces []string
	optKey             string
	optCert            string
)

func init() {
	flag.StringVar(&optKubeconfig, "kubeconfig", optKubeconfig, "path to a kubeconfig file")
	flag.StringArrayVar(&optWatchNamespaces, "watch-namespaces", optWatchNamespaces, "namespaces to watch")
	flag.StringVar(&optCert, "cert", optCert, "server cert")
	flag.StringVar(&optKey, "key", optKey, "server key")
}

func main() {
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	flag.Parse()
	logs.InitLogs()
	defer logs.FlushLogs()

	kubeConfig, err := clientcmd.BuildConfigFromFlags("", optKubeconfig)
	if err != nil {
		log.Failf(err.Error())
	}

	kubeCli, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		log.Failf(err.Error())
	}

	versionedCli, err := versioned.NewForConfig(kubeConfig)
	if err != nil {
		log.Failf(err.Error())
	}

	certBytes, err := ioutil.ReadFile(optCert)
	if err != nil {
		log.Failf(err.Error())
	}

	keyBytes, err := ioutil.ReadFile(optKey)
	if err != nil {
		log.Failf(err.Error())
	}

	cert, err := tls.X509KeyPair(certBytes, keyBytes)
	if err != nil {
		log.Failf(err.Error())
	}

	wh := webhook.NewWebhook(kubeCli, versionedCli, optWatchNamespaces)
	http.HandleFunc("/pods", wh.ServePods)
	server := &http.Server{
		Addr: ":443",
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
	}
	healthz.InstallHandler(http.DefaultServeMux)
	if err := server.ListenAndServeTLS("", ""); err != nil {
		log.Failf(err.Error())
	}
}
