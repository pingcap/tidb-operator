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

package monitor

import (
	"encoding/base64"
	. "github.com/onsi/gomega"
	"testing"
)

func TestRenderPrometheusConfig(t *testing.T) {
	g := NewGomegaWithT(t)
	encodedConfig := "Z2xvYmFsOgogIGV2YWx1YXRpb25faW50ZXJ2YWw6IDE1cwogIHNjcmFwZV9pbnRlcnZhbDogMTVzCnJ1bGVfZmlsZXM6Ci0gL3Byb21ldGhldXMtcnVsZXMvcnVsZXMvKi5ydWxlcy55bWwKc2NyYXBlX2NvbmZpZ3M6Ci0gaG9ub3JfbGFiZWxzOiB0cnVlCiAgam9iX25hbWU6IHRpZGItY2x1c3RlcgogIGt1YmVybmV0ZXNfc2RfY29uZmlnczoKICAtIG5hbWVzcGFjZXM6CiAgICAgIG5hbWVzOgogICAgICAtIG5zMQogICAgICAtIG5zMgogICAgcm9sZTogcG9kCiAgcmVsYWJlbF9jb25maWdzOgogIC0gYWN0aW9uOiBrZWVwCiAgICByZWdleDogcmVnZXgKICAgIHNvdXJjZV9sYWJlbHM6IFtfX21ldGFfa3ViZXJuZXRlc19wb2RfbGFiZWxfYXBwX2t1YmVybmV0ZXNfaW9faW5zdGFuY2VdCiAgLSBhY3Rpb246IGtlZXAKICAgIHJlZ2V4OiAidHJ1ZSIKICAgIHNvdXJjZV9sYWJlbHM6IFtfX21ldGFfa3ViZXJuZXRlc19wb2RfYW5ub3RhdGlvbl9wcm9tZXRoZXVzX2lvX3NjcmFwZV0KICAtIGFjdGlvbjogcmVwbGFjZQogICAgcmVnZXg6ICguKykKICAgIHNvdXJjZV9sYWJlbHM6IFtfX21ldGFfa3ViZXJuZXRlc19wb2RfYW5ub3RhdGlvbl9wcm9tZXRoZXVzX2lvX3BhdGhdCiAgICB0YXJnZXRfbGFiZWw6IF9fbWV0cmljc19wYXRoX18KICAtIGFjdGlvbjogcmVwbGFjZQogICAgcmVnZXg6IChbXjpdKykoPzo6XGQrKT87KFxkKykKICAgIHJlcGxhY2VtZW50OiAkMTokMgogICAgc291cmNlX2xhYmVsczogW19fYWRkcmVzc19fLCBfX21ldGFfa3ViZXJuZXRlc19wb2RfYW5ub3RhdGlvbl9wcm9tZXRoZXVzX2lvX3BvcnRdCiAgICB0YXJnZXRfbGFiZWw6IF9fYWRkcmVzc19fCiAgLSBhY3Rpb246IHJlcGxhY2UKICAgIHNvdXJjZV9sYWJlbHM6IFtfX21ldGFfa3ViZXJuZXRlc19uYW1lc3BhY2VdCiAgICB0YXJnZXRfbGFiZWw6IGt1YmVybmV0ZXNfcG9kX2lwCiAgLSBhY3Rpb246IHJlcGxhY2UKICAgIHNvdXJjZV9sYWJlbHM6IFtfX21ldGFfa3ViZXJuZXRlc19wb2RfbmFtZV0KICAgIHRhcmdldF9sYWJlbDogaW5zdGFuY2UKICAtIGFjdGlvbjogcmVwbGFjZQogICAgc291cmNlX2xhYmVsczogW19fbWV0YV9rdWJlcm5ldGVzX3BvZF9sYWJlbF9hcHBfa3ViZXJuZXRlc19pb19pbnN0YW5jZV0KICAgIHRhcmdldF9sYWJlbDogY2x1c3RlcgogIHNjcmFwZV9pbnRlcnZhbDogMTVzCiAgdGxzX2NvbmZpZzoKICAgIGluc2VjdXJlX3NraXBfdmVyaWZ5OiB0cnVlCg=="
	model := &MonitorConfigModel{
		ReleaseTargetRegex: "regex",
		ReleaseNamespaces: []string{
			"ns1",
			"ns2",
		},
		EnableTLSCluster: false,
	}
	content, err := RenderPrometheusConfig(model)
	g.Expect(err).NotTo(HaveOccurred())
	encodedContent := base64.StdEncoding.EncodeToString([]byte(content))
	g.Expect(encodedContent).Should(Equal(encodedConfig))
}
