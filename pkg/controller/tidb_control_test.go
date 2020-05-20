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

package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"
)

const (
	ContentTypeJSON string = "application/json"
	caData          string = `
-----BEGIN CERTIFICATE-----
MIIDrDCCApSgAwIBAgIUbd5rgQbs0yXA0lrdMeDG3ot1xhswDQYJKoZIhvcNAQEL
BQAwXDELMAkGA1UEBhMCVVMxEDAOBgNVBAgTB0JlaWppbmcxCzAJBgNVBAcTAkNB
MRAwDgYDVQQKEwdQaW5nQ0FQMQ0wCwYDVQQLEwRUaURCMQ0wCwYDVQQDEwRUaURC
MB4XDTIwMDUxODEzMDQwMFoXDTI1MDUxNzEzMDQwMFowXDELMAkGA1UEBhMCVVMx
EDAOBgNVBAgTB0JlaWppbmcxCzAJBgNVBAcTAkNBMRAwDgYDVQQKEwdQaW5nQ0FQ
MQ0wCwYDVQQLEwRUaURCMQ0wCwYDVQQDEwRUaURCMIIBIjANBgkqhkiG9w0BAQEF
AAOCAQ8AMIIBCgKCAQEA5hNTcyql8kNccZqPnbphPlfAREwzZ44FOLnTZKwFgAAL
fnoZrTnCupCZZTmkKFwJrpe9h20e/YzFRHe08XYRLC5FtOx/GcbKPdfXTrQeH/yP
nRAwJiL9+uuyjNAOBFl/D42kwaSRlhRCyGB2l9E2/LxHQIsC0o644jVf8R7wI3Or
DT58LaY/Ls5t++scffPqCrDPvtBycgSnXgxiN79zG86MjsWi37hzZ3bMtm5M3IL7
0MmeVPYaAa0DrUJEITMl+HOrZJgdtni6HIwhmmlJcyGvKgsJMEBFD3nbbZeHuA0b
YEuM4SPneyt0pfd+5PpKvWyyMxv22agu18GI2eWBLQIDAQABo2YwZDAOBgNVHQ8B
Af8EBAMCAQYwEgYDVR0TAQH/BAgwBgEB/wIBAjAdBgNVHQ4EFgQUZZ3644otjnxA
yR5qJQqc55JABsUwHwYDVR0jBBgwFoAUZZ3644otjnxAyR5qJQqc55JABsUwDQYJ
KoZIhvcNAQELBQADggEBAJFM6gzsZHKvEN/fjAs3hcawT/Rvm1m0HF5fQMNz9qFz
wJty8Z9l2NN2jqHvV20gByMV/CMsecvmPM6my01c1SEewnMEJ427Qlmw9cjT0sZY
N25C9IQQsiPtEPuhIBla/EEVgJI97BsKhDlZv+M4Y6rXZM7gDz4233jG0pG0b21i
aDToY4Yuv23b/2HvmC2vbthqQ79k8CkObIc0zHCtneu14cTtDiXZb/oWB118UUoh
6YL9bweBSkti8srFUi/jzH+UbsTkCfzYlY+xe2b3A2lgTkbA8SdSxzfGyYWCiDg3
AVtszHCdFeRCInTt0md3rZb7AR9onqkCmRdOVXLXuVc=
-----END CERTIFICATE-----
`
	certData string = `
-----BEGIN CERTIFICATE-----
MIIDBzCCAe+gAwIBAgIUJRIgZmjB0oCRS26oScdxQLqJCoUwDQYJKoZIhvcNAQEL
BQAwXDELMAkGA1UEBhMCVVMxEDAOBgNVBAgTB0JlaWppbmcxCzAJBgNVBAcTAkNB
MRAwDgYDVQQKEwdQaW5nQ0FQMQ0wCwYDVQQLEwRUaURCMQ0wCwYDVQQDEwRUaURC
MB4XDTIwMDUxODEzMDQwMFoXDTIxMDUxODEzMDQwMFowSDELMAkGA1UEBhMCVVMx
FjAUBgNVBAgTDVNhbiBGcmFuY2lzY28xCzAJBgNVBAcTAkNBMRQwEgYDVQQDEwtl
eGFtcGxlLm5ldDBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABMvDGev6J0DIaCqK
Fb4yrxRxlURbfk64yQm4bAgMemV23U4JrDOz/JLJctmhr7krAz5bljKbf3XD8fja
ZgF1e6SjgZ8wgZwwDgYDVR0PAQH/BAQDAgWgMBMGA1UdJQQMMAoGCCsGAQUFBwMC
MAwGA1UdEwEB/wQCMAAwHQYDVR0OBBYEFFT0RT/TccSXVGLom1mDYEpiQFOTMB8G
A1UdIwQYMBaAFGWd+uOKLY58QMkeaiUKnOeSQAbFMCcGA1UdEQQgMB6CC2V4YW1w
bGUubmV0gg93d3cuZXhhbXBsZS5uZXQwDQYJKoZIhvcNAQELBQADggEBADoX65Sb
0GWwiak/x4SNVJRx/Ktj3Hl9nNKsvbGh/3BMgBO6P8M7lrwmtIg5TJtO2lnYOjVr
yk0wi1lGEkbWkCPPXhYx8d9aYcNBLz6CAbSKzG6OX6OCUnKogVf4D44+6b6VgfqP
Ja4yJOe6C1pe9dDMh7VAcZyKFUcEkp4Klheh96PY6seDrzT/kRCTYA7X9tfIEFTO
Qcu9ZVAsypVgOUc1pZJGPED1oItx24V2ZX9E1SM/8tQcRt2s/3ah+LWQV8lpVXCi
o8O7UTMyQ7MUrPusaqsG/QuvppbdahOLzkVc0E5jUOL/dgSxsdOqc7EIxd94Cg65
cQQSTMrQTbQLo5c=
-----END CERTIFICATE-----
`
	keyData string = `
-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIFbUNEYv0ujI3dTLbnb5lTBfRxwst3lMROmRV2tN7NTroAoGCCqGSM49
AwEHoUQDQgAEy8MZ6/onQMhoKooVvjKvFHGVRFt+TrjJCbhsCAx6ZXbdTgmsM7P8
ksly2aGvuSsDPluWMpt/dcPx+NpmAXV7pA==
-----END EC PRIVATE KEY-----
`
)

