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
// See the License for the specific language governing permissions and
// limitations under the License.

package v1

import (
	"testing"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
)

func TestBuildTiCIChangefeedInfoAutoGenerateSinkURI(t *testing.T) {
	cases := []struct {
		name     string
		endpoint string
		access   string
		secret   string
		expect   string
	}{
		{
			name:     "minio uses s3 sink uri",
			endpoint: "http://minio-service:9000",
			access:   "minio",
			secret:   "minio-secret",
			expect:   "s3://tici-test/tici_default_prefix/cdc?endpoint=http://minio-service:9000&access-key=minio&secret-access-key=minio-secret&provider=minio&protocol=canal-json&enable-tidb-extension=true&output-row-key=true",
		},
		{
			name:     "gcs endpoint uses gcs sink uri",
			endpoint: "https://storage.googleapis.com",
			expect:   "gcs://tici-test/tici_default_prefix/cdc?protocol=canal-json&enable-tidb-extension=true&output-row-key=true",
		},
		{
			name:     "gcs endpoint without scheme uses gcs sink uri",
			endpoint: "storage.googleapis.com",
			expect:   "gcs://tici-test/tici_default_prefix/cdc?protocol=canal-json&enable-tidb-extension=true&output-row-key=true",
		},
	}

	for _, c := range cases {
		tc := newTiCIChangefeedTestCluster(c.endpoint, c.access, c.secret)
		info, err := buildTiCIChangefeedInfo(tc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.SinkURI != c.expect {
			t.Fatalf("unexpected sink uri, want %q, got %q", c.expect, info.SinkURI)
		}
	}
}

func TestBuildTiCIChangefeedInfoSinkURIOverride(t *testing.T) {
	tc := newTiCIChangefeedTestCluster("https://storage.googleapis.com", "", "")
	tc.Spec.TiCI.Changefeed = &v1alpha1.TiCIChangefeedSpec{
		SinkURI: "gcs://custom-bucket/custom-prefix/cdc?protocol=canal-json",
	}
	info, err := buildTiCIChangefeedInfo(tc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.SinkURI != tc.Spec.TiCI.Changefeed.SinkURI {
		t.Fatalf("expected sink uri override %q, got %q", tc.Spec.TiCI.Changefeed.SinkURI, info.SinkURI)
	}
}

func TestBuildTiCIChangefeedInfoDisabledWithoutTiCI(t *testing.T) {
	tc := &v1alpha1.TidbCluster{}
	info, err := buildTiCIChangefeedInfo(tc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Enabled {
		t.Fatalf("expected changefeed disabled when TiCI is not configured")
	}
}

func TestBuildTiCIChangefeedInfoDisabledByFlag(t *testing.T) {
	disable := false
	tc := newTiCIChangefeedTestCluster("http://minio-service:9000", "minio", "minio-secret")
	tc.Spec.TiCI.Changefeed = &v1alpha1.TiCIChangefeedSpec{
		Enable: &disable,
	}
	info, err := buildTiCIChangefeedInfo(tc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Enabled {
		t.Fatalf("expected changefeed disabled when tici.changefeed.enable=false")
	}
}

func newTiCIChangefeedTestCluster(endpoint, access, secret string) *v1alpha1.TidbCluster {
	return &v1alpha1.TidbCluster{
		Spec: v1alpha1.TidbClusterSpec{
			TiCI: &v1alpha1.TiCISpec{
				S3: &v1alpha1.TiCIS3Spec{
					Endpoint:  endpoint,
					AccessKey: access,
					SecretKey: secret,
					Bucket:    "tici-test",
					Prefix:    "tici_default_prefix",
				},
			},
		},
	}
}
