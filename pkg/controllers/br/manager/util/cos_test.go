// Copyright 2026 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
)

type cosTransport func(*http.Request) (*http.Response, error)

func (f cosTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestCOSMetadataAddressing(t *testing.T) {
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })
	var methods []string
	http.DefaultTransport = cosTransport(func(r *http.Request) (*http.Response, error) {
		methods = append(methods, r.Method)
		code := http.StatusOK
		body := "metadata"
		if r.URL.Host != "backup-1234567890.cos.ap-beijing.myqcloud.com" || r.URL.Path != "/full/backupmeta" {
			code = http.StatusForbidden
			body = "<Error><Code>PathStyleDomainForbidden</Code></Error>"
		}
		if r.Header.Get("Authorization") == "" {
			t.Error("request is not signed")
		}
		return &http.Response{StatusCode: code, Header: http.Header{"Content-Length": []string{"8"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})
	backend, err := NewStorageBackend(v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{
		Provider: "aws", Region: "ap-beijing", Endpoint: "https://cos.ap-beijing.myqcloud.com", Bucket: "backup-1234567890", Prefix: "/full/",
	}}, &StorageCredential{awsCred: credentials.NewStaticCredentials("test-id", "test-secret", "")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := backend.Close(); err != nil {
			t.Errorf("close storage backend: %v", err)
		}
	})
	exists, err := backend.Exists(context.Background(), "backupmeta")
	if err != nil || !exists {
		t.Fatalf("metadata HEAD failed: exists=%v err=%v", exists, err)
	}
	data, err := backend.ReadAll(context.Background(), "backupmeta")
	if err != nil || string(data) != "metadata" {
		t.Fatalf("metadata GET failed: %v", err)
	}
	if strings.Join(methods, ",") != "HEAD,GET" {
		t.Fatalf("unexpected methods: %v", methods)
	}
}

func TestS3AddressingCompatibility(t *testing.T) {
	for _, tt := range []struct {
		provider, endpoint string
		pathStyle          bool
	}{
		{"aws", "https://cos.ap-beijing.myqcloud.com", false},
		{"aws", "https://cos.ap-shanghai.myqcloud.com/", false},
		{"tencent", "https://cos.ap-beijing.myqcloud.com", false},
		{"aws", "https://s3.us-east-1.amazonaws.com", true},
		{"ceph", "http://minio:9000", true},
		{"aws", "https://cos.ap-beijing.myqcloud.com.evil.example", true},
		{"aws", "https://example.com/cos.ap-beijing.myqcloud.com", true},
		{"aws", "", true},
		{"alibaba", "https://oss-cn-beijing.aliyuncs.com", false},
		{"netease", "https://example.com", false},
	} {
		t.Run(tt.provider+tt.endpoint, func(t *testing.T) {
			cfg := makeS3Config(&v1alpha1.S3StorageProvider{Provider: v1alpha1.S3StorageProviderType(tt.provider), Endpoint: tt.endpoint}, false)
			if cfg.forcePathStyle != tt.pathStyle {
				t.Fatalf("path-style=%v, want %v", cfg.forcePathStyle, tt.pathStyle)
			}
		})
	}
}
