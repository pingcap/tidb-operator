// Copyright 2021 PingCAP, Inc.
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

package tidbngmonitoring

import (
	"fmt"
	"path"
	"testing"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apis/util/config"
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

func TestNGMonitorManager(t *testing.T) {

	t.Run("syncService", func(t *testing.T) {
		g := NewGomegaWithT(t)

		type testcase struct {
			name string

			setInputs func(tngm *v1alpha1.TidbNGMonitoring)

			getServices      func() (*corev1.Service /*old*/, *corev1.Service /*new*/) // old service is nil means that service isn't found
			createServiceErr error
			updateServiceErr error
			expectErr        func(err error)
		}

		cases := []testcase{
			{
				name: "manager is paused",
				setInputs: func(tngm *v1alpha1.TidbNGMonitoring) {
					tngm.Spec.Paused = true
				},
				getServices: func() (*corev1.Service, *corev1.Service) {
					return nil, nil // new service is nil because it shouldn't be arrived
				},
				expectErr: func(err error) {
					g.Expect(err).Should(Succeed())
				},
			},
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
				name: "shoudn't update equal service ",
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
					new := old.DeepCopy()
					controller.SetServiceLastAppliedConfigAnnotation(old)

					return old, new
				},
				createServiceErr: fmt.Errorf("shouldn't create service"),
				updateServiceErr: fmt.Errorf("shouldn't update service"),
				expectErr: func(err error) {
					g.Expect(err).Should(Succeed())
				},
			},
			{
				name: "should update different service ",
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
					new := old.DeepCopy()
					controller.SetServiceLastAppliedConfigAnnotation(old)
					new.Spec.ClusterIP = "10.0.0.1"

					return old, new
				},
				createServiceErr: fmt.Errorf("shouldn't create service"),
				updateServiceErr: fmt.Errorf("should update service"),
				expectErr: func(err error) {
					g.Expect(err).Should(HaveOccurred())
					g.Expect(err.Error()).Should(ContainSubstring("should update service"))
				},
			},
		}

		for _, testcase := range cases {
			t.Logf("testcase: %s", testcase.name)

			deps := controller.NewFakeDependencies()
			indexer := deps.KubeInformerFactory.Core().V1().Services().Informer().GetIndexer()

			manager := NewNGMonitorManager(deps)

			tngm := &v1alpha1.TidbNGMonitoring{}
			if testcase.setInputs != nil {
				testcase.setInputs(tngm)
			}

			// mock new and old service
			if testcase.getServices != nil {
				old, new := testcase.getServices()
				patch := gomonkey.ApplyFunc(GenerateNGMonitoringHeadlessService, func(_ *v1alpha1.TidbNGMonitoring) *corev1.Service {
					return new
				})
				defer patch.Reset()
				if old != nil {
					indexer.Add(old)
				}
			}
			// mock result of service creation
			deps.ServiceControl.(*controller.FakeServiceControl).SetCreateServiceError(testcase.createServiceErr, 0)
			// mock result of service update
			deps.ServiceControl.(*controller.FakeServiceControl).SetUpdateServiceError(testcase.updateServiceErr, 0)

			err := manager.syncService(tngm)
			testcase.expectErr(err)
		}
	})

	t.Run("syncCore", func(t *testing.T) {
		g := NewGomegaWithT(t)

		type testcase struct {
			name string

			setInputs            func(tngm *v1alpha1.TidbNGMonitoring, tc *v1alpha1.TidbCluster)
			getSts               func() (*apps.StatefulSet, *apps.StatefulSet) // old sts is nil means that sts isn't found
			createStatefulSetErr error
			updateStatefulSetErr error
			expectFn             func(tngm *v1alpha1.TidbNGMonitoring, err error)
		}

		cases := []testcase{
			{
				name: "manager is paused",
				setInputs: func(tngm *v1alpha1.TidbNGMonitoring, tc *v1alpha1.TidbCluster) {
					tngm.Spec.Paused = true
				},
				getSts: func() (*apps.StatefulSet, *apps.StatefulSet) {
					return nil, nil
				},
				createStatefulSetErr: fmt.Errorf("shouldn't create sts"),
				updateStatefulSetErr: fmt.Errorf("shouldn't update sts"),
				expectFn: func(tngm *v1alpha1.TidbNGMonitoring, err error) {
					g.Expect(err).Should(Succeed())
				},
			},
			{
				name: "should create sts if old sts isn't found",
				getSts: func() (*apps.StatefulSet, *apps.StatefulSet) {
					sts := &apps.StatefulSet{}
					sts.Name = NGMonitoringName("ngm")
					sts.Namespace = "default"
					return nil, sts.DeepCopy()
				},
				createStatefulSetErr: fmt.Errorf("should create sts"),
				updateStatefulSetErr: fmt.Errorf("shouldn't update sts"),
				expectFn: func(tngm *v1alpha1.TidbNGMonitoring, err error) {
					g.Expect(err).Should(HaveOccurred())
					g.Expect(err.Error()).Should(ContainSubstring("should create sts"))
				},
			},
			{
				name: "should update sts if old sts is exist",
				getSts: func() (*apps.StatefulSet, *apps.StatefulSet) {
					sts := &apps.StatefulSet{}
					sts.Name = NGMonitoringName("ngm")
					sts.Namespace = "default"
					return sts.DeepCopy(), sts.DeepCopy()
				},
				createStatefulSetErr: fmt.Errorf("shouldn't create sts"),
				updateStatefulSetErr: fmt.Errorf("should update sts"),
				expectFn: func(tngm *v1alpha1.TidbNGMonitoring, err error) {
					g.Expect(err).Should(HaveOccurred())
					g.Expect(err.Error()).Should(ContainSubstring("should update sts"))
				},
			},
		}

		for _, testcase := range cases {
			t.Logf("testcase: %s", testcase.name)

			deps := controller.NewFakeDependencies()
			indexer := deps.KubeInformerFactory.Apps().V1().StatefulSets().Informer().GetIndexer()
			manager := NewNGMonitorManager(deps)

			tngm := &v1alpha1.TidbNGMonitoring{}
			tc := &v1alpha1.TidbCluster{}
			tngm.Name = "ngm"
			tngm.Namespace = "default"
			tc.Name = "tc"
			tc.Namespace = "default"
			if testcase.setInputs != nil {
				testcase.setInputs(tngm, tc)
			}

			// mock old and new sts
			if testcase.getSts != nil {
				old, new := testcase.getSts()
				patch := gomonkey.ApplyFunc(GenerateNGMonitoringStatefulSet, func(_ *v1alpha1.TidbNGMonitoring, _ *v1alpha1.TidbCluster, _ *corev1.ConfigMap) (*apps.StatefulSet, error) {
					return new, nil
				})
				defer patch.Reset()
				if old != nil {
					indexer.Add(old)
				}
			}
			// mock result of sts creation
			deps.StatefulSetControl.(*controller.FakeStatefulSetControl).SetCreateStatefulSetError(testcase.createStatefulSetErr, 0)
			// mock result of sts update
			updateStsPatch := gomonkey.ApplyFunc(mngerutils.UpdateStatefulSet, func(_ controller.StatefulSetControlInterface, _ runtime.Object, _, _ *apps.StatefulSet) error {
				return testcase.updateStatefulSetErr
			})
			defer updateStsPatch.Reset()

			err := manager.syncCore(tngm, tc)
			testcase.expectFn(tngm, err)
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
			manager := NewNGMonitorManager(deps)

			tngm := &v1alpha1.TidbNGMonitoring{}
			tngm.Name = "ngm"
			tngm.Namespace = "default"

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
						Name:        NGMonitoringName(tngm.Name) + "-0",
						Namespace:   tngm.Namespace,
						Annotations: map[string]string{},
						Labels:      label.NewTiDBNGMonitoring().Instance(tngm.GetInstanceName()).NGMonitoring().Labels(),
					},
				}
				if testcase.setPod != nil {
					testcase.setPod(pod)
				}
				podIndexer.Add(pod)
			}
			upgrading, err := manager.confirmStatefulSetIsUpgrading(tngm, sts)
			testcase.expectFn(upgrading, err)
		}
	})
}

