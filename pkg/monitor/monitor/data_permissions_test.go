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

package monitor

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func TestMonitorDataPermissions(t *testing.T) {
	for _, tc := range []struct {
		name    string
		context *core.PodSecurityContext
		want    int64
	}{
		{"default", nil, 2000},
		{"existing groups", &core.PodSecurityContext{SupplementalGroups: []int64{5678}}, 2000},
		{"already present", &core.PodSecurityContext{SupplementalGroups: []int64{2000}}, 2000},
		{"fsGroup zero", &core.PodSecurityContext{FSGroup: pointer.Int64Ptr(0)}, 0},
		{"fsGroup", &core.PodSecurityContext{FSGroup: pointer.Int64Ptr(1234)}, 1234},
	} {
		t.Run(tc.name, func(t *testing.T) {
			monitor := &v1alpha1.TidbMonitor{ObjectMeta: metav1.ObjectMeta{Name: "probe", Namespace: "test"}, Spec: v1alpha1.TidbMonitorSpec{
				PodSecurityContext: tc.context,
				Grafana:            &v1alpha1.GrafanaSpec{Envs: map[string]string{"GF_PATHS_DATA": "/data/custom-grafana", "GF_PATHS_PLUGINS": "/data/plugins"}},
				Thanos:             &v1alpha1.ThanosSpec{ObjectStorageConfig: &core.SecretKeySelector{}},
				DM:                 &v1alpha1.DMMonitorSpec{Clusters: []v1alpha1.ClusterRef{{Name: "dm"}}},
			}}
			before := monitor.DeepCopy()
			sts, err := getMonitorStatefulSet(getMonitorServiceAccount(monitor), getMonitorSecret(monitor), monitor, &v1alpha1.TidbCluster{}, &v1alpha1.DMCluster{}, 0)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(monitor, before) {
				t.Fatal("mutated the input CR")
			}
			context := sts.Spec.Template.Spec.SecurityContext
			if tc.context != nil && !reflect.DeepEqual(tc.context.FSGroup, context.FSGroup) {
				t.Fatal("changed fsGroup")
			}
			if context.FSGroup == nil {
				count := 0
				for _, g := range context.SupplementalGroups {
					if g == 2000 {
						count++
					}
				}
				if count != 1 {
					t.Fatal("default group missing or duplicated", context)
				}
			}
			if tc.context != nil {
				for _, g := range tc.context.SupplementalGroups {
					found := false
					for _, actual := range context.SupplementalGroups {
						found = found || actual == g
					}
					if !found {
						t.Fatal("dropped an existing group")
					}
				}
			}
			if len(sts.Spec.Template.Spec.InitContainers) != 2 {
				t.Fatal("missing DM initializer")
			}
			for _, c := range sts.Spec.Template.Spec.InitContainers {
				if len(c.Command) < 7 || c.Command[5] != strconv.FormatInt(tc.want, 10) {
					t.Fatal("wrong initializer group", c.Command)
				}
				if !strings.Contains(c.Command[2], "exec /usr/bin/init.sh") || strings.Contains(c.Command[2], "chmod 777") {
					t.Fatal("wrong initializer command")
				}
				if c.SecurityContext != nil {
					t.Fatal("overrode initializer identity")
				}
			}
			expected := map[string][]string{"prometheus": {"/data/prometheus"}, "reloader": {"/data/latest-rules"}, "grafana": {"/data/custom-grafana", "/data/plugins"}, "thanos-sidecar": {"/data/prometheus"}}
			for _, c := range sts.Spec.Template.Spec.Containers {
				var got []string
				for _, mount := range c.VolumeMounts {
					if mount.Name == v1alpha1.TidbMonitorMemberType.String() {
						if mount.SubPath != strings.TrimPrefix(mount.MountPath, "/data/") {
							t.Fatal("incorrect data subPath", c.Name, mount)
						}
						got = append(got, mount.MountPath)
					}
				}
				if want, ok := expected[c.Name]; ok && !reflect.DeepEqual(want, got) {
					t.Fatal("unexpected data mounts", c.Name, got)
				}
			}
		})
	}
}

