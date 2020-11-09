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

package util

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	fuzz "github.com/google/gofuzz"
	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/pointer"
)

func TestGetOrdinalFromPodName(t *testing.T) {
	g := NewGomegaWithT(t)

	i, err := GetOrdinalFromPodName("pod-1")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(i).To(Equal(int32(1)))

	i, err = GetOrdinalFromPodName("pod-notint")
	g.Expect(err).To(HaveOccurred())
	g.Expect(i).To(Equal(int32(0)))
}

func TestIsSubMapOf(t *testing.T) {
	g := NewGomegaWithT(t)

	g.Expect(IsSubMapOf(
		nil,
		nil)).To(BeTrue())
	g.Expect(IsSubMapOf(
		nil,
		map[string]string{
			"k1": "v1",
		})).To(BeTrue())
	g.Expect(IsSubMapOf(
		map[string]string{
			"k1": "v1",
		},
		nil)).To(BeFalse())
	g.Expect(IsSubMapOf(
		map[string]string{
			"k1": "v1",
		},
		map[string]string{
			"k1": "v1",
		})).To(BeTrue())
	g.Expect(IsSubMapOf(
		map[string]string{
			"k1": "v1",
		},
		map[string]string{
			"k1": "v1",
			"k2": "v2",
		})).To(BeTrue())
	g.Expect(IsSubMapOf(
		map[string]string{},
		map[string]string{
			"k1": "v1",
		})).To(BeTrue())
	g.Expect(IsSubMapOf(
		map[string]string{
			"k1": "v1",
			"k2": "v2",
		},
		map[string]string{
			"k1": "v1",
		})).To(BeFalse())
}

func TestGetPodOrdinals(t *testing.T) {
	tests := []struct {
		name        string
		tc          *v1alpha1.TidbCluster
		memberType  v1alpha1.MemberType
		deleteSlots sets.Int32
	}{
		{
			name: "no delete slots",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{
						Replicas: 3,
					},
				},
			},
			memberType:  v1alpha1.TiDBMemberType,
			deleteSlots: sets.NewInt32(0, 1, 2),
		},
		{
			name: "delete slots",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						label.AnnTiDBDeleteSlots: "[1,2]",
					},
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{
						Replicas: 3,
					},
				},
			},
			memberType:  v1alpha1.TiDBMemberType,
			deleteSlots: sets.NewInt32(0, 3, 4),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetPodOrdinals(tt.tc, tt.memberType)
			if err != nil {
				t.Error(err)
			}
			if !got.Equal(tt.deleteSlots) {
				t.Errorf("expects %v got %v", tt.deleteSlots.List(), got.List())
			}
		})
	}
}

func TestGetAutoScalingOutSlots(t *testing.T) {
	g := NewGomegaWithT(t)
	slice := []int32{1, 2}
	sliceData, err := json.Marshal(slice)
	sliceString := string(sliceData)
	g.Expect(err).Should(BeNil())
	tc := &v1alpha1.TidbCluster{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				label.AnnTiKVAutoScalingOutOrdinals: sliceString,
				label.AnnTiDBAutoScalingOutOrdinals: sliceString,
			},
		},
	}

	var get sets.Int32
	get = GetAutoScalingOutSlots(tc, v1alpha1.PDMemberType)
	g.Expect(get).Should(Equal(sets.Int32{}))

	get = GetAutoScalingOutSlots(tc, v1alpha1.TiKVMemberType)
	g.Expect(get).Should(Equal(sets.NewInt32(slice...)))

	get = GetAutoScalingOutSlots(tc, v1alpha1.TiDBMemberType)
	g.Expect(get).Should(Equal(sets.NewInt32(slice...)))
}

func TestName(t *testing.T) {
	tcName := "test_cluster"
	com := "com_name"
	var name string
	g := NewGomegaWithT(t)

	name = ClusterClientTLSSecretName(tcName)
	g.Expect(name).Should(Equal(tcName + "-cluster-client-secret"))

	name = ClusterTLSSecretName(tcName, com)
	g.Expect(name).Should(Equal(tcName + "-" + com + "-cluster-secret"))

	name = TiDBClientTLSSecretName(tcName)
	g.Expect(name).Should(Equal(tcName + "-tidb-client-secret"))
}

