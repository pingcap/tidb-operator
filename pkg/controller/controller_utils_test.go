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

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
	g.Expect(err.Error()).To(Equal("i am a requeue error"))
	g.Expect(IsRequeueError(fmt.Errorf("i am not a requeue error"))).To(BeFalse())
}

func TestIgnoreError(t *testing.T) {
	g := NewGomegaWithT(t)

	err := IgnoreErrorf("i am an ignore %s", "error")
	g.Expect(IsIgnoreError(err)).To(BeTrue())
	g.Expect(err.Error()).To(Equal("i am an ignore error"))
	g.Expect(IsIgnoreError(fmt.Errorf("i am not an ignore error"))).To(BeFalse())
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

func TestGetDMOwnerRef(t *testing.T) {
	g := NewGomegaWithT(t)

	dc := newDMCluster()
	dc.UID = types.UID("demo-uid")
	ref := GetDMOwnerRef(dc)
	g.Expect(ref.APIVersion).To(Equal(DMControllerKind.GroupVersion().String()))
	g.Expect(ref.Kind).To(Equal(DMControllerKind.Kind))
	g.Expect(ref.Name).To(Equal(dc.GetName()))
	g.Expect(ref.UID).To(Equal(types.UID("demo-uid")))
	g.Expect(*ref.Controller).To(BeTrue())
	g.Expect(*ref.BlockOwnerDeletion).To(BeTrue())
}

func TestGetBackupOwnerRef(t *testing.T) {
	g := NewGomegaWithT(t)

	b := newBackup()
	b.UID = types.UID("demo-uid")
	ref := GetBackupOwnerRef(b)
	g.Expect(ref.APIVersion).To(Equal(BackupControllerKind.GroupVersion().String()))
	g.Expect(ref.Kind).To(Equal(BackupControllerKind.Kind))
	g.Expect(ref.Name).To(Equal(b.GetName()))
	g.Expect(ref.UID).To(Equal(types.UID("demo-uid")))
	g.Expect(*ref.Controller).To(BeTrue())
	g.Expect(*ref.BlockOwnerDeletion).To(BeTrue())
}

func TestGetRestoreOwnerRef(t *testing.T) {
	g := NewGomegaWithT(t)

	r := newRestore()
	r.UID = types.UID("demo-uid")
	ref := GetRestoreOwnerRef(r)
	g.Expect(ref.APIVersion).To(Equal(RestoreControllerKind.GroupVersion().String()))
	g.Expect(ref.Kind).To(Equal(RestoreControllerKind.Kind))
	g.Expect(ref.Name).To(Equal(r.GetName()))
	g.Expect(ref.UID).To(Equal(types.UID("demo-uid")))
	g.Expect(*ref.Controller).To(BeTrue())
	g.Expect(*ref.BlockOwnerDeletion).To(BeTrue())
}

func TestGetBackupScheduleOwnerRef(t *testing.T) {
	g := NewGomegaWithT(t)

	b := newBackupSchedule()
	b.UID = types.UID("demo-uid")
	ref := GetBackupScheduleOwnerRef(b)
	g.Expect(ref.APIVersion).To(Equal(backupScheduleControllerKind.GroupVersion().String()))
	g.Expect(ref.Kind).To(Equal(backupScheduleControllerKind.Kind))
	g.Expect(ref.Name).To(Equal(b.GetName()))
	g.Expect(ref.UID).To(Equal(types.UID("demo-uid")))
	g.Expect(*ref.Controller).To(BeTrue())
	g.Expect(*ref.BlockOwnerDeletion).To(BeTrue())
}

func TestGetTiDBNGMonitoringOwnerRef(t *testing.T) {
	g := NewGomegaWithT(t)

	tngm := newTidbNGMonitoring()
	tngm.UID = types.UID("demo-uid")
	ref := GetTiDBNGMonitoringOwnerRef(tngm)
	g.Expect(ref.APIVersion).To(Equal(tidbNGMonitoringKind.GroupVersion().String()))
	g.Expect(ref.Kind).To(Equal(tidbNGMonitoringKind.Kind))
	g.Expect(ref.Name).To(Equal(tngm.GetName()))
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
	g.Expect(MemberName("demo", v1alpha1.PDMemberType)).To(Equal("demo-pd"))
}

func TestPDPeerMemberName(t *testing.T) {
	g := NewGomegaWithT(t)
	g.Expect(PDPeerMemberName("demo")).To(Equal("demo-pd-peer"))
}

func TestTiKVMemberName(t *testing.T) {
	g := NewGomegaWithT(t)
	g.Expect(TiKVMemberName("demo")).To(Equal("demo-tikv"))
	g.Expect(MemberName("demo", v1alpha1.TiKVMemberType)).To(Equal("demo-tikv"))
}

func TestTiKVPeerMemberName(t *testing.T) {
	g := NewGomegaWithT(t)
	g.Expect(TiKVPeerMemberName("demo")).To(Equal("demo-tikv-peer"))
}

func TestTiDBMemberName(t *testing.T) {
	g := NewGomegaWithT(t)
	g.Expect(TiDBMemberName("demo")).To(Equal("demo-tidb"))
	g.Expect(MemberName("demo", v1alpha1.TiDBMemberType)).To(Equal("demo-tidb"))
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
	g.Expect(MemberName("demo", v1alpha1.PumpMemberType)).To(Equal("demo-pump"))
}

func TestDiscoveryMemberName(t *testing.T) {
	g := NewGomegaWithT(t)
	g.Expect(DiscoveryMemberName("demo")).To(Equal("demo-discovery"))
}

func TestDMMasterMemberName(t *testing.T) {
	g := NewGomegaWithT(t)
	g.Expect(DMMasterMemberName("demo")).To(Equal("demo-dm-master"))
	g.Expect(MemberName("demo", v1alpha1.DMMasterMemberType)).To(Equal("demo-dm-master"))
}

func TestDMMasterPeerMemberName(t *testing.T) {
	g := NewGomegaWithT(t)
	g.Expect(DMMasterPeerMemberName("demo")).To(Equal("demo-dm-master-peer"))
}

func TestDMWorkerMemberName(t *testing.T) {
	g := NewGomegaWithT(t)
	g.Expect(DMWorkerMemberName("demo")).To(Equal("demo-dm-worker"))
	g.Expect(MemberName("demo", v1alpha1.DMWorkerMemberType)).To(Equal("demo-dm-worker"))
}

func TestDMWorkerPeerMemberName(t *testing.T) {
	g := NewGomegaWithT(t)
	g.Expect(DMWorkerPeerMemberName("demo")).To(Equal("demo-dm-worker-peer"))
}

func TestAnnProm(t *testing.T) {
	g := NewGomegaWithT(t)

	ann := AnnProm(int32(9090), "/metrics")
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

func TestEmptyClone(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name  string
		obj   client.Object
		empty client.Object
	}

	cases := []testcase{
		{
			name:  "tidb-cluster",
			obj:   newTidbCluster(),
			empty: &v1alpha1.TidbCluster{},
		},
		{
			name:  "dm-cluster",
			obj:   newDMCluster(),
			empty: &v1alpha1.DMCluster{},
		},
		{
			name:  "tidb-ng-monitoring",
			obj:   newTidbNGMonitoring(),
			empty: &v1alpha1.TidbNGMonitoring{},
		},
		{
			name:  "backup",
			obj:   newBackup(),
			empty: &v1alpha1.Backup{},
		},
		{
			name:  "stateful-set",
			obj:   newStatefulSet(newTidbCluster(), ""),
			empty: &apps.StatefulSet{},
		},
		{
			name:  "service",
			obj:   newService(newTidbCluster(), ""),
			empty: &corev1.Service{},
		},
	}

	for _, tcase := range cases {
		t.Log(tcase.name)

		tcase.empty.SetName(tcase.obj.GetName())
		tcase.empty.SetNamespace(tcase.obj.GetNamespace())

		output, err := EmptyClone(tcase.obj)
		g.Expect(err).Should(Succeed())
		g.Expect(output).Should(Equal(tcase.empty))
	}
}

func TestDeepCopyClientObject(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name string
		obj  client.Object
	}

	cases := []testcase{
		{
			name: "tidb-cluster",
			obj:  newTidbCluster(),
		},
		{
			name: "dm-cluster",
			obj:  newDMCluster(),
		},
		{
			name: "tidb-ng-monitoring",
			obj:  newTidbNGMonitoring(),
		},
		{
			name: "backup",
			obj:  newBackup(),
		},
		{
			name: "stateful-set",
			obj:  newStatefulSet(newTidbCluster(), ""),
		},
		{
			name: "service",
			obj:  newService(newTidbCluster(), ""),
		},
	}

	for _, tcase := range cases {
		t.Log(tcase.name)

		obj := DeepCopyClientObject(tcase.obj)
		g.Expect(obj == tcase.obj).Should(BeFalse()) // compare pointer
		g.Expect(obj).Should(Equal(tcase.obj))       // compare content
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

func TestTrimName(t *testing.T) {
	g := NewGomegaWithT(t)
	name := "basic-tso-peer"
	g.Expect(PDMSTrimName(name)).To(Equal("tso"))

	name = "basic-tso"
	g.Expect(PDMSTrimName(name)).To(Equal("tso"))
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
			PD:              &v1alpha1.PDSpec{},
			TiKV:            &v1alpha1.TiKVSpec{},
			TiDB:            &v1alpha1.TiDBSpec{},
			PVReclaimPolicy: &retainPVP,
		},
	}
	return tc
}