func TestGrafanaDataPathOverrides(t *testing.T) {
	for _, value := range []string{"/data", "/data/../etc", "/data/x/../grafana", "/tmp/../data/grafana", "/data/$(POD_NAME)", "$(CUSTOM_PATH)", "relative-data", "/data/$$literal", "/data/${GRAFANA_DIR}", "/data/$__env{GRAFANA_DIR}", "/data/$__file{/etc/path}", "/data/$__vault{path}"} {
		monitor := &v1alpha1.TidbMonitor{Spec: v1alpha1.TidbMonitorSpec{Grafana: &v1alpha1.GrafanaSpec{Envs: map[string]string{"GF_PATHS_DATA": value}}}}
		if err := validateMonitorDataPaths(monitor); err == nil {
			t.Fatal("unsafe or ambiguous data path accepted", value)
		}
		for _, key := range []string{"GF_PATHS_DATA", "GF_PATHS_PLUGINS", "GF_PATHS_LOGS"} {
			monitor.Spec.Grafana.Envs = map[string]string{key: value}
			monitor.Spec.Grafana.AdditionalVolumeMounts = []core.VolumeMount{{Name: "external", MountPath: "/data"}}
			if strings.Contains(value, "$") && validateMonitorDataPaths(monitor) == nil {
				t.Fatal("explicit mount hid an unresolved expansion", key, value)
			}
		}
	}
	for _, value := range []string{"//data/grafana", "/data/./grafana/"} {
		monitor := &v1alpha1.TidbMonitor{Spec: v1alpha1.TidbMonitorSpec{Grafana: &v1alpha1.GrafanaSpec{Envs: map[string]string{"GF_PATHS_DATA": value}}}}
		if err := validateMonitorDataPaths(monitor); err != nil {
			t.Fatal(err)
		}
		if got := grafanaDataPaths(monitor); !reflect.DeepEqual(got, []string{"/data/grafana"}) {
			t.Fatal("equivalent path lost its PVC mount", value, got)
		}
	}
	monitor := &v1alpha1.TidbMonitor{Spec: v1alpha1.TidbMonitorSpec{Grafana: &v1alpha1.GrafanaSpec{AdditionalVolumeMounts: []core.VolumeMount{{Name: "external", MountPath: "/data/grafana"}}}}}
	c := getMonitorGrafanaContainer(&core.Secret{}, monitor)
	count := 0
	for _, mount := range c.VolumeMounts {
		if mount.MountPath == "/data/grafana" {
			count++
			if mount.Name != "external" {
				t.Fatal("replaced explicit mount")
			}
		}
	}
	if count != 1 {
		t.Fatal("duplicate explicit mount")
	}
}

func TestGrafanaExplicitMountPreparation(t *testing.T) {
	for _, tc := range []struct {
		name, key, value, mountPath string
		wantPaths                   []string
	}{
		{"logs on external root", "GF_PATHS_LOGS", "/data", "/data", nil},
		{"external plugins", "GF_PATHS_PLUGINS", "/data/plugins", "/data/plugins", []string{"/data/grafana"}},
		{"external plugin parent", "GF_PATHS_PLUGINS", "/data/external/plugins", "/data/external", []string{"/data/grafana"}},
		{"external data", "GF_PATHS_DATA", "/data/custom", "/data/custom", nil},
		{"relative external root", "GF_PATHS_LOGS", "/data", "data", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			explicit := core.VolumeMount{Name: "external", MountPath: tc.mountPath}
			monitor := &v1alpha1.TidbMonitor{ObjectMeta: metav1.ObjectMeta{Name: "probe", Namespace: "test"}, Spec: v1alpha1.TidbMonitorSpec{
				Grafana: &v1alpha1.GrafanaSpec{Envs: map[string]string{tc.key: tc.value}, AdditionalVolumeMounts: []core.VolumeMount{explicit}},
			}}
			sts, err := getMonitorStatefulSet(getMonitorServiceAccount(monitor), getMonitorSecret(monitor), monitor, &v1alpha1.TidbCluster{}, nil, 0)
			if err != nil {
				t.Fatal("explicit storage rejected", err)
			}
			if got := grafanaDataPaths(monitor); !reflect.DeepEqual(got, tc.wantPaths) {
				t.Fatal("prepared unused PVC paths", got, tc.wantPaths)
			}
			wantDirs := append([]string{"/data/prometheus", "/data/latest-rules"}, tc.wantPaths...)
			if got := sts.Spec.Template.Spec.InitContainers[0].Command[6:]; !reflect.DeepEqual(got, wantDirs) {
				t.Fatal("incorrect permission preparation", got, wantDirs)
			}
			c := getMonitorGrafanaContainer(&core.Secret{}, monitor)
			found := false
			for _, mount := range c.VolumeMounts {
				if mount.Name == "external" {
					found = reflect.DeepEqual(mount, explicit)
				}
			}
			if !found {
				t.Fatal("explicit mount lost", c.VolumeMounts)
			}
		})
	}
}