func TestSortEnvByName(t *testing.T) {
	f := fuzz.New().NilChance(0.0)
	for i := 0; i < 10; i++ {
		var envs []corev1.EnvVar
		f.Fuzz(&envs)

		sort.Sort(SortEnvByName(envs))
		// check sorted by name
		for i := 1; i < len(envs); i++ {
			if envs[i].Name < envs[i-1].Name {
				t.Fatal(envs, "not sorted by name")
			}
		}
	}
}

func TestAppendEnv(t *testing.T) {
	tests := []struct {
		name string
		a    []corev1.EnvVar
		b    []corev1.EnvVar
		want []corev1.EnvVar
	}{
		{
			name: "envs whose names exist are ignored",
			a: []corev1.EnvVar{
				{
					Name:  "foo",
					Value: "bar",
				},
				{
					Name:  "xxx",
					Value: "xxx",
				},
			},
			b: []corev1.EnvVar{
				{
					Name:  "foo",
					Value: "barbar",
				},
				{
					Name:  "new",
					Value: "bar",
				},
				{
					Name:  "xxx",
					Value: "yyy",
				},
			},
			want: []corev1.EnvVar{
				{
					Name:  "foo",
					Value: "bar",
				},
				{
					Name:  "xxx",
					Value: "xxx",
				},
				{
					Name:  "new",
					Value: "bar",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AppendEnv(tt.a, tt.b)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("unwant (-want, +got): %s", diff)
			}
		})
	}
}

func TestAppendEnvIfPresent(t *testing.T) {
	tests := []struct {
		name string
		a    []corev1.EnvVar
		envs map[string]string
		n    string
		want []corev1.EnvVar
	}{
		{
			"does not exist",
			[]corev1.EnvVar{
				{
					Name:  "foo",
					Value: "bar",
				},
			},
			nil,
			"TEST_ENV",
			[]corev1.EnvVar{
				{
					Name:  "foo",
					Value: "bar",
				},
			},
		},
		{
			"does exist",
			[]corev1.EnvVar{
				{
					Name:  "foo",
					Value: "bar",
				},
			},
			map[string]string{
				"TEST_ENV": "TEST_VAL",
			},
			"TEST_ENV",
			[]corev1.EnvVar{
				{
					Name:  "foo",
					Value: "bar",
				},
				{
					Name:  "TEST_ENV",
					Value: "TEST_VAL",
				},
			},
		},
		{
			"already exist",
			[]corev1.EnvVar{
				{
					Name:  "foo",
					Value: "bar",
				},
				{
					Name:  "TEST_ENV",
					Value: "TEST_OLD_VAL",
				},
			},
			map[string]string{
				"TEST_ENV": "TEST_VAL",
			},
			"TEST_ENV",
			[]corev1.EnvVar{
				{
					Name:  "foo",
					Value: "bar",
				},
				{
					Name:  "TEST_ENV",
					Value: "TEST_OLD_VAL",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			for k, v := range tt.envs {
				os.Setenv(k, v)
			}
			got := AppendEnvIfPresent(tt.a, tt.n)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("unwant (-want, +got): %s", diff)
			}
		})
	}
}

func TestAppendOverwriteEnv(t *testing.T) {
	g := NewGomegaWithT(t)

	a := []corev1.EnvVar{
		{
			Name:  "ak_1",
			Value: "ak_1",
		},
		{
			Name:  "ak_2",
			Value: "ak_2",
		},
	}
	b := []corev1.EnvVar{
		{
			Name:  "bk_1",
			Value: "bk_1",
		},
		{
			Name:  "ak_1",
			Value: "ak_10",
		},
		{
			Name:  "ak_2",
			Value: "ak_20",
		},
		{
			Name:  "bk_2",
			Value: "bk_2",
		},
	}

	expect := []corev1.EnvVar{
		{
			Name:  "ak_1",
			Value: "ak_10",
		},
		{
			Name:  "ak_2",
			Value: "ak_20",
		},
		{
			Name:  "bk_1",
			Value: "bk_1",
		},
		{
			Name:  "bk_2",
			Value: "bk_2",
		},
	}

	get := AppendOverwriteEnv(a, b)
	g.Expect(get).Should(Equal(expect))
}

