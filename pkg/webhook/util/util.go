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

	admission "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ARFail is a helper function to create an AdmissionResponse
// with an embedded error
func ARFail(err error) *admission.AdmissionResponse {
	return &admission.AdmissionResponse{
		Allowed: false,
		Result: &metav1.Status{
			Message: err.Error(),
			Reason:  metav1.StatusReasonNotAcceptable,
		},
	}
}

// ARSuccess return allow to action
func ARSuccess() *admission.AdmissionResponse {
	return &admission.AdmissionResponse{
		Allowed: true,
	}
}

// config tls cert for server
func ConfigTLS(certFile string, keyFile string) (*tls.Config, error) {
	sCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{sCert},
	}, nil
}
