// Copyright 2018 PingCAP, Inc.
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

package controller

import (
	"fmt"
	"testing"

	"github.com/pingcap/tidb-operator/pkg/label"
	"k8s.io/apimachinery/pkg/api/resource"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	apps "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestRequeueError(t *testing.T) {
	g := NewGomegaWithT(t)

	err := RequeueErrorf("i am a requeue %s", "error")
	g.Expect(IsRequeueError(err)).To(BeTrue())
	_, ok := err.(error)
	g.Expect(ok).To(BeTrue())
	g.Expect(err.Error()).To(Equal("i am a requeue error"))
	g.Expect(IsRequeueError(fmt.Errorf("i am not a requeue error"))).To(BeFalse())
}

func TestGetOwnerRef(t *testing.T) {
	g := NewGomegaWithT(t)

	tc := newTidbCluster()
	tc.UID = types.UID("demo-uid")
	ref := GetOwnerRef(tc)
	g.Expect(ref.APIVersion).To(Equal(ControllerKind.GroupVersion().String()))
	g.Expect(ref.Kind).To(Equal(ControllerKind.Kind))
	g.Expect(ref.Name).To(Equal(tc.GetName()))
	g.Expect(ref.UID).To(Equal(types.UID("demo-uid")))
	g.Expect(*ref.Controller).To(BeTrue())
	g.Expect(*ref.BlockOwnerDeletion).To(BeTrue())
}

func TestGetServiceType(t *testing.T) {
	g := NewGomegaWithT(t)

	services := []v1alpha1.Service{
		{
			Name: "a",
			Type: string(corev1.ServiceTypeNodePort),
		},
		{
			Name: "b",
			Type: string(corev1.ServiceTypeLoadBalancer),
		},
		{
			Name: "c",
			Type: "Other",
		},
	}

	g.Expect(GetServiceType(services, "a")).To(Equal(corev1.ServiceTypeNodePort))
	g.Expect(GetServiceType(services, "b")).To(Equal(corev1.ServiceTypeLoadBalancer))
	g.Expect(GetServiceType(services, "c")).To(Equal(corev1.ServiceTypeClusterIP))
	g.Expect(GetServiceType(services, "d")).To(Equal(corev1.ServiceTypeClusterIP))
}

