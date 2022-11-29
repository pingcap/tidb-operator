// Copyright 2022 PingCAP, Inc.
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

package tidbdashboard

import (
	"fmt"
	"testing"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	mngerutils "github.com/pingcap/tidb-operator/pkg/manager/utils"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestManager(t *testing.T) {

	t.Run("syncService", func(t *testing.T) {
		g := NewGomegaWithT(t)

		type testcase struct {
			name string

			setInputs func(td *v1alpha1.TidbDashboard)

			getServices      func() (*corev1.Service /*old*/, *corev1.Service /*new*/)
			createServiceErr error
			updateServiceErr error
			expectErr        func(err error)
		}

		cases := []testcase{
			{
				name: "create service if it isn't found",
				getServices: func() (*corev1.Service, *corev1.Service) {
					return nil, &corev1.Service{}
				},
				createServiceErr: fmt.Errorf("should create service"),
				updateServiceErr: fmt.Errorf("shouldn't update service"),
				expectErr: func(err error) {
					g.Expect(err).Should(HaveOccurred())
					g.Expect(err.Error()).Should(ContainSubstring("should create service"))
				},
			},
			{
				name: "shouldn't update equal service",
				getServices: func() (*corev1.Service, *corev1.Service) {
					old := &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "service",
							Namespace: "default",
						},
						Spec: corev1.ServiceSpec{
							ClusterIP: "10.0.0.0",
						},
					}
					newly := old.DeepCopy()
					_ = controller.SetServiceLastAppliedConfigAnnotation(old)

					return old, newly
				},
				createServiceErr: fmt.Errorf("shouldn't create service"),
				updateServiceErr: fmt.Errorf("shouldn't update service"),
				expectErr: func(err error) {
					g.Expect(err).Should(Succeed())
				},
			},
			{
				name: "should update different service",
				getServices: func() (*corev1.Service, *corev1.Service) {
					old := &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "service",
							Namespace: "default",
						},
						Spec: corev1.ServiceSpec{
							ClusterIP: "10.0.0.0",
						},
					}
					newly := old.DeepCopy()
					_ = controller.SetServiceLastAppliedConfigAnnotation(old)
					newly.Spec.ClusterIP = "10.0.0.1"

					return old, newly
				},
				createServiceErr: fmt.Errorf("shouldn't create service"),
				updateServiceErr: fmt.Errorf("should update service"),
				expectErr: func(err error) {
					g.Expect(err).Should(HaveOccurred())
					g.Expect(err.Error()).Should(ContainSubstring("should update service"))
				},
			},
		}

		runCase := func(testcase testcase) {
			t.Logf("testcase: %s", testcase.name)

			deps := controller.NewFakeDependencies()
			indexer := deps.KubeInformerFactory.Core().V1().Services().Informer().GetIndexer()

			manager := NewManager(deps)

			td := &v1alpha1.TidbDashboard{}
			if testcase.setInputs != nil {
				testcase.setInputs(td)
			}

			if testcase.getServices != nil {
				old, newly := testcase.getServices()
				patch := gomonkey.ApplyFunc(generateTiDBDashboardService, func(_ *v1alpha1.TidbDashboard) *corev1.Service {
					return newly
				})
				defer patch.Reset()
				if old != nil {
					indexer.Add(old)
				}
			}

			deps.ServiceControl.(*controller.FakeServiceControl).SetCreateServiceError(testcase.createServiceErr, 0)

			deps.ServiceControl.(*controller.FakeServiceControl).SetUpdateServiceError(testcase.updateServiceErr, 0)

			err := manager.syncService(td)
			testcase.expectErr(err)
		}

		for _, testcase := range cases {
			runCase(testcase)
		}
	})

	t.Run("syncCore", func(t *testing.T) {
		g := NewGomegaWithT(t)

		type testcase struct {
			name string

			setInputs            func(td *v1alpha1.TidbDashboard, tc *v1alpha1.TidbCluster)
			getSts               func() (*apps.StatefulSet, *apps.StatefulSet)
			createStatefulSetErr error
			updateStatefulSetErr error
			expectFn             func(td *v1alpha1.TidbDashboard, err error)
		}

		cases := []testcase{
			{
				name: "should create sts if old sts isn't found",
				getSts: func() (*apps.StatefulSet, *apps.StatefulSet) {
					sts := &apps.StatefulSet{}
					sts.Name = StatefulSetName("td")
					sts.Namespace = "default"
					return nil, sts.DeepCopy()
				},
				createStatefulSetErr: fmt.Errorf("should create sts"),
				updateStatefulSetErr: fmt.Errorf("shouldn't update sts"),
				expectFn: func(td *v1alpha1.TidbDashboard, err error) {
					g.Expect(err).Should(HaveOccurred())
					g.Expect(err.Error()).Should(ContainSubstring("should create sts"))
				},
			},
			{
				name: "should update sts if old sts is exist",
				getSts: func() (*apps.StatefulSet, *apps.StatefulSet) {
					sts := &apps.StatefulSet{}
					sts.Name = StatefulSetName("td")
					sts.Namespace = "default"
					return sts.DeepCopy(), sts.DeepCopy()
				},
				createStatefulSetErr: fmt.Errorf("shouldn't create sts"),
				updateStatefulSetErr: fmt.Errorf("should update sts"),
				expectFn: func(td *v1alpha1.TidbDashboard, err error) {
					g.Expect(err).Should(HaveOccurred())
					g.Expect(err.Error()).Should(ContainSubstring("should update sts"))
				},
			},
		}

		runCase := func(testcase testcase) {
			t.Logf("testcase: %s", testcase.name)

			deps := controller.NewFakeDependencies()
			indexer := deps.KubeInformerFactory.Apps().V1().StatefulSets().Informer().GetIndexer()
			manager := NewManager(deps)

			td := &v1alpha1.TidbDashboard{}
			tc := &v1alpha1.TidbCluster{}
			td.Name = "td"
			td.Namespace = "default"
			tc.Name = "tc"
			tc.Namespace = "default"
			if testcase.setInputs != nil {
				testcase.setInputs(td, tc)
			}

			if testcase.getSts != nil {
				old, newly := testcase.getSts()
				patch := gomonkey.ApplyFunc(generateTiDBDashboardStatefulSet, func(_ *v1alpha1.TidbDashboard, _ *v1alpha1.TidbCluster) (*apps.StatefulSet, error) {
					return newly, nil
				})
				defer patch.Reset()
				if old != nil {
					indexer.Add(old)
				}
			}

			deps.StatefulSetControl.(*controller.FakeStatefulSetControl).SetCreateStatefulSetError(testcase.createStatefulSetErr, 0)

			updateStsPatch := gomonkey.ApplyFunc(mngerutils.UpdateStatefulSet, func(_ controller.StatefulSetControlInterface, _ runtime.Object, _, _ *apps.StatefulSet) error {
				return testcase.updateStatefulSetErr
			})
			defer updateStsPatch.Reset()

			err := manager.syncCore(td, tc)
			testcase.expectFn(td, err)
		}

		for _, testcase := range cases {
			runCase(testcase)
		}
	})

	t.Run("confirmStatefulSetIsUpgrading", func(t *testing.T) {
		g := NewGomegaWithT(t)

		type testcase struct {
			name     string
			setSts   func(*apps.StatefulSet)
			hasPod   bool
			setPod   func(*corev1.Pod)
			expectFn func(bool, error)
		}

		cases := []testcase{
			{
				name: "sts is upgrading",
				setSts: func(sts *apps.StatefulSet) {
					sts.Status.CurrentRevision = "v1"
					sts.Status.UpdateRevision = "v2"
					sts.Status.ObservedGeneration = 1000
				},
				hasPod: false,
				setPod: nil,
				expectFn: func(upgrading bool, err error) {
					g.Expect(upgrading).Should(BeTrue())
					g.Expect(err).Should(Succeed())
				},
			},
			{
				name:   "pod don't have revision hash",
				setSts: nil,
				hasPod: false,
				setPod: nil,
				expectFn: func(upgrading bool, err error) {
					g.Expect(upgrading).Should(BeFalse())
					g.Expect(err).Should(Succeed())
				},
			},
			{
				name:   "pod have revision hash, not equal statefulset's",
				setSts: nil,
				hasPod: true,
				setPod: func(pod *corev1.Pod) {
					pod.Labels[apps.ControllerRevisionHashLabelKey] = "v2"
				},
				expectFn: func(upgrading bool, err error) {
					g.Expect(upgrading).Should(BeTrue())
					g.Expect(err).Should(Succeed())
				},
			},
			{
				name:   "pod have revision hash, equal statefulset's",
				setSts: nil,
				hasPod: true,
				setPod: func(pod *corev1.Pod) {
					pod.Labels[apps.ControllerRevisionHashLabelKey] = "v3"
				},
				expectFn: func(upgrading bool, err error) {
					g.Expect(upgrading).Should(BeFalse())
					g.Expect(err).Should(Succeed())
				},
			},
		}

		for _, testcase := range cases {
			t.Logf("testcase: %s", testcase.name)

			deps := controller.NewFakeDependencies()
			podIndexer := deps.KubeInformerFactory.Core().V1().Pods().Informer().GetIndexer()
			manager := NewManager(deps)

			td := &v1alpha1.TidbDashboard{}
			td.Name = "td"
			td.Namespace = "default"

			sts := &apps.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: metav1.NamespaceDefault,
				},
				Status: apps.StatefulSetStatus{
					UpdateRevision:  "v3",
					CurrentRevision: "v3",
				},
			}
			if testcase.setSts != nil {
				testcase.setSts(sts)
			}
			if testcase.hasPod {
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:        StatefulSetName(td.Name) + "-0",
						Namespace:   td.Namespace,
						Annotations: map[string]string{},
						Labels:      label.NewTiDBDashboard().Instance(td.Name).TiDBDashboard().Labels(),
					},
				}
				if testcase.setPod != nil {
					testcase.setPod(pod)
				}
				podIndexer.Add(pod)
			}
			upgrading, err := manager.confirmStatefulSetIsUpgrading(td, sts)
			testcase.expectFn(upgrading, err)
		}
	})
}

