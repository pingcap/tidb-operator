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

package util

import (
	"crypto/tls"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func GetNewClient() (versioned.Interface, kubernetes.Interface, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		glog.Errorf("failed to get config: %v", err)
		return nil, nil, err
	}

	cli, err := versioned.NewForConfig(cfg)
	if err != nil {
		glog.Errorf("failed to create Clientset: %v", err)
		return nil, nil, err
	}

	kubeCli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Errorf("failed to get kubernetes Clientset: %v", err)
		return nil, nil, err
	}
	return cli, kubeCli, nil
}

// toAdmissionResponse is a helper function to create an AdmissionResponse
// with an embedded error
func ARFail(err error) *v1beta1.AdmissionResponse {
	return &v1beta1.AdmissionResponse{
		Allowed: false,
		Result: &metav1.Status{
			Message: err.Error(),
			Reason:  metav1.StatusReasonNotAcceptable,
		},
	}
}

// toAdmissionResponse return allow to action
func ARSuccess() *v1beta1.AdmissionResponse {
	return &v1beta1.AdmissionResponse{
		Allowed: true,
	}
}

// config tls cert for server
func ConfigTLS(cert []byte, key []byte) (*tls.Config, error) {
	sCert, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{sCert},
	}, nil
}