func getClientServer(h func(http.ResponseWriter, *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(h))
}

func TestHealth(t *testing.T) {
	g := NewGomegaWithT(t)

	cases := []struct {
		caseName       string
		path           string
		method         string
		failed         bool
		healthExpected bool
	}{
		{
			caseName:       "GetHealth health",
			path:           "/status",
			method:         "GET",
			failed:         false,
			healthExpected: true,
		},
		{
			caseName:       "GetHealth not health",
			path:           "/status",
			method:         "GET",
			failed:         true,
			healthExpected: false,
		},
	}

	for _, c := range cases {
		svc := getClientServer(func(w http.ResponseWriter, request *http.Request) {
			g.Expect(request.Method).To(Equal(c.method), "check method")
			g.Expect(request.URL.Path).To(Equal(c.path), "check url")

			w.Header().Set("Content-Type", ContentTypeJSON)
			if c.failed {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		})
		defer svc.Close()

		fakeClient := &fake.Clientset{}
		tc := getTidbCluster()

		control := NewDefaultTiDBControl(fakeClient)
		control.testURL = svc.URL
		result, err := control.GetHealth(tc, 0)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(result).To(Equal(c.healthExpected))
	}
}

func TestInfo(t *testing.T) {
	g := NewGomegaWithT(t)

	cases := []struct {
		caseName string
		path     string
		method   string
		resp     DBInfo
		failed   bool
		expected *DBInfo
	}{
		{
			caseName: "GetInfo is not owner",
			path:     "/info",
			method:   "POST",
			resp:     DBInfo{IsOwner: false},
			failed:   false,
			expected: &DBInfo{IsOwner: false},
		},
		{
			caseName: "GetInfo is owner",
			path:     "/info",
			method:   "POST",
			resp:     DBInfo{IsOwner: true},
			failed:   false,
			expected: &DBInfo{IsOwner: true},
		},
		{
			caseName: "GetInfo failed",
			path:     "/info",
			method:   "POST",
			resp:     DBInfo{IsOwner: true},
			failed:   true,
		},
	}

	for _, c := range cases {
		svc := getClientServer(func(w http.ResponseWriter, request *http.Request) {
			g.Expect(request.Method).To(Equal(c.method), "check method")
			g.Expect(request.URL.Path).To(Equal(c.path), "check url")

			w.Header().Set("Content-Type", ContentTypeJSON)
			data, err := json.Marshal(c.resp)
			g.Expect(err).NotTo(HaveOccurred())

			if c.failed {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.Write(data)
			}
		})
		defer svc.Close()

		fakeClient := &fake.Clientset{}
		control := NewDefaultTiDBControl(fakeClient)
		control.testURL = svc.URL
		tc := getTidbCluster()
		result, err := control.GetInfo(tc, 0)
		if c.failed {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(result).To(Equal(c.expected))
		}
	}
}

func TestSettings(t *testing.T) {
	g := NewGomegaWithT(t)

	cases := []struct {
		caseName string
		path     string
		method   string
		failed   bool
		resp     config.Config
		expected *config.Config
	}{
		{
			caseName: "GetSettings",
			path:     "/settings",
			method:   "GET",
			failed:   false,
			resp:     config.Config{Host: "host1", Port: 1},
			expected: &config.Config{Host: "host1", Port: 1},
		},
		{
			caseName: "GetSettings",
			path:     "/settings",
			method:   "GET",
			failed:   true,
			resp:     config.Config{Host: "host2", Port: 2},
			expected: nil,
		},
	}

	for _, c := range cases {
		svc := getClientServer(func(w http.ResponseWriter, request *http.Request) {
			g.Expect(request.Method).To(Equal(c.method), "check method")
			g.Expect(request.URL.Path).To(Equal(c.path), "check url")

			w.Header().Set("Content-Type", ContentTypeJSON)
			if c.failed {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				data, err := json.Marshal(c.resp)
				g.Expect(err).NotTo(HaveOccurred())
				w.Write(data)
			}
		})
		defer svc.Close()

		fakeClient := &fake.Clientset{}
		control := NewDefaultTiDBControl(fakeClient)
		control.testURL = svc.URL
		tc := getTidbCluster()
		result, err := control.GetSettings(tc, 0)
		if c.failed {
			g.Expect(err).To(HaveOccurred())
		}
		g.Expect(result).To(Equal(c.expected))
	}
}

func TestGetHTTPClient(t *testing.T) {
	g := NewGomegaWithT(t)

	cases := []struct {
		caseName string
		updateTC func(*v1alpha1.TidbCluster)
		expected func(*http.Client)
	}{
		{
			caseName: "getHTTPClient tls not enabled",
			updateTC: func(cluster *v1alpha1.TidbCluster) {},
			expected: func(client *http.Client) {
				g.Expect(client.Transport).To(BeNil())
			},
		},
		{
			caseName: "getHTTPClient tls enabled",
			updateTC: func(cluster *v1alpha1.TidbCluster) {
				cluster.Spec.TLSCluster = &v1alpha1.TLSCluster{
					Enabled: true,
				}
			},
			expected: func(client *http.Client) {
				g.Expect(client.Transport).NotTo(BeNil())
			},
		},
	}

	for _, c := range cases {
		fakeClient := &fake.Clientset{}
		fakeSecret(fakeClient)
		control := NewDefaultTiDBControl(fakeClient)
		tc := getTidbCluster()
		c.updateTC(tc)
		httpClient, err := control.getHTTPClient(tc)
		g.Expect(err).NotTo(HaveOccurred())
		c.expected(httpClient)
	}
}

func getTidbCluster() *v1alpha1.TidbCluster {
	return &v1alpha1.TidbCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TidbCluster",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("test"),
		},
		Spec: v1alpha1.TidbClusterSpec{
			TiDB: v1alpha1.TiDBSpec{},
		},
	}
}

func fakeSecret(fakeClient *fake.Clientset) {
	fakeClient.AddReactor("get", "secrets", func(action core.Action) (bool, runtime.Object, error) {
		return true, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "ns",
			},
			Data: map[string][]byte{
				corev1.TLSCertKey:              []byte(certData),
				corev1.TLSPrivateKeyKey:        []byte(keyData),
				corev1.ServiceAccountRootCAKey: []byte(caData),
			},
		}, nil
	})
}