func newDMCluster() *v1alpha1.DMCluster {
	retainPVP := corev1.PersistentVolumeReclaimRetain
	dc := &v1alpha1.DMCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: v1alpha1.DMClusterSpec{
			Version:         "v2.0.0-rc.2",
			Master:          v1alpha1.MasterSpec{},
			Worker:          &v1alpha1.WorkerSpec{},
			PVReclaimPolicy: &retainPVP,
		},
	}
	return dc
}

func newTidbNGMonitoring() *v1alpha1.TidbNGMonitoring {
	tngm := &v1alpha1.TidbNGMonitoring{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: v1alpha1.TidbNGMonitoringSpec{},
	}

	return tngm
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
		Spec: v1alpha1.BackupSpec{
			From: &v1alpha1.TiDBAccessConfig{},
		},
	}
	return backup
}

func newRestore() *v1alpha1.Restore {
	restore := &v1alpha1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-backup",
			Namespace: metav1.NamespaceDefault,
			Labels: map[string]string{
				label.RestoreJobLabelVal: "test-job",
			},
		},
		Spec: v1alpha1.RestoreSpec{
			To: &v1alpha1.TiDBAccessConfig{},
		},
	}
	return restore
}

func newBackupSchedule() *v1alpha1.BackupSchedule {
	backup := &v1alpha1.BackupSchedule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-backup",
			Namespace: metav1.NamespaceDefault,
			Labels: map[string]string{
				label.BackupScheduleLabelKey: "test-schedule",
			},
		},
		Spec: v1alpha1.BackupScheduleSpec{
			BackupTemplate: v1alpha1.BackupSpec{
				From: &v1alpha1.TiDBAccessConfig{},
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
