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
	"github.com/pingcap/tidb-operator/pkg/controller"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
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

func TestRenderTiCIMetaStartScriptInjectsTiDBAuthFromSecretVolume(t *testing.T) {
	tc := newTidbClusterForTiCIConfig()
	tidbAuth := &tiCIMetaTiDBAuth{User: "root"}

	script := renderTiCIMetaStartScript(tc, "tici-test-tici-meta-peer", tidbAuth)
	for _, expected := range []string{
		`tidb_auth_user="root"`,
		`tidb_auth_file="/etc/tici-tidb-auth/tidb-auth"`,
		`od -An -tx1 -v "${tidb_auth_file}"`,
		`sed "s#mysql://${tidb_auth_user}@#mysql://${tidb_auth_user}:${encoded_tidb_auth}@#"`,
		`exec /tici-server meta --config=/etc/tici/tici.toml`,
	} {
		if !strings.Contains(script, expected) {
			t.Fatalf("expected meta start script to contain %q, got: %s", expected, script)
		}
	}
}

func TestRenderTiCIMetaStartScriptWithoutTiDBAuthCopiesConfig(t *testing.T) {
	tc := newTidbClusterForTiCIConfig()

	script := renderTiCIMetaStartScript(tc, "tici-test-tici-meta-peer", nil)
	for _, expected := range []string{
		`cp "${config_template}" "${config_file}"`,
		`exec /tici-server meta --config=/etc/tici/tici.toml`,
	} {
		if !strings.Contains(script, expected) {
			t.Fatalf("expected meta start script to contain %q, got: %s", expected, script)
		}
	}
	if strings.Contains(script, "tidb_auth_file") {
		t.Fatalf("expected meta start script without TiDB auth to avoid auth file handling, got: %s", script)
	}
}

func TestBuildTiCIMetaConfigUsesTiDBAuthUser(t *testing.T) {
	tc := newTidbClusterForTiCIConfig()
	tc.Spec.TiCI.Meta.TiDBAuth = &v1alpha1.TiCITiDBAuth{
		User: "tici",
		PasswordSecret: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: "tidb-auth"},
			Key:                  "auth",
		},
	}

	cfg, err := buildTiCIMetaConfig(tc)
	if err != nil {
		t.Fatalf("build meta config failed: %v", err)
	}
	if !strings.Contains(cfg, `mysql://tici@tici-test-tidb:4000`) {
		t.Fatalf("expected meta config to use configured TiDB auth user without credential data, got: %s", cfg)
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

func TestPrepareTiCIRollingUpgradeWithTemplateAnnotationChange(t *testing.T) {
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
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						ticiMetaTiDBAuthHashAnnotation: "old",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "tici-meta", Image: "tici:v1"}},
				},
			},
		},
		Status: appsv1.StatefulSetStatus{
			CurrentRevision: "tc-tici-meta-current",
			UpdateRevision:  "tc-tici-meta-current",
		},
	}
	specData, err := json.Marshal(appsv1.StatefulSetSpec{Template: oldSet.Spec.Template})
	if err != nil {
		t.Fatalf("marshal spec failed: %v", err)
	}
	oldSet.Annotations[LastAppliedConfigAnnotation] = string(specData)

	newSet := oldSet.DeepCopy()
	newSet.Spec.Template.Annotations[ticiMetaTiDBAuthHashAnnotation] = "new"

	prepareTiCIRollingUpgrade(newSet, oldSet)
	if newSet.Spec.UpdateStrategy.RollingUpdate == nil || newSet.Spec.UpdateStrategy.RollingUpdate.Partition == nil {
		t.Fatalf("rolling update partition should be set")
	}
	if got := *newSet.Spec.UpdateStrategy.RollingUpdate.Partition; got != 0 {
		t.Fatalf("expected partition 0 when TiCI template annotation changes, got: %d", got)
	}
}