func TestMonitorDataMountOverrides(t *testing.T) {
	for _, component := range []string{"prometheus", "thanos", "grafana"} {
		dataPath := "/data/prometheus"
		if component == "grafana" {
			dataPath = "/data/grafana"
		}
		for _, tc := range []struct {
			name, mountPath string
			wantDefault     bool
		}{
			{"default", "", true},
			{"same path", dataPath, false},
			{"parent", "/data", false},
			{"parent trailing slash", "/data/", false},
			{"relative parent", "data", false},
			{"relative same path", strings.TrimPrefix(dataPath, "/"), false},
			{"equivalent path", "/" + dataPath + "/", false},
			{"sibling", dataPath + "-other", true},
			{"child", dataPath + "/custom", true},
		} {
			t.Run(component+"/"+tc.name, func(t *testing.T) {
				var additional []core.VolumeMount
				if tc.mountPath != "" {
					additional = []core.VolumeMount{{Name: "external-data", MountPath: tc.mountPath, SubPath: "existing-data", ReadOnly: true}}
				}
				monitor := &v1alpha1.TidbMonitor{Spec: v1alpha1.TidbMonitorSpec{
					Grafana: &v1alpha1.GrafanaSpec{},
					Thanos:  &v1alpha1.ThanosSpec{ObjectStorageConfig: &core.SecretKeySelector{}},
				}}
				var c core.Container
				switch component {
				case "prometheus":
					monitor.Spec.Prometheus.AdditionalVolumeMounts = additional
					c = getMonitorPrometheusContainer(monitor, 0)
				case "thanos":
					monitor.Spec.Thanos.AdditionalVolumeMounts = additional
					c = getThanosSidecarContainer(monitor)
				case "grafana":
					monitor.Spec.Grafana.AdditionalVolumeMounts = additional
					c = getMonitorGrafanaContainer(&core.Secret{}, monitor)
				}
				foundDefault, foundExplicit := false, false
				paths := map[string]bool{}
				for _, mount := range c.VolumeMounts {
					if paths[mount.MountPath] {
						t.Fatal("duplicate mountPath", mount.MountPath)
					}
					paths[mount.MountPath] = true
					if mount.Name == v1alpha1.TidbMonitorMemberType.String() && mount.MountPath == dataPath {
						foundDefault = true
					}
					if mount.Name == "external-data" {
						foundExplicit = true
						if !reflect.DeepEqual(mount, additional[0]) {
							t.Fatal("changed explicit mount", mount)
						}
					}
				}
				if foundDefault != tc.wantDefault || foundExplicit != (len(additional) > 0) {
					t.Fatal("incorrect mount precedence", c.VolumeMounts)
				}
			})
		}
	}
}