func TestMustNewRequirement(t *testing.T) {
	g := NewGomegaWithT(t)
	var r *labels.Requirement

	// test panic
	g.Expect(func() {
		_ = MustNewRequirement("key", selection.Operator("un known"), nil)
	}).Should(Panic())

	// test normal case
	r = MustNewRequirement("key", selection.Equals, []string{"value"})
	g.Expect(r).ShouldNot(BeNil())
}

func TestIsOwnedByTidbCluster(t *testing.T) {

}

func TestRetainManagedFields(t *testing.T) {
	tests := []struct {
		name       string
		desiredSvc *corev1.Service
		existedSvc *corev1.Service
		expect     *corev1.Service
	}{
		{
			name:       "test keep HealthCheckNodePort",
			desiredSvc: &corev1.Service{},
			existedSvc: &corev1.Service{
				Spec: corev1.ServiceSpec{
					HealthCheckNodePort: 10,
				},
			},
			expect: &corev1.Service{
				Spec: corev1.ServiceSpec{
					HealthCheckNodePort: 10,
				},
			},
		},
		{
			name: "test keep retain NodePorts",
			desiredSvc: &corev1.Service{
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeNodePort,
					Ports: []corev1.ServicePort{
						corev1.ServicePort{
							NodePort: 8080,
						},
						corev1.ServicePort{
							NodePort: 0,
							Port:     10,
							Protocol: corev1.ProtocolTCP,
						},
						corev1.ServicePort{
							NodePort: 30,
							Port:     20,
							Protocol: corev1.ProtocolTCP,
						},
					},
				},
			},
			existedSvc: &corev1.Service{
				Spec: corev1.ServiceSpec{
					Type:                corev1.ServiceTypeNodePort,
					HealthCheckNodePort: 10,
					Ports: []corev1.ServicePort{
						corev1.ServicePort{
							NodePort: 9090,
							Port:     10,
							Protocol: corev1.ProtocolTCP,
						},
					},
				},
			},
			expect: &corev1.Service{
				Spec: corev1.ServiceSpec{
					Type:                corev1.ServiceTypeNodePort,
					HealthCheckNodePort: 10,
					Ports: []corev1.ServicePort{
						corev1.ServicePort{
							NodePort: 8080,
						},
						corev1.ServicePort{
							NodePort: 9090,
							Port:     10,
							Protocol: corev1.ProtocolTCP,
						},
						corev1.ServicePort{
							NodePort: 30,
							Port:     20,
							Protocol: corev1.ProtocolTCP,
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		RetainManagedFields(test.desiredSvc, test.existedSvc)
		if diff := cmp.Diff(test.expect.Spec, test.desiredSvc.Spec); diff != "" {
			t.Errorf("%v unwant (-want, +got): %s", test.name, diff)
		}
	}
}

func TestBuildAdditionalVolumeAndVolumeMount(t *testing.T) {
	tests := []struct {
		name             string
		storageVolumes   []v1alpha1.StorageVolume
		storageClassName *string
		memberType       v1alpha1.MemberType
		testResult       func([]corev1.VolumeMount, []corev1.PersistentVolumeClaim)
	}{
		{
			name:             "unknown memberType",
			storageVolumes:   []v1alpha1.StorageVolume{},
			memberType:       "test",
			storageClassName: nil,
			testResult: func(volMounts []corev1.VolumeMount, volumeClaims []corev1.PersistentVolumeClaim) {
				g := NewGomegaWithT(t)
				g.Expect(volMounts).Should(BeNil())
				g.Expect(volumeClaims).Should(BeNil())
			},
		},
		{
			name: "tidb spec storageVolumes",
			storageVolumes: []v1alpha1.StorageVolume{
				{
					Name:        "log",
					StorageSize: "2Gi",
					MountPath:   "/var/lib/log",
				}},
			memberType: v1alpha1.TiDBMemberType,
			testResult: func(volMounts []corev1.VolumeMount, volumeClaims []corev1.PersistentVolumeClaim) {
				g := NewGomegaWithT(t)
				q, _ := resource.ParseQuantity("2Gi")
				g.Expect(volumeClaims).To(Equal([]corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: v1alpha1.TiDBMemberType.String() + "-log",
						},
						Spec: corev1.PersistentVolumeClaimSpec{
							AccessModes: []corev1.PersistentVolumeAccessMode{
								corev1.ReadWriteOnce,
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: q,
								},
							},
						},
					},
				}))
				g.Expect(volMounts).To(Equal([]corev1.VolumeMount{
					{
						Name: fmt.Sprintf("%s-%s", v1alpha1.TiDBMemberType, "log"), MountPath: "/var/lib/log",
					},
				}))
			},
		},
		{
			name: "tikv spec storageVolumes",
			storageVolumes: []v1alpha1.StorageVolume{
				{
					Name:        "wal",
					StorageSize: "2Gi",
					MountPath:   "/var/lib/wal",
				}},
			memberType: v1alpha1.TiKVMemberType,
			testResult: func(volMounts []corev1.VolumeMount, volumeClaims []corev1.PersistentVolumeClaim) {
				g := NewGomegaWithT(t)
				q, _ := resource.ParseQuantity("2Gi")
				g.Expect(volumeClaims).To(Equal([]corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: v1alpha1.TiKVMemberType.String() + "-wal",
						},
						Spec: corev1.PersistentVolumeClaimSpec{
							AccessModes: []corev1.PersistentVolumeAccessMode{
								corev1.ReadWriteOnce,
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: q,
								},
							},
						},
					},
				}))
				g.Expect(volMounts).To(Equal([]corev1.VolumeMount{
					{
						Name: fmt.Sprintf("%s-%s", v1alpha1.TiKVMemberType, "wal"), MountPath: "/var/lib/wal",
					},
				}))
			},
		},
		{
			name: "pd spec storageVolumes",
			storageVolumes: []v1alpha1.StorageVolume{
				{
					Name:        "log",
					StorageSize: "2Gi",
					MountPath:   "/var/log",
				}},
			memberType: v1alpha1.PDMemberType,
			testResult: func(volMounts []corev1.VolumeMount, volumeClaims []corev1.PersistentVolumeClaim) {
				g := NewGomegaWithT(t)
				q, _ := resource.ParseQuantity("2Gi")
				g.Expect(volumeClaims).To(Equal([]corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: v1alpha1.PDMemberType.String() + "-log",
						},
						Spec: corev1.PersistentVolumeClaimSpec{
							AccessModes: []corev1.PersistentVolumeAccessMode{
								corev1.ReadWriteOnce,
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: q,
								},
							},
						},
					},
				}))
				g.Expect(volMounts).To(Equal([]corev1.VolumeMount{
					{
						Name: fmt.Sprintf("%s-%s", v1alpha1.PDMemberType, "log"), MountPath: "/var/log",
					},
				}))
			},
		},
		{
			name:             "tikv spec multiple storageVolumes",
			storageClassName: pointer.StringPtr("ns2"),
			storageVolumes: []v1alpha1.StorageVolume{
				{
					Name:             "wal",
					StorageSize:      "2Gi",
					MountPath:        "/var/lib/wal",
					StorageClassName: pointer.StringPtr("ns1"),
				},
				{
					Name:        "log",
					StorageSize: "2Gi",
					MountPath:   "/var/lib/log",
				}},
			memberType: v1alpha1.TiKVMemberType,
			testResult: func(volMounts []corev1.VolumeMount, volumeClaims []corev1.PersistentVolumeClaim) {
				g := NewGomegaWithT(t)
				q, _ := resource.ParseQuantity("2Gi")
				g.Expect(volumeClaims).To(Equal([]corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: v1alpha1.TiKVMemberType.String() + "-wal",
						},
						Spec: corev1.PersistentVolumeClaimSpec{
							AccessModes: []corev1.PersistentVolumeAccessMode{
								corev1.ReadWriteOnce,
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: q,
								},
							},
							StorageClassName: pointer.StringPtr("ns1"),
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: v1alpha1.TiKVMemberType.String() + "-log",
						},
						Spec: corev1.PersistentVolumeClaimSpec{
							AccessModes: []corev1.PersistentVolumeAccessMode{
								corev1.ReadWriteOnce,
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: q,
								},
							},
							StorageClassName: pointer.StringPtr("ns2"),
						},
					},
				}))

			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			volMounts, volumeClaims := BuildAdditionalVolumeAndVolumeMount(tt.storageVolumes, tt.storageClassName, tt.memberType)
			tt.testResult(volMounts, volumeClaims)
		})
	}
}
