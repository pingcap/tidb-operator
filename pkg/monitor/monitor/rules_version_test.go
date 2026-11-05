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
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	clientfake "github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	"github.com/pingcap/tidb-operator/pkg/controller"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kubefake "k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
	"k8s.io/utils/pointer"
)

func rulesVersionFixture(t *testing.T) (*MonitorManager, *v1alpha1.TidbMonitor) {
	t.Helper()
	m := newFakeTidbMonitorManager()
	tm := newTidbMonitor(v1alpha1.TidbClusterRef{Name: "tc", Namespace: "ns"})
	tm.UID = types.UID("monitor-uid")
	tm.Spec.Initializer.Version = "new-image-tag"
	if _, err := m.deps.Clientset.PingcapV1alpha1().TidbMonitors(tm.Namespace).Create(context.Background(), tm, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	return m, tm
}

func TestMonitorRulesVersionRecordsBeforeRollout(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprintf("recordFails=%v", fail), func(t *testing.T) {
			m, tm := rulesVersionFixture(t)
			tc := &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{Name: "tc", Namespace: tm.Namespace},
				Spec:       v1alpha1.TidbClusterSpec{TiDB: &v1alpha1.TiDBSpec{}, TiKV: &v1alpha1.TiKVSpec{}},
			}
			if err := m.deps.TiDBClusterControl.Create(tc); err != nil {
				t.Fatal(err)
			}
			oldRulesWorkload(t, m, tm, "old-monitor", "tidb:old", false)
			if fail {
				m.deps.Clientset.(*clientfake.Clientset).PrependReactor("patch", "tidbmonitors", func(ktesting.Action) (bool, runtime.Object, error) {
					return true, nil, fmt.Errorf("conflict")
				})
			}
			err := m.SyncMonitor(tm)
			if fail && (err == nil || !strings.Contains(err.Error(), "record monitor rules version")) {
				t.Fatalf("recording failure did not stop reconciliation: %v", err)
			}
			if !fail && err != nil && !controller.IsRequeueError(err) {
				t.Fatal(err)
			}
			sets, err := m.deps.StatefulSetLister.StatefulSets(tm.Namespace).List(labels.Everything())
			if err != nil {
				t.Fatal(err)
			}
			if fail {
				if len(sets) != 0 {
					t.Fatal("rollout happened despite recording failure")
				}
				return
			}
			if len(sets) != 1 {
				t.Fatalf("expected rollout after recording, got %d StatefulSets", len(sets))
			}
			version, err := reloaderRulesVersion(sets[0].Spec.Template.Spec.Containers)
			if err != nil || version != "tidb:old" {
				t.Fatal("rollout changed rule directory", version, err)
			}
			if !strings.HasSuffix(sets[0].Spec.Template.Spec.InitContainers[0].Image, ":new-image-tag") {
				t.Fatal("rollout did not update initializer image")
			}
		})
	}
}

func TestMonitorRulesVersionUsesCurrentResourceVersion(t *testing.T) {
	m, tm := rulesVersionFixture(t)
	client := m.deps.Clientset.(*clientfake.Clientset)
	client.PrependReactor("update", "tidbmonitors", func(action ktesting.Action) (bool, runtime.Object, error) {
		updated := action.(ktesting.UpdateAction).GetObject().(*v1alpha1.TidbMonitor).DeepCopy()
		updated.ResourceVersion = "after-finalizer"
		return true, updated, nil
	})
	if err := m.addProtectionFinalizerIfNeed(tm); err != nil {
		t.Fatal(err)
	}
	client.PrependReactor("patch", "tidbmonitors", func(action ktesting.Action) (bool, runtime.Object, error) {
		var patch struct {
			Metadata metav1.ObjectMeta `json:"metadata"`
		}
		if err := json.Unmarshal(action.(ktesting.PatchAction).GetPatch(), &patch); err != nil {
			t.Fatal(err)
		}
		if patch.Metadata.ResourceVersion != "after-finalizer" {
			t.Fatal("annotation patch does not protect the current resource version")
		}
		return false, nil, nil
	})
	if err := m.syncMonitorRulesVersion(tm); err != nil {
		t.Fatal(err)
	}
}

