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
	encodedConfig := "Z2xvYmFsOgogIGV2YWx1YXRpb25faW50ZXJ2YWw6IDE1cwogIHNjcmFwZV9pbnRlcnZhbDogMTVzCnJ1bGVfZmlsZXM6Ci0gL3Byb21ldGhldXMtcnVsZXMvcnVsZXMvKi5ydWxlcy55bWwKc2NyYXBlX2NvbmZpZ3M6Ci0gaG9ub3JfbGFiZWxzOiB0cnVlCiAgam9iX25hbWU6IHRpZGItY2x1c3RlcgogIGt1YmVybmV0ZXNfc2RfY29uZmlnczoKICAtIHJvbGU6IHBvZAogIHJlbGFiZWxfY29uZmlnczoKICAtIGFjdGlvbjoga2VlcAogICAgcmVnZXg6IHJlZ2V4CiAgICBzb3VyY2VfbGFiZWxzOiBbX19tZXRhX2t1YmVybmV0ZXNfcG9kX2xhYmVsX2FwcF9rdWJlcm5ldGVzX2lvX2luc3RhbmNlXQogIC0gYWN0aW9uOiBrZWVwCiAgICByZWdleDogInRydWUiCiAgICBzb3VyY2VfbGFiZWxzOiBbX19tZXRhX2t1YmVybmV0ZXNfcG9kX2Fubm90YXRpb25fcHJvbWV0aGV1c19pb19zY3JhcGVdCiAgLSBhY3Rpb246IHJlcGxhY2UKICAgIHJlZ2V4OiAoLispCiAgICBzb3VyY2VfbGFiZWxzOiBbX19tZXRhX2t1YmVybmV0ZXNfcG9kX2Fubm90YXRpb25fcHJvbWV0aGV1c19pb19wYXRoXQogICAgdGFyZ2V0X2xhYmVsOiBfX21ldHJpY3NfcGF0aF9fCiAgLSBhY3Rpb246IHJlcGxhY2UKICAgIHJlZ2V4OiAoW146XSspKD86OlxkKyk/OyhcZCspCiAgICByZXBsYWNlbWVudDogJDE6JDIKICAgIHNvdXJjZV9sYWJlbHM6IFtfX2FkZHJlc3NfXywgX19tZXRhX2t1YmVybmV0ZXNfcG9kX2Fubm90YXRpb25fcHJvbWV0aGV1c19pb19wb3J0XQogICAgdGFyZ2V0X2xhYmVsOiBfX2FkZHJlc3NfXwogIC0gYWN0aW9uOiByZXBsYWNlCiAgICBzb3VyY2VfbGFiZWxzOiBbX19tZXRhX2t1YmVybmV0ZXNfbmFtZXNwYWNlXQogICAgdGFyZ2V0X2xhYmVsOiBrdWJlcm5ldGVzX3BvZF9pcAogIC0gYWN0aW9uOiByZXBsYWNlCiAgICBzb3VyY2VfbGFiZWxzOiBbX19tZXRhX2t1YmVybmV0ZXNfcG9kX25hbWVdCiAgICB0YXJnZXRfbGFiZWw6IGluc3RhbmNlCiAgLSBhY3Rpb246IHJlcGxhY2UKICAgIHNvdXJjZV9sYWJlbHM6IFtfX21ldGFfa3ViZXJuZXRlc19wb2RfbGFiZWxfYXBwX2t1YmVybmV0ZXNfaW9faW5zdGFuY2VdCiAgICB0YXJnZXRfbGFiZWw6IGNsdXN0ZXIKICBzY3JhcGVfaW50ZXJ2YWw6IDE1cwogIHRsc19jb25maWc6CiAgICBpbnNlY3VyZV9za2lwX3ZlcmlmeTogdHJ1ZQo="
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
