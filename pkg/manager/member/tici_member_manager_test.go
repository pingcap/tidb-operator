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

package member

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	tcconfig "github.com/pingcap/tidb-operator/pkg/apis/util/config"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildTiCIWorkerConfigWithCustomConfig(t *testing.T) {
	tc := newTidbClusterForTiCIConfig()
	tc.Spec.TiCI.Worker.Config = `[log]
level = "debug"`

	cfg, err := buildTiCIWorkerConfig(tc)
	if err != nil {
		t.Fatalf("build worker config failed: %v", err)
	}

	wrapper := tcconfig.New(map[string]interface{}{})
	if err := wrapper.UnmarshalTOML([]byte(cfg)); err != nil {
		t.Fatalf("worker config should be valid TOML, got err: %v, config: %s", err, cfg)
	}
	pdAddr := wrapper.Get("server.pd-addr")
	if pdAddr == nil || pdAddr.MustString() != "tici-test-pd:2379" {
		t.Fatalf("worker config should include generated server section, got: %s", cfg)
	}
	endpoint := wrapper.Get("s3.endpoint")
	if endpoint == nil || endpoint.MustString() != "http://minio-service:9000" {
		t.Fatalf("worker config should include generated s3 section, got: %s", cfg)
	}
	logLevel := wrapper.Get("log.level")
	if logLevel == nil || logLevel.MustString() != "debug" {
		t.Fatalf("worker config should merge custom log level, got: %s", cfg)
	}
}

func TestBuildTiCIMetaConfigWithCustomConfig(t *testing.T) {
	tc := newTidbClusterForTiCIConfig()
	tc.Spec.TiCI.Meta.Config = `[storage]
data-dir = "/data/tici-meta"`

	cfg, err := buildTiCIMetaConfig(tc)
	if err != nil {
		t.Fatalf("build meta config failed: %v", err)
	}

	wrapper := tcconfig.New(map[string]interface{}{})
	if err := wrapper.UnmarshalTOML([]byte(cfg)); err != nil {
		t.Fatalf("meta config should be valid TOML, got err: %v, config: %s", err, cfg)
	}
	dsns := wrapper.Get("tidb-server.dsns")
	if dsns == nil || len(dsns.MustStringSlice()) == 0 {
		t.Fatalf("meta config should include generated tidb-server section, got: %s", cfg)
	}
	if wrapper.Get("shard.max-size") == nil {
		t.Fatalf("meta config should include generated shard section, got: %s", cfg)
	}
	dataDir := wrapper.Get("storage.data-dir")
	if dataDir == nil || dataDir.MustString() != "/data/tici-meta" {
		t.Fatalf("meta config should merge custom storage.data-dir, got: %s", cfg)
	}
}

func TestBuildTiCIConfigIncludesS3Prefix(t *testing.T) {
	tc := newTidbClusterForTiCIConfig()
	tc.Spec.TiCI.S3.Prefix = "custom_prefix"

	workerCfg, err := buildTiCIWorkerConfig(tc)
	if err != nil {
		t.Fatalf("build worker config failed: %v", err)
	}
	assertS3Prefix(t, workerCfg, "worker", "custom_prefix")

	metaCfg, err := buildTiCIMetaConfig(tc)
	if err != nil {
		t.Fatalf("build meta config failed: %v", err)
	}
	assertS3Prefix(t, metaCfg, "meta", "custom_prefix")
}

func assertS3Prefix(t *testing.T, cfg, component, expected string) {
	t.Helper()
	wrapper := tcconfig.New(map[string]interface{}{})
	if err := wrapper.UnmarshalTOML([]byte(cfg)); err != nil {
		t.Fatalf("%s config should be valid TOML, got err: %v, config: %s", component, err, cfg)
	}
	prefix := wrapper.Get("s3.prefix")
	if prefix == nil || prefix.MustString() != expected {
		t.Fatalf("%s config should include s3.prefix=%q, got: %s", component, expected, cfg)
	}
}

func TestAppendTiCICustomConfig(t *testing.T) {
	base := "[server]\npd-addr = \"x\"\n"

	got, err := appendTiCICustomConfig(base, "  ")
	if err != nil {
		t.Fatalf("append config should not fail: %v", err)
	}
	if got != base {
		t.Fatalf("expected base config unchanged when custom config is empty, got: %q", got)
	}

	got, err = appendTiCICustomConfig(base, "[server]\npd-addr = \"y\"\n[log]\nlevel = \"info\"")
	if err != nil {
		t.Fatalf("append config should not fail: %v", err)
	}
	if !strings.Contains(got, `[log]`) || !strings.Contains(got, `level = "info"`) {
		t.Fatalf("merged config should contain custom log section, got: %s", got)
	}

	wrapper := tcconfig.New(map[string]interface{}{})
	if err := wrapper.UnmarshalTOML([]byte(got)); err != nil {
		t.Fatalf("merged config should remain valid TOML: %v", err)
	}
	pdAddr := wrapper.Get("server.pd-addr")
	if pdAddr == nil || pdAddr.MustString() != "y" {
		t.Fatalf("custom server.pd-addr should override generated value, got: %v", pdAddr)
	}
}

func TestPrepareTiCIRollingUpgrade(t *testing.T) {
	partition := int32(1)
	oldSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "tc-tici-meta",
			Namespace:   "default",
			Annotations: map[string]string{},
		},
		Spec: appsv1.StatefulSetSpec{
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type:          appsv1.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{Partition: &partition},
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "tici-meta", Image: "tici:v1"}},
				},
			},
		},
		Status: appsv1.StatefulSetStatus{
			CurrentRevision: "tc-tici-meta-old",
			UpdateRevision:  "tc-tici-meta-new",
		},
	}
	specData, err := json.Marshal(appsv1.StatefulSetSpec{Template: oldSet.Spec.Template})
	if err != nil {
		t.Fatalf("marshal spec failed: %v", err)
	}
	oldSet.Annotations[LastAppliedConfigAnnotation] = string(specData)

	newSet := oldSet.DeepCopy()
	prepareTiCIRollingUpgrade(newSet, oldSet)
	if newSet.Spec.UpdateStrategy.RollingUpdate == nil || newSet.Spec.UpdateStrategy.RollingUpdate.Partition == nil {
		t.Fatalf("rolling update partition should be set")
	}
	if got := *newSet.Spec.UpdateStrategy.RollingUpdate.Partition; got != 0 {
		t.Fatalf("expected partition 0 when TiCI has pending upgrade, got: %d", got)
	}
}

func newTidbClusterForTiCIConfig() *v1alpha1.TidbCluster {
	return &v1alpha1.TidbCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tici-test",
			Namespace: "test-ns",
		},
		Spec: v1alpha1.TidbClusterSpec{
			TiDB: &v1alpha1.TiDBSpec{},
			TiCI: &v1alpha1.TiCISpec{
				Meta:   &v1alpha1.TiCIMetaSpec{},
				Worker: &v1alpha1.TiCIWorkerSpec{},
				S3: &v1alpha1.TiCIS3Spec{
					Endpoint: "http://minio-service:9000",
					Bucket:   "ticidefaultbucket",
				},
			},
		},
	}
}