func TestGetNewTiCIMetaStatefulSetUsesConfigMapAndTiDBAuthSecret(t *testing.T) {
	tc := newTidbClusterForTiCIConfig()
	tidbAuth := &tiCIMetaTiDBAuth{
		User:       "root",
		SecretName: "tidb-auth",
		SecretKey:  "auth",
		Hash:       "tidb-auth-hash",
	}
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tici-test-tici-meta",
			Namespace: "test-ns",
		},
		Data: map[string]string{"config-file": "[server]\npd-addr = \"x\"\n"},
	}

	sts, err := getNewTiCIMetaStatefulSet(tc, cm, tidbAuth)
	if err != nil {
		t.Fatalf("build TiCI meta statefulset failed: %v", err)
	}
	if sts == nil {
		t.Fatal("expected non-nil TiCI meta statefulset")
		return
	}

	foundConfigVolume := false
	foundRuntimeConfigVolume := false
	foundTiDBAuthVolume := false
	for _, volume := range sts.Spec.Template.Spec.Volumes {
		if volume.Name == "config" && volume.ConfigMap != nil && volume.ConfigMap.Name == cm.Name {
			foundConfigVolume = true
		}
		if volume.Name == "runtime-config" && volume.EmptyDir != nil {
			foundRuntimeConfigVolume = true
		}
		if volume.Name == "tidb-auth" && volume.Secret != nil && volume.Secret.SecretName == "tidb-auth" &&
			len(volume.Secret.Items) == 1 && volume.Secret.Items[0].Key == "auth" && volume.Secret.Items[0].Path == "tidb-auth" &&
			volume.Secret.Optional == nil {
			foundTiDBAuthVolume = true
		}
	}
	if !foundConfigVolume {
		t.Fatalf("expected TiCI meta statefulset to mount config from configmap, got volumes: %+v", sts.Spec.Template.Spec.Volumes)
	}
	if !foundRuntimeConfigVolume {
		t.Fatalf("expected TiCI meta statefulset to include writable runtime config volume, got volumes: %+v", sts.Spec.Template.Spec.Volumes)
	}
	if !foundTiDBAuthVolume {
		t.Fatalf("expected TiCI meta statefulset to mount TiDB auth secret, got volumes: %+v", sts.Spec.Template.Spec.Volumes)
	}
	if sts.Spec.Template.Annotations[ticiMetaTiDBAuthHashAnnotation] != "tidb-auth-hash" {
		t.Fatalf("expected TiCI meta statefulset to include TiDB auth hash annotation, got: %s", sts.Spec.Template.Annotations[ticiMetaTiDBAuthHashAnnotation])
	}
}

func TestGetNewTiCIMetaStatefulSetWithoutTiDBAuthDoesNotMountAuthSecret(t *testing.T) {
	tc := newTidbClusterForTiCIConfig()
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tici-test-tici-meta",
			Namespace: "test-ns",
		},
		Data: map[string]string{"config-file": "[server]\npd-addr = \"x\"\n"},
	}

	sts, err := getNewTiCIMetaStatefulSet(tc, cm, nil)
	if err != nil {
		t.Fatalf("build TiCI meta statefulset failed: %v", err)
	}
	if sts == nil {
		t.Fatal("expected non-nil TiCI meta statefulset")
		return
	}
	for _, volume := range sts.Spec.Template.Spec.Volumes {
		if volume.Name == "tidb-auth" {
			t.Fatalf("expected TiCI meta statefulset without TiDB auth to avoid auth secret volume, got volumes: %+v", sts.Spec.Template.Spec.Volumes)
		}
	}
	if _, ok := sts.Spec.Template.Annotations[ticiMetaTiDBAuthHashAnnotation]; ok {
		t.Fatalf("expected TiCI meta statefulset without TiDB auth to avoid TiDB auth hash annotation, got: %+v", sts.Spec.Template.Annotations)
	}
}

