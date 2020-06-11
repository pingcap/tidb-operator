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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/manager/member"
	"k8s.io/klog"
)

func buildUrl(tcName string, tlsEnabled bool) (*url.URL, error) {
	url := &url.URL{
		Host:   fmt.Sprintf("%s-pd:2379", tcName),
		Scheme: "http",
	}

	if tlsEnabled {
		url.Scheme = "https"
	}
	return url, nil
}

func buildProxy(tcName string, tlsEnabled bool) (*httputil.ReverseProxy, error) {
	url, err := buildUrl(tcName, tlsEnabled)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	proxy := httputil.NewSingleHostReverseProxy(url)
	if tlsEnabled {
		// load crt and key
		certPath := fmt.Sprintf("%s/tls.crt", member.PdTlsCertPath)
		keyPath := fmt.Sprintf("%s/tls.key", member.PdTlsCertPath)
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			klog.Error(err)
			return nil, err
		}
		// load ca
		rootCAs := x509.NewCertPool()
		caPath := fmt.Sprintf("%s/ca.crt", member.PdTlsCertPath)
		caByte, err := ioutil.ReadFile(caPath)
		if err != nil {
			klog.Error(err)
			return nil, err
		}
		rootCAs.AppendCertsFromPEM(caByte)
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      rootCAs,
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
	return proxy, nil
}

type handler struct {
	tcName       string
	namespace    string
	tcTlsEnabled bool
}

func NewHandler(tcName string, tcTlsEnabled bool) *handler {
	return &handler{
		tcName:       tcName,
		tcTlsEnabled: tcTlsEnabled,
	}
}

func (handler *handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	proxy, err := buildProxy(handler.tcName, handler.tcTlsEnabled)
	if err != nil {
		msg := fmt.Sprintf("Error Happed, err:%v", err)
		w.Write([]byte(msg))
		return
	}
	proxy.ServeHTTP(w, req)
}

func StartProxyServer(tcName string, tcTlsEnabled bool, port int) {
	handler := NewHandler(tcName, tcTlsEnabled)
	klog.Infof("start proxy-server")
	klog.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), handler))
}