func TestGenerateTiDBDashboardService(t *testing.T) {

	type testcase struct {
		name      string
		setInputs func(td *v1alpha1.TidbDashboard)
		expectFn  func(td *v1alpha1.TidbDashboard, svc *corev1.Service)
	}

	cases := []testcase{
		{
			name:      "basic",
			setInputs: nil,
			expectFn: func(td *v1alpha1.TidbDashboard, svc *corev1.Service) {
				expectSvc := corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ServiceName(td.Name),
						Namespace: td.Namespace,
						Labels: map[string]string{
							"app.kubernetes.io/name":       "tidb-dashboard",
							"app.kubernetes.io/managed-by": "tidb-operator",
							"app.kubernetes.io/instance":   td.Name,
							"app.kubernetes.io/component":  "tidb-dashboard",
						},
						Annotations: make(map[string]string),
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "pingcap.com/v1alpha1",
								Kind:       v1alpha1.TiDBDashboardKind,
								Name:       td.Name,
								UID:        "",
								Controller: func(b bool) *bool {
									return &b
								}(true),
								BlockOwnerDeletion: func(b bool) *bool {
									return &b
								}(true),
							},
						},
					},
					Spec: corev1.ServiceSpec{
						Selector: map[string]string{
							"app.kubernetes.io/component":  "tidb-dashboard",
							"app.kubernetes.io/instance":   td.Name,
							"app.kubernetes.io/managed-by": "tidb-operator",
							"app.kubernetes.io/name":       "tidb-dashboard",
						},
						Ports: []corev1.ServicePort{
							{
								Name:       "tidb-dashboard",
								Port:       port,
								TargetPort: intstr.FromInt(port),
								Protocol:   corev1.ProtocolTCP,
							},
						},
						PublishNotReadyAddresses: true,
					},
				}
				if diff := cmp.Diff(*svc, expectSvc); diff != "" {
					t.Errorf("unexpected Service (-want, +got): %s", diff)
				}
			},
		},
	}

	for _, testcase := range cases {
		t.Logf("testcase: %s", testcase.name)

		td := &v1alpha1.TidbDashboard{}
		td.Name = "td"
		td.Namespace = "default"
		if testcase.setInputs != nil {
			testcase.setInputs(td)
		}

		svc := generateTiDBDashboardService(td)
		testcase.expectFn(td, svc)
	}
}