func oldRulesWorkload(t *testing.T, m *MonitorManager, tm *v1alpha1.TidbMonitor, name, version string, deployment bool) {
	t.Helper()
	meta := metav1.ObjectMeta{Name: name, Namespace: tm.Namespace, OwnerReferences: []metav1.OwnerReference{
		*metav1.NewControllerRef(tm, v1alpha1.SchemeGroupVersion.WithKind("TidbMonitor")),
	}}
	template := core.PodTemplateSpec{Spec: core.PodSpec{Containers: []core.Container{{Name: "reloader", Command: []string{"/bin/reload", "--sub-store-path=" + version}}}}}
	var err error
	if deployment {
		_, err = m.deps.KubeClientset.AppsV1().Deployments(tm.Namespace).Create(context.Background(), &apps.Deployment{ObjectMeta: meta, Spec: apps.DeploymentSpec{Template: template}}, metav1.CreateOptions{})
	} else {
		_, err = m.deps.KubeClientset.AppsV1().StatefulSets(tm.Namespace).Create(context.Background(), &apps.StatefulSet{ObjectMeta: meta, Spec: apps.StatefulSetSpec{Template: template}}, metav1.CreateOptions{})
	}
	if err != nil {
		t.Fatal(err)
	}
}

func TestMonitorRulesVersionSurvivesImageUpgrade(t *testing.T) {
	for _, deployment := range []bool{false, true} {
		t.Run(fmt.Sprintf("deployment=%v", deployment), func(t *testing.T) {
			m, tm := rulesVersionFixture(t)
			// Only the live API has the workload: the informer may not have observed it yet.
			oldRulesWorkload(t, m, tm, "old-monitor", "tidb:v8.5.7", deployment)
			if err := m.syncMonitorRulesVersion(tm); err != nil {
				t.Fatal(err)
			}
			stored, err := m.deps.Clientset.PingcapV1alpha1().TidbMonitors(tm.Namespace).Get(context.Background(), tm.Name, metav1.GetOptions{})
			if err != nil || stored.Annotations[label.AnnMonitorRulesVersion] != "tidb:v8.5.7" {
				t.Fatal("effective version was not durably recorded", stored, err)
			}
			if stored.Spec.AlertManagerRulesVersion != nil || stored.Spec.Initializer.Version != "new-image-tag" {
				t.Fatal("recording version changed the user's spec")
			}
			// Even complete workload deletion and another image upgrade retain the choice.
			if deployment {
				err = m.deps.KubeClientset.AppsV1().Deployments(tm.Namespace).Delete(context.Background(), "old-monitor", metav1.DeleteOptions{})
			} else {
				err = m.deps.KubeClientset.AppsV1().StatefulSets(tm.Namespace).Delete(context.Background(), "old-monitor", metav1.DeleteOptions{})
			}
			if err != nil {
				t.Fatal(err)
			}
			stored.Spec.Initializer.Version = "another-image-tag"
			if err := m.syncMonitorRulesVersion(stored); err != nil {
				t.Fatal(err)
			}
			if got := getAlertManagerRulesVersion(stored); got != "tidb:v8.5.7" {
				t.Fatalf("workload recreation switched rules directory to %s", got)
			}
			reloader := getMonitorReloaderContainer(stored)
			version, err := reloaderRulesVersion([]core.Container{reloader})
			if err != nil || version != "tidb:v8.5.7" {
				t.Fatal("generated reloader changed persistence directory", version, err)
			}
			stored.Spec.DM = &v1alpha1.DMMonitorSpec{}
			for key, container := range map[string]core.Container{
				"TIDB_VERSION": getMonitorInitContainer(stored, nil),
				"DM_VERSION":   getMonitorDMInitContainer(stored, &v1alpha1.DMCluster{}),
			} {
				found := false
				for _, env := range container.Env {
					if env.Name == key {
						found = env.Value == version
					}
				}
				if !found {
					t.Fatalf("%s does not restore the reloader's persistence directory", key)
				}
			}
		})
	}
}