func TestGetTiCIMetaTiDBAuthFromExplicitSecret(t *testing.T) {
	tc := newTidbClusterForTiCIConfig()
	tc.Spec.TiCI.Meta.TiDBAuth = &v1alpha1.TiCITiDBAuth{
		PasswordSecret: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: "tidb-auth"},
			Key:                  "auth",
		},
	}
	deps := controller.NewFakeDependencies()
	secretIndexer := deps.KubeInformerFactory.Core().V1().Secrets().Informer().GetIndexer()
	if err := secretIndexer.Add(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "tidb-auth", Namespace: "test-ns"},
		Data:       map[string][]byte{"auth": []byte("secret-data")},
	}); err != nil {
		t.Fatalf("add secret failed: %v", err)
	}
	m := &ticiMemberManager{deps: deps}

	auth, err := m.getTiCIMetaTiDBAuth(tc)
	if err != nil {
		t.Fatalf("get TiCI meta TiDB auth failed: %v", err)
	}
	if auth == nil {
		t.Fatal("expected TiCI meta TiDB auth")
		return
	}
	if auth.User != "root" || auth.SecretName != "tidb-auth" || auth.SecretKey != "auth" || auth.Hash == "" {
		t.Fatalf("unexpected TiCI meta TiDB auth: %+v", auth)
	}
}

func TestGetTiCIMetaTiDBAuthMissingSecretWarnsAndRequeues(t *testing.T) {
	tc := newTidbClusterForTiCIConfig()
	tc.Spec.TiCI.Meta.TiDBAuth = &v1alpha1.TiCITiDBAuth{
		PasswordSecret: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: "tidb-auth"},
			Key:                  "auth",
		},
	}
	deps := controller.NewFakeDependencies()
	m := &ticiMemberManager{deps: deps}

	auth, err := m.getTiCIMetaTiDBAuth(tc)
	if auth != nil {
		t.Fatalf("expected no TiCI meta TiDB auth when secret is missing, got: %+v", auth)
	}
	if !controller.IsRequeueError(err) {
		t.Fatalf("expected missing TiCI meta TiDB auth secret to requeue, got: %v", err)
	}
	event := readFakeRecorderEvent(t, deps)
	if !strings.Contains(event, corev1.EventTypeWarning) || !strings.Contains(event, ticiMetaTiDBAuthRefNotFound) || !strings.Contains(event, "test-ns/tidb-auth") {
		t.Fatalf("unexpected event for missing TiCI meta TiDB auth secret: %s", event)
	}
}

func TestGetTiCIMetaTiDBAuthMissingSecretKeyWarnsAndErrors(t *testing.T) {
	tc := newTidbClusterForTiCIConfig()
	tc.Spec.TiCI.Meta.TiDBAuth = &v1alpha1.TiCITiDBAuth{
		PasswordSecret: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: "tidb-auth"},
			Key:                  "auth",
		},
	}
	deps := controller.NewFakeDependencies()
	secretIndexer := deps.KubeInformerFactory.Core().V1().Secrets().Informer().GetIndexer()
	if err := secretIndexer.Add(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "tidb-auth", Namespace: "test-ns"},
		Data:       map[string][]byte{"other": []byte("secret-data")},
	}); err != nil {
		t.Fatalf("add secret failed: %v", err)
	}
	m := &ticiMemberManager{deps: deps}

	auth, err := m.getTiCIMetaTiDBAuth(tc)
	if auth != nil {
		t.Fatalf("expected no TiCI meta TiDB auth when secret key is missing, got: %+v", auth)
	}
	if err == nil || controller.IsRequeueError(err) || !strings.Contains(err.Error(), `key "auth"`) {
		t.Fatalf("expected missing TiCI meta TiDB auth secret key to return a clear non-requeue error, got: %v", err)
	}
	event := readFakeRecorderEvent(t, deps)
	if !strings.Contains(event, corev1.EventTypeWarning) || !strings.Contains(event, ticiMetaTiDBAuthRefKeyNotFound) || !strings.Contains(event, `key "auth"`) {
		t.Fatalf("unexpected event for missing TiCI meta TiDB auth secret key: %s", event)
	}
}

func readFakeRecorderEvent(t *testing.T, deps *controller.Dependencies) string {
	t.Helper()
	fakeRecorder, ok := deps.Recorder.(*record.FakeRecorder)
	if !ok {
		t.Fatalf("expected fake recorder, got %T", deps.Recorder)
	}
	select {
	case event := <-fakeRecorder.Events:
		return event
	default:
		t.Fatal("expected event to be recorded")
		return ""
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
