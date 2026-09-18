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
	"os/exec"
	"regexp"
	"strings"
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
			expect:   "s3://tici-test/tici_default_prefix/cdc?endpoint=http://minio-service:9000&access-key=minio&secret-access-key=minio-secret&provider=minio&protocol=canal-json&enable-tidb-extension=true&output-row-key=true&use-table-id-as-path=true",
		},
		{
			name:     "gcs endpoint uses gcs sink uri",
			endpoint: "https://storage.googleapis.com",
			expect:   "gcs://tici-test/tici_default_prefix/cdc?protocol=canal-json&enable-tidb-extension=true&output-row-key=true&use-table-id-as-path=true",
		},
		{
			name:     "gcs endpoint without scheme uses gcs sink uri",
			endpoint: "storage.googleapis.com",
			expect:   "gcs://tici-test/tici_default_prefix/cdc?protocol=canal-json&enable-tidb-extension=true&output-row-key=true&use-table-id-as-path=true",
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

func TestRenderTiCDCStartScriptTiCIChangefeedBootstrap(t *testing.T) {
	tc := newTiCIChangefeedTestCluster("http://minio-service:9000", "minio", "minio-secret")
	tc.Name = "tici-test"
	tc.Namespace = "tici-test-ns"
	tc.Spec.TiCDC = &v1alpha1.TiCDCSpec{}

	script, err := RenderTiCDCStartScript(tc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := validateScript(script); err != nil {
		t.Fatalf("rendered script is not valid shell: %v", err)
	}
	for _, want := range []string{
		"--newarch=true",
		"--no-confirm",
		"timeout 10 /cdc cli capture list",
		"timeout 10 /cdc cli changefeed query",
		"redact()",
		"<REDACTED>",
		`redact < "${CHANGEFEED_LOG}" | tail -5`,
		"exit 1",
		`echo "tici: changefeed ${CHANGEFEED_ID} already exists, skip creation"`,
		`echo "tici: changefeed ${CHANGEFEED_ID} created"`,
		`tici: creating changefeed ${CHANGEFEED_ID} (attempt`,
		`echo "tici: failed to bootstrap changefeed ${CHANGEFEED_ID} after 15 attempts; exiting to trigger pod restart"`,
		"wait ${CDC_PID}",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("rendered script does not contain %q:\n%s", want, script)
		}
	}
	if strings.Contains(script, ">/dev/null") {
		t.Errorf("rendered script should not silently discard changefeed CLI output:\n%s", script)
	}
}

func TestRenderTiCDCStartScriptTiCIChangefeedRedactSecrets(t *testing.T) {
	tc := newTiCIChangefeedTestCluster("http://minio-service:9000", "minio", "minio-secret")
	tc.Name = "tici-test"
	tc.Namespace = "tici-test-ns"
	tc.Spec.TiCDC = &v1alpha1.TiCDCSpec{}

	script, err := RenderTiCDCStartScript(tc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	fn := regexp.MustCompile(`(?s)redact\(\) \{.*?\n\}`).FindString(script)
	if fn == "" {
		t.Fatalf("rendered script does not contain the redact function:\n%s", script)
	}

	// The TiCDC CLI echoes the raw sink uri in parse errors, e.g.
	// `parse "s3://...&access-key=...&secret-access-key=...": invalid URL escape`.
	// The redact function must mask the credentials before the error is logged.
	sample := `Error: parse "s3://mybucket/tici_default_prefix/cdc?endpoint=http://minio-service:9000&access-key=minio&secret-access-key=minio-secret&provider=minio&protocol=canal-json&enable-tidb-extension=true&output-row-key=true&use-table-id-as-path=true": invalid URL escape "%zz"` + "\n"
	cmd := exec.Command("sh", "-c", fn+"\nredact")
	cmd.Stdin = strings.NewReader(sample)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("running the redact function failed: %v", err)
	}
	for _, secret := range []string{"access-key=minio", "secret-access-key=minio-secret", "minio-secret"} {
		if strings.Contains(string(out), secret) {
			t.Errorf("redact output still contains %q:\n%s", secret, out)
		}
	}
	for _, want := range []string{"access-key=<REDACTED>", "secret-access-key=<REDACTED>"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("redact output does not contain %q:\n%s", want, out)
		}
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
