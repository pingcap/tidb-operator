// Copyright 2020 PingCAP, Inc.
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

package server

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

func buildUrl(cli versioned.Interface, tcName, namespace string) (*url.URL, error) {
	url := &url.URL{
		Host:   fmt.Sprintf("%s-pd:2379", tcName),
		Scheme: "http",
	}
	tc, err := cli.PingcapV1alpha1().TidbClusters(namespace).Get(tcName, metav1.GetOptions{})
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	if tc.IsTLSClusterEnabled() {
		url.Scheme = "https"
	}
	return url, nil
}

func buildProxy(cli versioned.Interface, kubeCli kubernetes.Interface, tcName, namespace string) *httputil.ReverseProxy {
	url, err := buildUrl(cli, tcName, namespace)
	if err != nil {
		klog.Fatal(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(url)
	if url.Scheme == "https" {
		tlsConfig, err := pdapi.GetTLSConfig(kubeCli, pdapi.Namespace(namespace), tcName, nil)
		if err != nil {
			klog.Fatal(err)
		}
		proxy.Transport = &http.Transport{TLSClientConfig: tlsConfig}
	}
	director := proxy.Director
	proxy.Director = func(req *http.Request) {
		if strings.HasPrefix(req.RequestURI, "/dashboard") {
			director(req)
			req.Host = req.URL.Host
		}
	}
	return proxy
}

func StartProxyServer(cli versioned.Interface, kubeCli kubernetes.Interface, tcName, namespace string, port int) {
	proxy := buildProxy(cli, kubeCli, tcName, namespace)
	klog.Infof("start proxy-server")
	klog.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), proxy))
}
