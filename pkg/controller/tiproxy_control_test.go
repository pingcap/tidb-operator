// Copyright 2024 PingCAP, Inc.
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

package controller

import (
	"io"
	"net/http"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
)

func TestSetTiProxyLabels(t *testing.T) {
	g := NewGomegaWithT(t)

	cases := []struct {
		caseName string
		labels   map[string]string
		failed   bool
		body     string
	}{
		{
			caseName: "SetLabelsSuccessfully",
			failed:   false,
			labels:   map[string]string{"zone": "test1"},
			body:     "[labels]\n  zone = \"test1\"\n",
		},
		{
			caseName: "SetLabelsFailed",
			failed:   true,
			labels:   map[string]string{"zone": "test1", "topology.kubernetes.io/zone": "test1"},
			body:     "[labels]\n  \"topology.kubernetes.io/zone\" = \"test1\"\n  zone = \"test1\"\n",
		},
	}

	for _, c := range cases {
		svc := getClientServer(func(w http.ResponseWriter, request *http.Request) {
			g.Expect(request.Method).To(Equal(http.MethodPut), "check method")
			g.Expect(request.URL.Path).To(Equal("/api/admin/config/"), "check url")
			body, err := io.ReadAll(request.Body)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(string(body)).To(Equal(c.body))

			w.Header().Set("Content-Type", ContentTypeJSON)
			if c.failed {
				w.WriteHeader(http.StatusInternalServerError)
			}
		})
		defer svc.Close()

		control := NewDefaultTiProxyControl()
		url := svc.URL
		if idx := strings.Index(url, "://"); idx >= 0 {
			url = url[idx+3:]
		}
		control.testURL = url
		tc := getTidbCluster()
		tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: false}
		err := control.SetLabels(tc, 0, c.labels)
		if c.failed {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}
	}
}
