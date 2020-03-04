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

package httputil

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"k8s.io/klog"
)

const (
	k8sCAFile  = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	clientCert = "/var/lib/tls/client.crt"
	clientKey  = "/var/lib/tls/client.key"
)

// DeferClose captures and prints the error from closing (if an error occurs).
// This is designed to be used in a defer statement.
func DeferClose(c io.Closer) {
	if err := c.Close(); err != nil {
		klog.Error(err)
	}
}

// ReadErrorBody in the error case ready the body message.
// But return it as an error (or return an error from reading the body).
func ReadErrorBody(body io.Reader) (err error) {
	bodyBytes, err := ioutil.ReadAll(body)
	if err != nil {
		return err
	}
	return fmt.Errorf(string(bodyBytes))
}

// GetBodyOK returns the body or an error if the response is not okay
func GetBodyOK(httpClient *http.Client, apiURL string) ([]byte, error) {
	res, err := httpClient.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer DeferClose(res.Body)
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 400 {
		errMsg := fmt.Errorf("Error response %v URL %s,body response: %s", res.StatusCode, apiURL, string(body[:]))
		return nil, errMsg
	}
	return body, err
}