func TestGenerateDashboardStatefulSet(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name string

		setInputs func(td *v1alpha1.TidbDashboard, tc *v1alpha1.TidbCluster)
		expectFn  func(sts *apps.StatefulSet, err error)
	}

	cases := []testcase{
		{
			name: "don't run in host network",
			expectFn: func(sts *apps.StatefulSet, err error) {
				g.Expect(sts.Spec.Template.Spec.HostNetwork).Should(BeFalse())
				g.Expect(sts.Spec.Template.Spec.DNSPolicy).Should(Equal(corev1.DNSClusterFirst))
			},
		},
		{
			name: "run in host network",
			setInputs: func(td *v1alpha1.TidbDashboard, tc *v1alpha1.TidbCluster) {
				hostnetwork := true
				td.Spec.HostNetwork = &hostnetwork
			},
			expectFn: func(sts *apps.StatefulSet, err error) {
				g.Expect(sts.Spec.Template.Spec.HostNetwork).Should(BeTrue())
				g.Expect(sts.Spec.Template.Spec.DNSPolicy).Should(Equal(corev1.DNSClusterFirstWithHostNet))
			},
		},
		{
			name: "should set resource",
			setInputs: func(td *v1alpha1.TidbDashboard, tc *v1alpha1.TidbCluster) {
				td.Spec.ResourceRequirements = corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:              resource.MustParse("1"),
						corev1.ResourceMemory:           resource.MustParse("2Gi"),
						corev1.ResourceEphemeralStorage: resource.MustParse("10Gi"),
						corev1.ResourceStorage:          resource.MustParse("100Gi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:              resource.MustParse("1"),
						corev1.ResourceMemory:           resource.MustParse("2Gi"),
						corev1.ResourceEphemeralStorage: resource.MustParse("10Gi"),
					},
				}
			},
			expectFn: func(sts *apps.StatefulSet, err error) {
				g.Expect(sts.Spec.VolumeClaimTemplates[0].Spec.Resources).To(Equal(corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("100Gi"),
					},
				}))
				container := getDashboardContainer(sts)
				g.Expect(container.Resources).To(Equal(corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:              resource.MustParse("1"),
						corev1.ResourceMemory:           resource.MustParse("2Gi"),
						corev1.ResourceEphemeralStorage: resource.MustParse("10Gi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:              resource.MustParse("1"),
						corev1.ResourceMemory:           resource.MustParse("2Gi"),
						corev1.ResourceEphemeralStorage: resource.MustParse("10Gi"),
					},
				}))
			},
		},
		{
			name: "should set custom env",
			setInputs: func(td *v1alpha1.TidbDashboard, tc *v1alpha1.TidbCluster) {
				td.Spec.Env = []corev1.EnvVar{
					{
						Name:  "SOURCE1",
						Value: "mysql_replica1",
					},
					{
						Name:  "TZ",
						Value: "ignored",
					},
				}
			},
			expectFn: func(sts *apps.StatefulSet, err error) {
				expectEnvs := []corev1.EnvVar{
					{
						Name:  "SOURCE1",
						Value: "mysql_replica1",
					},
					{
						Name:  "TZ",
						Value: "ignored",
					},
				}
				container := getDashboardContainer(sts)
				g.Expect(container.Env).Should(ContainElements(expectEnvs))
			},
		},
		{
			name: "should add additional containers",
			setInputs: func(td *v1alpha1.TidbDashboard, tc *v1alpha1.TidbCluster) {
				td.Spec.AdditionalContainers = []corev1.Container{
					{
						Name: "custom1",
					},
					{
						Name: "custom2",
					},
				}
			},
			expectFn: func(sts *apps.StatefulSet, err error) {
				expectContainers := []corev1.Container{
					{
						Name: "custom1",
					},
					{
						Name: "custom2",
					},
				}
				g.Expect(sts.Spec.Template.Spec.Containers).Should(ContainElements(expectContainers))
			},
		},
		{
			name: "should add additional storage volumes",
			setInputs: func(td *v1alpha1.TidbDashboard, tc *v1alpha1.TidbCluster) {
				td.Spec.StorageVolumes = []v1alpha1.StorageVolume{
					{
						Name:        "test",
						StorageSize: "100Gi",
						MountPath:   "/path",
					},
				}
			},
			expectFn: func(sts *apps.StatefulSet, err error) {
				quantity, _ := resource.ParseQuantity("100Gi")
				expectVolumeMounts := []corev1.VolumeMount{
					{
						Name:      fmt.Sprintf("%s-test", v1alpha1.TiDBDashboardMemberType),
						MountPath: "/path",
					},
				}
				expectPVCs := []corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-test", v1alpha1.TiDBDashboardMemberType)},
						Spec: corev1.PersistentVolumeClaimSpec{
							AccessModes: []corev1.PersistentVolumeAccessMode{
								corev1.ReadWriteOnce,
							},
							StorageClassName: nil,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: quantity,
								},
							},
						},
					},
				}
				container := getDashboardContainer(sts)
				g.Expect(container.VolumeMounts).Should(ContainElements(expectVolumeMounts))
				g.Expect(sts.Spec.VolumeClaimTemplates).Should(ContainElements(expectPVCs))
			},
		},
		{
			name: "should add custom labels",
			setInputs: func(td *v1alpha1.TidbDashboard, tc *v1alpha1.TidbCluster) {
				td.Spec.Labels = map[string]string{
					"test1": "test1",
					"test2": "test2",
				}
			},
			expectFn: func(sts *apps.StatefulSet, err error) {
				expectLabels := map[string]string{
					"test1": "test1",
					"test2": "test2",
				}
				for k, v := range expectLabels {
					g.Expect(sts.Spec.Template.Labels).Should(HaveKeyWithValue(k, v))
				}
			},
		},
		{
			name: "should add custom annotation",
			setInputs: func(td *v1alpha1.TidbDashboard, tc *v1alpha1.TidbCluster) {
				td.Spec.Annotations = map[string]string{
					"test1": "test1",
					"test2": "test2",
				}
			},
			expectFn: func(sts *apps.StatefulSet, err error) {
				expectAnno := map[string]string{
					"test1": "test1",
					"test2": "test2",
				}
				for k, v := range expectAnno {
					g.Expect(sts.Spec.Template.Annotations).Should(HaveKeyWithValue(k, v))
				}
			},
		},
		{
			name: "should add additional volumes and mounts",
			setInputs: func(td *v1alpha1.TidbDashboard, tc *v1alpha1.TidbCluster) {
				td.Spec.AdditionalVolumeMounts = []corev1.VolumeMount{{Name: "test"}}
				td.Spec.AdditionalVolumes = []corev1.Volume{{Name: "test", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}}
			},
			expectFn: func(sts *apps.StatefulSet, err error) {
				expectVolumeMounts := []corev1.VolumeMount{{Name: "test"}}
				expectVolumes := []corev1.Volume{{Name: "test", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}}
				container := getDashboardContainer(sts)
				g.Expect(container.VolumeMounts).Should(ContainElements(expectVolumeMounts))
				g.Expect(sts.Spec.Template.Spec.Volumes).Should(ContainElements(expectVolumes))
			},
		},
		{
			name: "should add secret volume when tls is enable",
			setInputs: func(td *v1alpha1.TidbDashboard, tc *v1alpha1.TidbCluster) {
				tc.Spec.TLSCluster = &v1alpha1.TLSCluster{
					Enabled: true,
				}
			},
			expectFn: func(sts *apps.StatefulSet, err error) {
				expectVolumeMounts := []corev1.VolumeMount{{Name: clusterTLSVolumeName, ReadOnly: true, MountPath: clusterTLSMountPath}}
				expectVolumes := []corev1.Volume{{Name: clusterTLSVolumeName, VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: TCClusterClientTLSSecretName("td")}}}}
				container := getDashboardContainer(sts)
				g.Expect(container.VolumeMounts).Should(ContainElements(expectVolumeMounts))
				g.Expect(sts.Spec.Template.Spec.Volumes).Should(ContainElements(expectVolumes))
			},
		},
	}

	for _, testcase := range cases {
		t.Logf("testcase: %s", testcase.name)

		tc := &v1alpha1.TidbCluster{}
		tc.Name = "tc"
		tc.Namespace = "default"
		td := &v1alpha1.TidbDashboard{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "td",
				Namespace: "default",
			},
			Spec: v1alpha1.TidbDashboardSpec{
				Clusters: []v1alpha1.TidbClusterRef{
					{
						Name:      "tc",
						Namespace: "default",
					},
				},
			},
		}

		if testcase.setInputs != nil {
			testcase.setInputs(td, tc)
		}

		sts, err := generateTiDBDashboardStatefulSet(td, tc)
		testcase.expectFn(sts, err)
	}
}

func getDashboardContainer(sts *apps.StatefulSet) *corev1.Container {
	for i := range sts.Spec.Template.Spec.Containers {
		container := sts.Spec.Template.Spec.Containers[i]
		if container.Name == v1alpha1.TiDBDashboardMemberType.String() {
			return &container
		}
	}
	return nil
}