func TestMonitorRulesVersionExplicitSelection(t *testing.T) {
	m, tm := rulesVersionFixture(t)
	if err := m.syncMonitorRulesVersion(tm); err != nil {
		t.Fatal(err)
	}
	if getAlertManagerRulesVersion(tm) != "tidb:new-image-tag" {
		t.Fatal("new deployment did not initialize its rules version")
	}
	oldRulesWorkload(t, m, tm, "old-monitor", "tidb:old", false)
	tm.Spec.AlertManagerRulesVersion = pointer.String("chosen")
	if err := m.syncMonitorRulesVersion(tm); err != nil || getAlertManagerRulesVersion(tm) != "tidb:chosen" {
		t.Fatal("explicit rules selection did not override the recorded version", err)
	}
	tm.Spec.AlertManagerRulesVersion = nil
	tm.Spec.Initializer.Version = "next-image"
	if err := m.syncMonitorRulesVersion(tm); err != nil || getAlertManagerRulesVersion(tm) != "tidb:chosen" {
		t.Fatal("clearing explicit selection forgot the last selected directory", err)
	}
}

func TestMonitorRulesVersionRefusesToGuess(t *testing.T) {
	for _, scenario := range []string{"shards disagree", "unreadable workload", "missing old workload", "list fails", "record fails"} {
		t.Run(scenario, func(t *testing.T) {
			m, tm := rulesVersionFixture(t)
			switch scenario {
			case "shards disagree":
				oldRulesWorkload(t, m, tm, "shard-0", "tidb:old", false)
				oldRulesWorkload(t, m, tm, "shard-1", "tidb:other", false)
			case "unreadable workload":
				oldRulesWorkload(t, m, tm, "old-monitor", "", false)
			case "missing old workload":
				tm.Status.StatefulSet = &apps.StatefulSetStatus{}
			case "list fails":
				m.deps.KubeClientset.(*kubefake.Clientset).PrependReactor("list", "statefulsets", func(ktesting.Action) (bool, runtime.Object, error) {
					return true, nil, fmt.Errorf("API unavailable")
				})
			case "record fails":
				oldRulesWorkload(t, m, tm, "old-monitor", "tidb:old", false)
				m.deps.Clientset.(*clientfake.Clientset).PrependReactor("patch", "tidbmonitors", func(ktesting.Action) (bool, runtime.Object, error) {
					return true, nil, fmt.Errorf("conflict")
				})
			}
			if err := m.syncMonitorRulesVersion(tm); err == nil {
				t.Fatal("uncertain rules version or failed recording was accepted")
			}
			if tm.Annotations[label.AnnMonitorRulesVersion] != "" {
				t.Fatal("failed recording changed the in-memory version")
			}
		})
	}
}

func TestMonitorRulesVersionIgnoresOtherOwners(t *testing.T) {
	m, tm := rulesVersionFixture(t)
	other := tm.DeepCopy()
	other.UID = "other-monitor-uid"
	oldRulesWorkload(t, m, other, "unrelated", "tidb:other", false)
	oldRulesWorkload(t, m, tm, "shard-0", "tidb:old", false)
	oldRulesWorkload(t, m, tm, "shard-1", "tidb:old", false)
	if err := m.syncMonitorRulesVersion(tm); err != nil || getAlertManagerRulesVersion(tm) != "tidb:old" {
		t.Fatal("unrelated workload influenced the rules directory", err)
	}
}

func TestReloaderRulesVersionFlags(t *testing.T) {
	for _, test := range []struct {
		args []string
		want string
	}{
		{[]string{"--sub-store-path", "tidb:old"}, "tidb:old"},
		{[]string{"--sub-store-path=first", "--sub-store-path=custom/old"}, "custom/old"},
		{[]string{"--", "--sub-store-path=ignored"}, ""},
		{[]string{"--sub-store-path"}, ""},
		{[]string{"--sub-store-path", "--watch-path=rules"}, ""},
		{[]string{"--sub-store-path="}, ""},
	} {
		t.Run(strings.Join(test.args, " "), func(t *testing.T) {
			got, err := reloaderRulesVersion([]core.Container{{Name: "reloader", Command: []string{"/bin/reload"}, Args: test.args}})
			if got != test.want || (err == nil) != (test.want != "") {
				t.Fatalf("got %q, %v; want %q", got, err, test.want)
			}
		})
	}
}
