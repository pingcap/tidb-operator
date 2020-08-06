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

package query

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/util/crypto"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

type ExternalResponse struct {
	Name                string `json:"name"`
	Namespace           string `json:"namespace"`
	Type                string `json:"type"`
	RecommendedReplicas int32  `json:"recommendedReplicas"`
}

const (
	defaultTimeout = 5 * time.Second
)

func ExternalService(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType, endpoint *v1alpha1.ExternalEndpoint, kubecli kubernetes.Interface) (int32, error) {
	bytes, err := sendRequest(tc, memberType, endpoint, kubecli)
	if err != nil {
		return -1, err
	}
	resp := &ExternalResponse{}
	err = json.Unmarshal(bytes, resp)
	if err != nil {
		return -1, err
	}
	if resp.Name != tc.Name || resp.Namespace != tc.Namespace || resp.Type != memberType.String() {
		return -1, fmt.Errorf("external endpoint returns unexpected info, get %#v, expect %s/%s/%s", resp, tc.Namespace, tc.Name, memberType)
	}
	return resp.RecommendedReplicas, nil
}

func sendRequest(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType, endpoint *v1alpha1.ExternalEndpoint, kubecli kubernetes.Interface) ([]byte, error) {
	client, err := getClient(endpoint, kubecli)
	if err != nil {
		return nil, err
	}
	scheme := "http"
	if endpoint.TLSSecret != nil {
		scheme = "https"
	}
	url := fmt.Sprintf("%s://%s:%d%s?name=%s&namespace=%s&type=%s", scheme, endpoint.Host, endpoint.Port, endpoint.Path, tc.Name, tc.Namespace, memberType.String())
	r, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	bytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("query from external endpoint [%s] failed, response: %v, status code: %v", url, string(bytes), r.StatusCode)
	}
	return bytes, nil
}

func getClient(endpoint *v1alpha1.ExternalEndpoint, kubecli kubernetes.Interface) (*http.Client, error) {
	var client *http.Client
	if endpoint.TLSSecret != nil {
		tlsConfig, err := loadTLSConfig(endpoint, kubecli)
		if err != nil {
			return nil, err
		}
		tr := &http.Transport{
			TLSClientConfig:   tlsConfig,
			DisableKeepAlives: true,
		}
		client = &http.Client{
			Timeout:   defaultTimeout,
			Transport: tr,
		}
	} else {
		client = &http.Client{
			Timeout: defaultTimeout,
		}
	}
	return client, nil
}

func loadTLSConfig(endpoint *v1alpha1.ExternalEndpoint, kubecli kubernetes.Interface) (*tls.Config, error) {
	secret, err := kubecli.CoreV1().Secrets(endpoint.TLSSecret.Namespace).Get(endpoint.TLSSecret.Name, metav1.GetOptions{})
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	return crypto.LoadTlsConfigFromSecret(secret)
}
