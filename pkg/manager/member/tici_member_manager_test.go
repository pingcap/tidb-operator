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

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	tcconfig "github.com/pingcap/tidb-operator/pkg/apis/util/config"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager/suspender"
	"github.com/pingcap/tidb-operator/pkg/manager/volumes"

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

func newFakeTiCIMemberManager() *ticiMemberManager {
	fakeDeps := controller.NewFakeDependencies()
	return &ticiMemberManager{
		deps:              fakeDeps,
		scaler:            NewFakeTiCIScaler(),
		suspender:         suspender.NewFakeSuspender(),
		podVolumeModifier: &volumes.FakePodVolumeModifier{},
	}
}

// newTidbClusterForTiCISync builds a cluster in the middle of a whole-cluster
// suspension: TiDB has already been fully suspended (its members are cleared
// by the suspender) while PD and TiKV are still running, so TiCI meta/worker
// are next in the suspend order.
func newTidbClusterForTiCISync() *v1alpha1.TidbCluster {
	tc := newTidbClusterForTiCIConfig()
	tc.Spec.PD = &v1alpha1.PDSpec{Replicas: 3}
	tc.Spec.TiKV = &v1alpha1.TiKVSpec{Replicas: 1}
	tc.Spec.TiDB = &v1alpha1.TiDBSpec{Replicas: 1}

	// PD and TiKV are still running and available.
	tc.Status.PD.Members = map[string]v1alpha1.PDMember{
		"pd-0": {Name: "pd-0", Health: true},
		"pd-1": {Name: "pd-1", Health: true},
		"pd-2": {Name: "pd-2", Health: true},
	}
	tc.Status.PD.StatefulSet = &appsv1.StatefulSetStatus{ReadyReplicas: 3}
	tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{
		"1": {ID: "1", PodName: "tici-test-tikv-0", State: v1alpha1.TiKVStateUp},
	}
	tc.Status.TiKV.StatefulSet = &appsv1.StatefulSetStatus{ReadyReplicas: 1}

	// TiDB has been suspended: members are cleared and it can never be ready.
	tc.Status.TiDB.Members = nil
	return tc
}

func TestTiCIMemberManagerSyncSuspend(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name          string
		modify        func(tc *v1alpha1.TidbCluster)
		suspend       func(component v1alpha1.MemberType) (bool, error)
		expectErr     bool
		expectSuspend bool
		// expectMemberTypes asserts which member types SuspendComponent is
		// called with; nil means no assertion.
		expectMemberTypes []v1alpha1.MemberType
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		tc := newTidbClusterForTiCISync()
		if test.modify != nil {
			test.modify(tc)
		}
		tmm := newFakeTiCIMemberManager()

		suspendCalled := false
		var suspendedComponents []v1alpha1.MemberType
		tmm.suspender.(*suspender.FakeSuspender).SuspendComponentFunc = func(c v1alpha1.Cluster, mt v1alpha1.MemberType) (bool, error) {
			suspendCalled = true
			suspendedComponents = append(suspendedComponents, mt)
			return test.suspend(mt)
		}

		err := tmm.Sync(tc)
		if test.expectErr {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}
		g.Expect(suspendCalled).To(Equal(test.expectSuspend))
		if test.expectMemberTypes != nil {
			g.Expect(suspendedComponents).To(ConsistOf(test.expectMemberTypes))
		}
	}

	tests := []testcase{
		{
			// Regression test: TiCI meta/worker must be suspendable even if
			// TiDB is not ready, e.g. TiDB has already been suspended during
			// a whole-cluster suspension. Otherwise the whole-cluster
			// suspension deadlocks on TiCI forever.
			name: "suspend when TiDB has been suspended",
			suspend: func(component v1alpha1.MemberType) (bool, error) {
				return true, nil
			},
			expectErr:     false,
			expectSuspend: true,
			expectMemberTypes: []v1alpha1.MemberType{
				v1alpha1.TiCIMetaMemberType, v1alpha1.TiCIWorkerMemberType,
			},
		},
		{
			// Availability checks still apply when not suspending.
			name: "requeue when TiDB is not ready and not suspending",
			suspend: func(component v1alpha1.MemberType) (bool, error) {
				return false, nil
			},
			expectErr:     true,
			expectSuspend: true,
			expectMemberTypes: []v1alpha1.MemberType{
				v1alpha1.TiCIMetaMemberType, v1alpha1.TiCIWorkerMemberType,
			},
		},
		{
			// Only meta is suspended while worker is still being synced
			// normally: the availability checks still apply for worker.
			name: "requeue when only meta is suspended and TiDB is not ready",
			suspend: func(component v1alpha1.MemberType) (bool, error) {
				return component == v1alpha1.TiCIMetaMemberType, nil
			},
			expectErr:     true,
			expectSuspend: true,
			expectMemberTypes: []v1alpha1.MemberType{
				v1alpha1.TiCIMetaMemberType, v1alpha1.TiCIWorkerMemberType,
			},
		},
		{
			// A cluster configuring only meta (worker is nil) must also be
			// able to finish suspending. Note this state is not reachable
			// through the API in practice — defaulting fills an empty struct
			// for a nil worker — but the skip logic should treat nil as
			// skipped rather than rely on defaulting.
			name:   "suspend when only meta is configured",
			modify: func(tc *v1alpha1.TidbCluster) { tc.Spec.TiCI.Worker = nil },
			suspend: func(component v1alpha1.MemberType) (bool, error) {
				return true, nil
			},
			expectErr:     false,
			expectSuspend: true,
			expectMemberTypes: []v1alpha1.MemberType{
				v1alpha1.TiCIMetaMemberType,
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}