func TestGenerateNGMonitoringHeadlessService(t *testing.T) {

	type testcase struct {
		name      string
		setInputs func(tngm *v1alpha1.TidbNGMonitoring)
		expectFn  func(tngm *v1alpha1.TidbNGMonitoring, svc *corev1.Service)
	}

	cases := []testcase{
		{
			name:      "basic",
			setInputs: nil,
			expectFn: func(tngm *v1alpha1.TidbNGMonitoring, svc *corev1.Service) {
				expectSvc := corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      NGMonitoringHeadlessServiceName(tngm.Name),
						Namespace: tngm.Namespace,
						Labels: map[string]string{
							"app.kubernetes.io/name":       "tidb-ng-monitoring",
							"app.kubernetes.io/managed-by": "tidb-operator",
							"app.kubernetes.io/instance":   tngm.Name,
							"app.kubernetes.io/component":  "ng-monitoring",
						},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "pingcap.com/v1alpha1",
								Kind:       v1alpha1.TiDBNGMonitoringKind,
								Name:       tngm.Name,
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
							"app.kubernetes.io/component":  "ng-monitoring",
							"app.kubernetes.io/instance":   tngm.Name,
							"app.kubernetes.io/managed-by": "tidb-operator",
							"app.kubernetes.io/name":       "tidb-ng-monitoring",
						},
						Ports: []corev1.ServicePort{
							{
								Name:       "ng-monitoring",
								Port:       ngmServicePort,
								TargetPort: intstr.FromInt(ngmServicePort),
								Protocol:   corev1.ProtocolTCP,
							},
						},
						ClusterIP:                "None",
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

		tngm := &v1alpha1.TidbNGMonitoring{}
		tngm.Name = "ngm"
		tngm.Namespace = "default"
		if testcase.setInputs != nil {
			testcase.setInputs(tngm)
		}

		svc := GenerateNGMonitoringHeadlessService(tngm)
		testcase.expectFn(tngm, svc)
	}
}

func TestGenerateNGMonitoringStatefulSet(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name string

		setInputs    func(tngm *v1alpha1.TidbNGMonitoring, tc *v1alpha1.TidbCluster)
		nilConfigMap bool
		expectFn     func(sts *apps.StatefulSet, err error)
	}

	cases := []testcase{
		{
			name:         "configmap is nil",
			setInputs:    nil,
			nilConfigMap: true,
			expectFn: func(sts *apps.StatefulSet, err error) {
				g.Expect(err).Should(HaveOccurred())
			},
		},
		{
			name: "don't run in host network",
			expectFn: func(sts *apps.StatefulSet, err error) {
				g.Expect(sts.Spec.Template.Spec.HostNetwork).Should(BeFalse())
				g.Expect(sts.Spec.Template.Spec.DNSPolicy).Should(Equal(corev1.DNSClusterFirst))
			},
		},
		{
			name: "run in host network",
			setInputs: func(tngm *v1alpha1.TidbNGMonitoring, tc *v1alpha1.TidbCluster) {
				hostnetwork := true
				tngm.Spec.HostNetwork = &hostnetwork
			},
			expectFn: func(sts *apps.StatefulSet, err error) {
				g.Expect(sts.Spec.Template.Spec.HostNetwork).Should(BeTrue())
				g.Expect(sts.Spec.Template.Spec.DNSPolicy).Should(Equal(corev1.DNSClusterFirstWithHostNet))
			},
		},
		{
			name: "should set resouce",
			setInputs: func(tngm *v1alpha1.TidbNGMonitoring, tc *v1alpha1.TidbCluster) {
				tngm.Spec.NGMonitoring.ResourceRequirements = corev1.ResourceRequirements{
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
				container := getNGMonitoringContainer(sts)
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
			setInputs: func(tngm *v1alpha1.TidbNGMonitoring, tc *v1alpha1.TidbCluster) {
				tngm.Spec.NGMonitoring.Env = []corev1.EnvVar{
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
				container := getNGMonitoringContainer(sts)
				g.Expect(container.Env).Should(ContainElements(expectEnvs))
			},
		},
		{
			name: "should add additional containers",
			setInputs: func(tngm *v1alpha1.TidbNGMonitoring, tc *v1alpha1.TidbCluster) {
				tngm.Spec.NGMonitoring.AdditionalContainers = []corev1.Container{
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
			setInputs: func(tngm *v1alpha1.TidbNGMonitoring, tc *v1alpha1.TidbCluster) {
				tngm.Spec.NGMonitoring.StorageVolumes = []v1alpha1.StorageVolume{
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
						Name:      fmt.Sprintf("%s-test", v1alpha1.NGMonitoringMemberType),
						MountPath: "/path",
					},
				}
				expectPVCs := []corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-test", v1alpha1.NGMonitoringMemberType)},
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
				container := getNGMonitoringContainer(sts)
				g.Expect(container.VolumeMounts).Should(ContainElements(expectVolumeMounts))
				g.Expect(sts.Spec.VolumeClaimTemplates).Should(ContainElements(expectPVCs))
			},
		},
		{
			name: "should add custom labels",
			setInputs: func(tngm *v1alpha1.TidbNGMonitoring, tc *v1alpha1.TidbCluster) {
				tngm.Spec.NGMonitoring.Labels = map[string]string{
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
			setInputs: func(tngm *v1alpha1.TidbNGMonitoring, tc *v1alpha1.TidbCluster) {
				tngm.Spec.NGMonitoring.Annotations = map[string]string{
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
			setInputs: func(tngm *v1alpha1.TidbNGMonitoring, tc *v1alpha1.TidbCluster) {
				tngm.Spec.NGMonitoring.AdditionalVolumeMounts = []corev1.VolumeMount{{Name: "test"}}
				tngm.Spec.NGMonitoring.AdditionalVolumes = []corev1.Volume{{Name: "test", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}}
			},
			expectFn: func(sts *apps.StatefulSet, err error) {
				expectVolumeMounts := []corev1.VolumeMount{{Name: "test"}}
				expectVolumes := []corev1.Volume{{Name: "test", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}}
				container := getNGMonitoringContainer(sts)
				g.Expect(container.VolumeMounts).Should(ContainElements(expectVolumeMounts))
				g.Expect(sts.Spec.Template.Spec.Volumes).Should(ContainElements(expectVolumes))
			},
		},
		{
			name: "should add secret volume when tls is enable",
			setInputs: func(tngm *v1alpha1.TidbNGMonitoring, tc *v1alpha1.TidbCluster) {
				tc.Spec.TLSCluster = &v1alpha1.TLSCluster{
					Enabled: true,
				}
			},
			expectFn: func(sts *apps.StatefulSet, err error) {
				expectVolumeMounts := []corev1.VolumeMount{{Name: "tc-client-tls", ReadOnly: true, MountPath: ngmTCClientTLSMountDir}}
				expectVolumes := []corev1.Volume{{Name: "tc-client-tls", VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: TCClientTLSSecretName("ngm")}}}}
				container := getNGMonitoringContainer(sts)
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
		tngm := &v1alpha1.TidbNGMonitoring{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ngm",
				Namespace: "default",
			},
			Spec: v1alpha1.TidbNGMonitoringSpec{
				Clusters: []v1alpha1.TidbClusterRef{
					{
						Name:      "tc",
						Namespace: "default",
					},
				},
			},
		}

		if testcase.setInputs != nil {
			testcase.setInputs(tngm, tc)
		}

		var cm *corev1.ConfigMap
		if !testcase.nilConfigMap {
			cm = &corev1.ConfigMap{}
		}

		sts, err := GenerateNGMonitoringStatefulSet(tngm, tc, cm)
		testcase.expectFn(sts, err)
	}
}

func TestGenerateNGMonitoringConfigMap(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcases struct {
		name      string
		setInputs func(tngm *v1alpha1.TidbNGMonitoring, tc *v1alpha1.TidbCluster)
		expectFn  func(tngm *v1alpha1.TidbNGMonitoring, cm *corev1.ConfigMap, err error)
	}

	cases := []testcases{
		{
			name: "custom config is nil",
			expectFn: func(tngm *v1alpha1.TidbNGMonitoring, cm *corev1.ConfigMap, err error) {
				g.Expect(err).Should(Succeed())

				expectCM := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      NGMonitoringName(tngm.Name),
						Namespace: "default",
						Labels: map[string]string{
							"app.kubernetes.io/name":       "tidb-ng-monitoring",
							"app.kubernetes.io/managed-by": "tidb-operator",
							"app.kubernetes.io/instance":   tngm.Name,
							"app.kubernetes.io/component":  "ng-monitoring",
						},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "pingcap.com/v1alpha1",
								Kind:       v1alpha1.TiDBNGMonitoringKind,
								Name:       tngm.Name,
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
					Data: map[string]string{
						ngmConfigMapConfigKey: "",
					},
				}

				g.Expect(cm).Should(Equal(expectCM))
			},
		},
		{
			name: "should add custom config",
			setInputs: func(tngm *v1alpha1.TidbNGMonitoring, tc *v1alpha1.TidbCluster) {
				tngm.Spec.NGMonitoring.Config = config.New(map[string]interface{}{
					"test": "test",
					"table": map[string]interface{}{
						"test": "test",
					},
				})
			},
			expectFn: func(tngm *v1alpha1.TidbNGMonitoring, cm *corev1.ConfigMap, err error) {
				g.Expect(err).Should(Succeed())

				cfg := config.New(nil)
				err = cfg.UnmarshalTOML([]byte(cm.Data[ngmConfigMapConfigKey]))
				g.Expect(err).Should(Succeed())

				expectConfig := map[string]interface{}{
					"test": "test",
					"table": map[string]interface{}{
						"test": "test",
					},
				}
				for k, v := range expectConfig {
					g.Expect(cfg.MP).Should(HaveKeyWithValue(k, v))
				}
			},
		},
		{
			name: "should set config about cert when tls is enable",
			setInputs: func(tngm *v1alpha1.TidbNGMonitoring, tc *v1alpha1.TidbCluster) {
				tc.Spec.TLSCluster = &v1alpha1.TLSCluster{
					Enabled: true,
				}
			},
			expectFn: func(tngm *v1alpha1.TidbNGMonitoring, cm *corev1.ConfigMap, err error) {
				g.Expect(err).Should(Succeed())

				cfg := config.New(nil)
				err = cfg.UnmarshalTOML([]byte(cm.Data[ngmConfigMapConfigKey]))
				g.Expect(err).Should(Succeed())

				expectConfig := map[string]interface{}{
					"security": map[string]interface{}{
						"ca-path":   path.Join(ngmTCClientTLSMountDir, assetKey("tc", "default", corev1.ServiceAccountRootCAKey)),
						"cert-path": path.Join(ngmTCClientTLSMountDir, assetKey("tc", "default", corev1.TLSCertKey)),
						"key-path":  path.Join(ngmTCClientTLSMountDir, assetKey("tc", "default", corev1.TLSPrivateKeyKey)),
					},
				}
				for k, v := range expectConfig {
					g.Expect(cfg.MP).Should(HaveKeyWithValue(k, v))
				}
			},
		},
		{
			name: "shouldn't change config in spec",
			setInputs: func(tngm *v1alpha1.TidbNGMonitoring, tc *v1alpha1.TidbCluster) {
				tngm.Spec.NGMonitoring.Config = config.New(map[string]interface{}{
					"test": "test",
					"table": map[string]interface{}{
						"test": "test",
					},
				})
				tc.Spec.TLSCluster = &v1alpha1.TLSCluster{
					Enabled: true,
				}
			},
			expectFn: func(tngm *v1alpha1.TidbNGMonitoring, cm *corev1.ConfigMap, err error) {
				g.Expect(err).Should(Succeed())

				cfg := config.New(nil)
				err = cfg.UnmarshalTOML([]byte(cm.Data[ngmConfigMapConfigKey]))
				g.Expect(err).Should(Succeed())

				expectConfig := config.New(map[string]interface{}{
					"test": "test",
					"table": map[string]interface{}{
						"test": "test",
					},
				})
				g.Expect(tngm.Spec.NGMonitoring.Config).Should(Equal(expectConfig))
			},
		},
	}

	for _, testcase := range cases {
		t.Logf("testcase: %s", testcase.name)

		tc := &v1alpha1.TidbCluster{}
		tc.Name = "tc"
		tc.Namespace = "default"
		tngm := &v1alpha1.TidbNGMonitoring{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ngm",
				Namespace: "default",
			},
			Spec: v1alpha1.TidbNGMonitoringSpec{
				Clusters: []v1alpha1.TidbClusterRef{
					{
						Name:      "tc",
						Namespace: "default",
					},
				},
			},
		}
		if testcase.setInputs != nil {
			testcase.setInputs(tngm, tc)
		}

		cm, err := GenerateNGMonitoringConfigMap(tngm, tc)
		testcase.expectFn(tngm, cm, err)
	}
}

func getNGMonitoringContainer(sts *apps.StatefulSet) *corev1.Container {
	for i := range sts.Spec.Template.Spec.Containers {
		container := sts.Spec.Template.Spec.Containers[i]
		if container.Name == v1alpha1.NGMonitoringMemberType.String() {
			return &container
		}
	}
	return nil
}