func TestTiKVCapacity(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name     string
		limit    corev1.ResourceList
		expectFn func(*GomegaWithT, string)
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		test.expectFn(g, TiKVCapacity(test.limit))
	}
	tests := []testcase{
		{
			name:  "limit is nil",
			limit: nil,
			expectFn: func(g *GomegaWithT, s string) {
				g.Expect(s).To(Equal("0"))
			},
		},
		{
			name:  "storage is empty",
			limit: corev1.ResourceList{},
			expectFn: func(g *GomegaWithT, s string) {
				g.Expect(s).To(Equal("0"))
			},
		},
		{
			name: "100Gi",
			limit: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("100Gi"),
			},
			expectFn: func(g *GomegaWithT, s string) {
				g.Expect(s).To(Equal("100GB"))
			},
		},
		{
			name: "1G",
			limit: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("1G"),
			},
			expectFn: func(g *GomegaWithT, s string) {
				g.Expect(s).To(Equal("953MB"))
			},
		},
		{
			name: "1.5G",
			limit: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("1.5G"),
			},
			expectFn: func(g *GomegaWithT, s string) {
				g.Expect(s).To(Equal("1430MB"))
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestPDMemberName(t *testing.T) {
	g := NewGomegaWithT(t)
	g.Expect(PDMemberName("demo")).To(Equal("demo-pd"))
}

func TestPDPeerMemberName(t *testing.T) {
	g := NewGomegaWithT(t)
	g.Expect(PDPeerMemberName("demo")).To(Equal("demo-pd-peer"))
}

func TestTiKVMemberName(t *testing.T) {
	g := NewGomegaWithT(t)
	g.Expect(TiKVMemberName("demo")).To(Equal("demo-tikv"))
}

func TestTiKVPeerMemberName(t *testing.T) {
	g := NewGomegaWithT(t)
	g.Expect(TiKVPeerMemberName("demo")).To(Equal("demo-tikv-peer"))
}

func TestTiDBMemberName(t *testing.T) {
	g := NewGomegaWithT(t)
	g.Expect(TiDBMemberName("demo")).To(Equal("demo-tidb"))
}

func TestTiDBPeerMemberName(t *testing.T) {
	g := NewGomegaWithT(t)
	g.Expect(TiDBPeerMemberName("demo")).To(Equal("demo-tidb-peer"))
}

func TestPumpMemberName(t *testing.T) {
	g := NewGomegaWithT(t)
	g.Expect(PumpMemberName("demo")).To(Equal("demo-pump"))
}

func TestPumpPeerMemberName(t *testing.T) {
	g := NewGomegaWithT(t)
	g.Expect(PumpPeerMemberName("demo")).To(Equal("demo-pump"))
}

func TestDiscoveryMemberName(t *testing.T) {
	g := NewGomegaWithT(t)
	g.Expect(DiscoveryMemberName("demo")).To(Equal("demo-discovery"))
}

func TestAnnProm(t *testing.T) {
	g := NewGomegaWithT(t)

	ann := AnnProm(int32(9090))
	g.Expect(ann["prometheus.io/scrape"]).To(Equal("true"))
	g.Expect(ann["prometheus.io/path"]).To(Equal("/metrics"))
	g.Expect(ann["prometheus.io/port"]).To(Equal("9090"))
}

func TestMemberConfigMapName(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name        string
		annotations map[string]string
		tcName      string
		member      v1alpha1.MemberType
		expectFn    func(*GomegaWithT, string)
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		tc := &v1alpha1.TidbCluster{}
		tc.Name = test.tcName
		tc.Annotations = test.annotations
		test.expectFn(g, MemberConfigMapName(tc, test.member))
	}
	tests := []testcase{
		{
			name:        "backward compatible when no annotations set",
			annotations: map[string]string{},
			tcName:      "cluster-name",
			member:      v1alpha1.TiKVMemberType,
			expectFn: func(g *GomegaWithT, s string) {
				g.Expect(s).To(Equal("cluster-name-tikv"))
			},
		},
		{
			name: "configmap digest presented",
			annotations: map[string]string{
				"pingcap.com/tikv.cluster-name-tikv.sha": "uuuuuuuu",
			},
			tcName: "cluster-name",
			member: v1alpha1.TiKVMemberType,
			expectFn: func(g *GomegaWithT, s string) {
				g.Expect(s).To(Equal("cluster-name-tikv-uuuuuuuu"))
			},
		},
		{
			name:        "nil annotations",
			annotations: nil,
			tcName:      "cluster-name",
			member:      v1alpha1.TiKVMemberType,
			expectFn: func(g *GomegaWithT, s string) {
				g.Expect(s).To(Equal("cluster-name-tikv"))
			},
		},
		{
			name: "annotation presented with empty value empty",
			annotations: map[string]string{
				"pingcap.com/tikv.cluster-name-tikv.sha": "",
			},
			tcName: "cluster-name",
			member: v1alpha1.TiKVMemberType,
			expectFn: func(g *GomegaWithT, s string) {
				g.Expect(s).To(Equal("cluster-name-tikv"))
			},
		},
		{
			name: "no matched annotation key",
			annotations: map[string]string{
				"pingcap.com/pd.cluster-name-tikv.sha": "",
			},
			tcName: "cluster-name",
			member: v1alpha1.TiKVMemberType,
			expectFn: func(g *GomegaWithT, s string) {
				g.Expect(s).To(Equal("cluster-name-tikv"))
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestSetIfNotEmpty(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name     string
		key      string
		value    string
		expectFn func(*GomegaWithT, map[string]string)
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		m := map[string]string{"a": "a"}
		setIfNotEmpty(m, test.key, test.value)

		test.expectFn(g, m)
	}
	tests := []testcase{
		{
			name:  "has key",
			key:   "a",
			value: "aa",
			expectFn: func(g *GomegaWithT, m map[string]string) {
				g.Expect(m["a"]).To(Equal("aa"))
			},
		},
		{
			name:  "don't have key",
			key:   "b",
			value: "b",
			expectFn: func(g *GomegaWithT, m map[string]string) {
				g.Expect(m["b"]).To(Equal("b"))
			},
		},
		{
			name:  "new key's value is empty",
			key:   "b",
			value: "",
			expectFn: func(g *GomegaWithT, m map[string]string) {
				g.Expect(m["b"]).To(Equal(""))
			},
		},
		{
			name:  "old key's value is empty",
			key:   "a",
			value: "",
			expectFn: func(g *GomegaWithT, m map[string]string) {
				g.Expect(m["a"]).To(Equal("a"))
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func collectEvents(source <-chan string) []string {
	done := false
	events := make([]string, 0)
	for !done {
		select {
		case event := <-source:
			events = append(events, event)
		default:
			done = true
		}
	}
	return events
}

func newTidbCluster() *v1alpha1.TidbCluster {
	retainPVP := corev1.PersistentVolumeReclaimRetain
	tc := &v1alpha1.TidbCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: v1alpha1.TidbClusterSpec{
			PVReclaimPolicy: &retainPVP,
		},
	}
	return tc
}

func newService(tc *v1alpha1.TidbCluster, _ string) *corev1.Service {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetName(tc.Name, "pd"),
			Namespace: metav1.NamespaceDefault,
		},
	}
	return svc
}

func newBackup() *v1alpha1.Backup {
	backup := &v1alpha1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-backup",
			Namespace: metav1.NamespaceDefault,
			Labels: map[string]string{
				label.BackupScheduleLabelKey: "test-schedule",
			},
		},
	}
	return backup
}

func newPVCFromBackup(backup *v1alpha1.Backup) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: "v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      backup.GetBackupPVCName(),
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("test"),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			VolumeName: "backup-pv-1",
		},
	}
}

func newJobFromBackup(backup *v1alpha1.Backup) *batchv1.Job {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backup.GetBackupJobName(),
			Namespace: metav1.NamespaceDefault,
		},
	}
	return job
}

func newStatefulSet(tc *v1alpha1.TidbCluster, _ string) *apps.StatefulSet {
	set := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetName(tc.Name, "pd"),
			Namespace: metav1.NamespaceDefault,
		},
	}
	return set
}

// GetName concatenate tidb cluster name and member name, used for controller managed resource name
func GetName(tcName string, name string) string {
	return fmt.Sprintf("%s-%s", tcName, name)
}