func TestPrepareMonitorData(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("initializer uses Linux stat")
	}
	run := func(root string, dirs ...string) ([]byte, error) {
		script := strings.TrimSuffix(prepareMonitorData, "exec /usr/bin/init.sh")
		args := append([]string{"-ec", script, "prepare-monitor-data", root, strconv.Itoa(os.Getgid())}, dirs...)
		return exec.Command("/bin/sh", args...).CombinedOutput()
	}
	t.Run("unused externally mounted plugins may be a symlink on the PVC", func(t *testing.T) {
		root, outside := t.TempDir(), t.TempDir()
		plugins := filepath.Join(root, "plugins")
		if err := os.Symlink(outside, plugins); err != nil {
			t.Fatal(err)
		}
		monitor := &v1alpha1.TidbMonitor{Spec: v1alpha1.TidbMonitorSpec{Grafana: &v1alpha1.GrafanaSpec{
			Envs: map[string]string{"GF_PATHS_PLUGINS": "/data/plugins"}, AdditionalVolumeMounts: []core.VolumeMount{{Name: "external", MountPath: "/data/plugins"}},
		}}}
		var dirs []string
		for _, dir := range getInitCommand(monitor)[6:] {
			dirs = append(dirs, root+strings.TrimPrefix(dir, "/data"))
		}
		before, _ := os.Stat(outside)
		if out, err := run(root, dirs...); err != nil {
			t.Fatalf("unused symlink blocked initialization: %v: %s", err, out)
		}
		after, _ := os.Stat(outside)
		if before.Mode() != after.Mode() {
			t.Fatal("modified unused symlink target")
		}
		if target, err := os.Readlink(plugins); err != nil || target != outside {
			t.Fatal("modified unused symlink", target, err)
		}
	})
	t.Run("prepare and repeat without touching files", func(t *testing.T) {
		root := t.TempDir()
		dir := filepath.Join(root, "a' b;$literal")
		if err := os.Mkdir(dir, 0777); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(dir, 0777); err != nil {
			t.Fatal(err)
		}
		file := filepath.Join(dir, "old-data")
		if err := os.WriteFile(file, []byte("unchanged"), 0600); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 2; i++ {
			if out, err := run(root, dir); err != nil {
				t.Fatalf("%v: %s", err, out)
			}
		}
		info, err := os.Stat(dir)
		if err != nil || info.Mode().Perm()&0772 != 0770 || info.Mode()&os.ModeSetgid == 0 {
			t.Fatal("directory permissions not prepared", info, err)
		}
		info, err = os.Stat(file)
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatal("changed existing file permissions", info, err)
		}
		data, err := os.ReadFile(file)
		if err != nil || string(data) != "unchanged" {
			t.Fatal("changed existing data")
		}
	})
	for _, attack := range []string{"symlink", "ancestor", "file", "outside", "root"} {
		t.Run(attack, func(t *testing.T) {
			root := t.TempDir()
			outside := t.TempDir()
			dir := filepath.Join(root, "target")
			switch attack {
			case "symlink", "ancestor":
				if err := os.Symlink(outside, dir); err != nil {
					t.Fatal(err)
				}
				if attack == "ancestor" {
					dir = filepath.Join(dir, "child")
				}
			case "file":
				if err := os.WriteFile(dir, []byte("keep"), 0600); err != nil {
					t.Fatal(err)
				}
			case "outside":
				dir = outside
			case "root":
				dir = root
			}
			before, _ := os.Stat(outside)
			if out, err := run(root, dir); err == nil {
				t.Fatalf("accepted %s: %s", attack, out)
			}
			after, _ := os.Stat(outside)
			if before.Mode() != after.Mode() {
				t.Fatal("modified outside directory")
			}
		})
	}
}

func TestInitializerReloaderUID(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		podUID, containerUID *int64
		explicit             string
		want                 string
	}{
		{name: "default"},
		{name: "container UID", containerUID: pointer.Int64Ptr(1234), want: "1234"},
		{name: "inherited UID", podUID: pointer.Int64Ptr(1234), want: "1234"},
		{name: "root override", podUID: pointer.Int64Ptr(1234), containerUID: pointer.Int64Ptr(0)},
		{name: "explicit env", containerUID: pointer.Int64Ptr(1234), explicit: "5678", want: "5678"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			monitor := &v1alpha1.TidbMonitor{Spec: v1alpha1.TidbMonitorSpec{
				PodSecurityContext: &core.PodSecurityContext{RunAsUser: tc.podUID},
				Reloader:           v1alpha1.ReloaderSpec{MonitorContainer: v1alpha1.MonitorContainer{SecurityContext: &core.SecurityContext{RunAsUser: tc.containerUID}}},
				DM:                 &v1alpha1.DMMonitorSpec{},
			}}
			if tc.explicit != "" {
				monitor.Spec.Initializer.Envs = map[string]string{"RELOADER_UID": tc.explicit}
				monitor.Spec.DM.Initializer.Envs = map[string]string{"RELOADER_UID": tc.explicit}
			}
			for _, c := range []core.Container{getMonitorInitContainer(monitor, nil), getMonitorDMInitContainer(monitor, &v1alpha1.DMCluster{})} {
				got := ""
				for _, env := range c.Env {
					if env.Name == "RELOADER_UID" {
						got = env.Value
					}
				}
				if got != tc.want {
					t.Fatal(c.Name, "incorrect serving UID", got, tc.want)
				}
			}
		})
	}
}
