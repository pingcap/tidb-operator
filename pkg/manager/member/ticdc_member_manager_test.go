// Copyright 2020 PingCAP, Inc.
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
	"fmt"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager/suspender"
	"github.com/pingcap/tidb-operator/pkg/manager/volumes"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/utils/pointer"
)

func TestTiCDCMemberManagerSyncCreate(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name             string
		errSync          bool
		err              bool
		tls              bool
		suspendComponent func() (bool, error)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		tc := newTidbClusterForCDC()
		if test.tls {
			tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
		}

		oldSpec := tc.Spec

		tmm, fakeSetControl, _, _ := newFakeTiCDCMemberManager()

		if test.errSync {
			fakeSetControl.SetCreateStatefulSetError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.suspendComponent != nil {
			tmm.suspender.(*suspender.FakeSuspender).SuspendComponentFunc = func(c v1alpha1.Cluster, mt v1alpha1.MemberType) (bool, error) {
				return test.suspendComponent()
			}
		}

		err := tmm.Sync(tc)
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}

		g.Expect(tc.Spec).To(Equal(oldSpec))
	}

	tests := []testcase{
		{
			name:    "normal",
			errSync: false,
			err:     false,
		},
		{
			name:    "normal with tls",
			errSync: false,
			err:     false,
			tls:     true,
		},
		{
			name:    "error when sync",
			errSync: true,
			err:     true,
		},
		{
			name:             "skip create when suspend",
			suspendComponent: func() (bool, error) { return true, nil },
			errSync:          true,
			err:              false,
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestTiCDCMemberManagerSyncUpdate(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                string
		modify              func(cluster *v1alpha1.TidbCluster)
		errSync             bool
		statusChange        func(*apps.StatefulSet)
		status              v1alpha1.MemberPhase
		err                 bool
		expectStatefulSetFn func(*GomegaWithT, *apps.StatefulSet, error)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		tc := newTidbClusterForCDC()
		tc.Status.TiCDC.Phase = v1alpha1.NormalPhase
		tc.Status.TiCDC.StatefulSet = &apps.StatefulSetStatus{ReadyReplicas: 1}

		ns := tc.GetNamespace()
		tcName := tc.GetName()

		tmm, fakeSetControl, _, _ := newFakeTiCDCMemberManager()

		if test.statusChange == nil {
			fakeSetControl.SetStatusChange(func(set *apps.StatefulSet) {
				set.Status.Replicas = *set.Spec.Replicas
				set.Status.CurrentRevision = "ticdc-1"
				set.Status.UpdateRevision = "ticdc-1"
				observedGeneration := int64(1)
				set.Status.ObservedGeneration = observedGeneration
			})
		} else {
			fakeSetControl.SetStatusChange(test.statusChange)
		}

		err := tmm.Sync(tc)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(tc.Status.TiCDC.Phase).To(Equal(test.status))

		_, err = tmm.deps.StatefulSetLister.StatefulSets(ns).Get(controller.TiCDCMemberName(tcName))
		g.Expect(err).NotTo(HaveOccurred())

		tc1 := tc.DeepCopy()
		test.modify(tc1)

		if test.errSync {
			fakeSetControl.SetUpdateStatefulSetError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		err = tmm.Sync(tc1)
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}

		if test.expectStatefulSetFn != nil {
			set, err := tmm.deps.StatefulSetLister.StatefulSets(ns).Get(controller.TiCDCMemberName(tcName))
			test.expectStatefulSetFn(g, set, err)
		}
	}

	tests := []testcase{
		{
			name: "normal",
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TiCDC.Replicas = 5
			},
			errSync: false,
			err:     false,
			status:  v1alpha1.NormalPhase,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(int(*set.Spec.Replicas)).To(Equal(4)) // scale out one-by-one now
			},
		},
		{
			name: "error when update statefulset",
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TiCDC.Replicas = 5
			},
			errSync: true,
			err:     true,
			status:  v1alpha1.NormalPhase,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestTiCDCMemberManagerTiCDCStatefulSetIsUpgrading(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name            string
		setUpdate       func(*apps.StatefulSet)
		hasPod          bool
		updatePod       func(*corev1.Pod)
		errExpectFn     func(*GomegaWithT, error)
		expectUpgrading bool
	}
	testFn := func(test *testcase, t *testing.T) {
		pmm, _, _, indexers := newFakeTiCDCMemberManager()
		tc := newTidbClusterForCDC()
		tc.Status.TiCDC.StatefulSet = &apps.StatefulSetStatus{
			UpdateRevision: "v3",
		}

		set := &apps.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: metav1.NamespaceDefault,
			},
		}
		if test.setUpdate != nil {
			test.setUpdate(set)
		}

		if test.hasPod {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        ordinalPodName(v1alpha1.TiCDCMemberType, tc.GetName(), 0),
					Namespace:   metav1.NamespaceDefault,
					Annotations: map[string]string{},
					Labels:      label.New().Instance(tc.GetInstanceName()).TiCDC().Labels(),
				},
			}
			if test.updatePod != nil {
				test.updatePod(pod)
			}
			indexers.pod.Add(pod)
		}
		b, err := pmm.statefulSetIsUpgradingFn(
			pmm.deps.PodLister,
			pmm.deps.PDControl,
			set, tc)
		if test.errExpectFn != nil {
			test.errExpectFn(g, err)
		}
		if test.expectUpgrading {
			g.Expect(b).To(BeTrue())
		} else {
			g.Expect(b).NotTo(BeTrue())
		}
	}
	tests := []testcase{
		{
			name: "stateful set is upgrading",
			setUpdate: func(set *apps.StatefulSet) {
				set.Status.CurrentRevision = "v1"
				set.Status.UpdateRevision = "v2"
				set.Status.ObservedGeneration = 1000
			},
			hasPod:          false,
			updatePod:       nil,
			errExpectFn:     nil,
			expectUpgrading: true,
		},
		{
			name:            "pod don't have revision hash",
			setUpdate:       nil,
			hasPod:          true,
			updatePod:       nil,
			errExpectFn:     nil,
			expectUpgrading: false,
		},
		{
			name:      "pod have revision hash, not equal statefulset's",
			setUpdate: nil,
			hasPod:    true,
			updatePod: func(pod *corev1.Pod) {
				pod.Labels[apps.ControllerRevisionHashLabelKey] = "v2"
			},
			errExpectFn:     nil,
			expectUpgrading: true,
		},
		{
			name:      "pod have revision hash, equal statefulset's",
			setUpdate: nil,
			hasPod:    true,
			updatePod: func(pod *corev1.Pod) {
				pod.Labels[apps.ControllerRevisionHashLabelKey] = "v3"
			},
			errExpectFn:     nil,
			expectUpgrading: false,
		},
	}

	for i := range tests {
		t.Logf(tests[i].name)
		testFn(&tests[i], t)
	}
}

func TestTiCDCMemberManagerSyncTidbClusterStatus(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name             string
		updateTC         func(*v1alpha1.TidbCluster)
		updateSts        func(*apps.StatefulSet)
		upgradingFn      func(corelisters.PodLister, pdapi.PDControlInterface, *apps.StatefulSet, *v1alpha1.TidbCluster) (bool, error)
		healthInfo       map[string]bool
		beforeSyncStatus func(tc *v1alpha1.TidbCluster, pmm *ticdcMemberManager, indexer *fakeIndexers)
		errExpectFn      func(*GomegaWithT, error)
		tcExpectFn       func(*GomegaWithT, *v1alpha1.TidbCluster)
	}
	spec := apps.StatefulSetSpec{
		Replicas: pointer.Int32Ptr(3),
	}
	status := apps.StatefulSetStatus{
		Replicas: int32(3),
	}
	testFn := func(test *testcase, t *testing.T) {
		tc := newTidbClusterForCDC()
		tc.Spec.TiCDC.Replicas = int32(3)
		set := &apps.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{},
			Spec:       spec,
			Status:     status,
		}
		if test.updateTC != nil {
			test.updateTC(tc)
		}
		if test.updateSts != nil {
			test.updateSts(set)
		}
		pmm, _, tidbControl, indexers := newFakeTiCDCMemberManager()

		if test.upgradingFn != nil {
			pmm.statefulSetIsUpgradingFn = test.upgradingFn
		}
		if test.healthInfo != nil {
			tidbControl.SetHealth(test.healthInfo)
		}

		if test.beforeSyncStatus != nil {
			test.beforeSyncStatus(tc, pmm, indexers)
		}

		err := pmm.syncTiCDCStatus(tc, set)
		if test.errExpectFn != nil {
			test.errExpectFn(g, err)
		}
		if test.tcExpectFn != nil {
			test.tcExpectFn(g, tc)
		}
	}
	tests := []testcase{
		{
			name:     "whether statefulset is upgrading returns failed",
			updateTC: nil,
			upgradingFn: func(lister corelisters.PodLister, pc pdapi.PDControlInterface, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return false, fmt.Errorf("whether upgrading failed")
			},
			healthInfo: map[string]bool{},
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "whether upgrading failed")).To(BeTrue())
			},
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.TiCDC.StatefulSet.Replicas).To(Equal(int32(3)))
			},
		},
		{
			name:     "statefulset is upgrading",
			updateTC: nil,
			upgradingFn: func(lister corelisters.PodLister, pc pdapi.PDControlInterface, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return true, nil
			},
			healthInfo:  map[string]bool{},
			errExpectFn: nil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.TiCDC.StatefulSet.Replicas).To(Equal(int32(3)))
				g.Expect(tc.Status.TiCDC.Phase).To(Equal(v1alpha1.UpgradePhase))
			},
		},
		{
			name:     "statefulset is not upgrading",
			updateTC: nil,
			upgradingFn: func(lister corelisters.PodLister, pc pdapi.PDControlInterface, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return false, nil
			},
			healthInfo:  map[string]bool{},
			errExpectFn: nil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.TiCDC.StatefulSet.Replicas).To(Equal(int32(3)))
				g.Expect(tc.Status.TiCDC.Phase).To(Equal(v1alpha1.NormalPhase))
			},
		},
		{
			name:     "get health empty",
			updateTC: nil,
			updateSts: func(sts *apps.StatefulSet) {
				sts.Status = apps.StatefulSetStatus{
					Replicas: 0,
				}
			},
			upgradingFn: func(lister corelisters.PodLister, pc pdapi.PDControlInterface, set *apps.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return false, nil
			},
			healthInfo:  map[string]bool{},
			errExpectFn: errExpectNil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.TiCDC.StatefulSet.Replicas).To(Equal(int32(0)))
			},
		},
		{
			name:     "pod 1 isn't existing",
			updateTC: nil,
			updateSts: func(sts *apps.StatefulSet) {
				sts.Status = apps.StatefulSetStatus{
					Replicas: 3,
				}
			},
			beforeSyncStatus: func(tc *v1alpha1.TidbCluster, m *ticdcMemberManager, indexer *fakeIndexers) {
				// mock pods
				for i := int32(0); i < 3; i++ {
					if i == 1 {
						continue
					}
					indexer.pod.Add(&corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name:      ordinalPodName(v1alpha1.TiCDCMemberType, tc.GetName(), i),
							Namespace: metav1.NamespaceDefault,
							Labels:    label.New().Instance(tc.GetInstanceName()).TiCDC().Labels(),
						},
					})
				}

				// mock status of captures
				cdcControl := m.deps.CDCControl.(*controller.FakeTiCDCControl)
				cdcControl.GetStatusFn = func(tc *v1alpha1.TidbCluster, ordinal int32) (*controller.CaptureStatus, error) {
					return &controller.CaptureStatus{}, nil
				}
			},
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				// check status of statefulset
				g.Expect(tc.Status.TiCDC.StatefulSet.Replicas).To(Equal(int32(3)))
				// check status of captures
				g.Expect(tc.Status.TiCDC.Captures).To(HaveLen(2))
				for podName, capture := range tc.Status.TiCDC.Captures {
					g.Expect(podName).NotTo(Equal(ordinalPodName(v1alpha1.TiCDCMemberType, tc.GetName(), 1)))
					g.Expect(capture.Ready).To(BeTrue())
				}
				// check synced
				g.Expect(tc.Status.TiCDC.Synced).To(BeFalse())
			},
		},
		{
			name:     "capture 1 is unready",
			updateTC: nil,
			updateSts: func(sts *apps.StatefulSet) {
				sts.Status = apps.StatefulSetStatus{
					Replicas: 3,
				}
			},
			beforeSyncStatus: func(tc *v1alpha1.TidbCluster, m *ticdcMemberManager, indexer *fakeIndexers) {
				// mock pods
				for i := int32(0); i < 3; i++ {
					indexer.pod.Add(&corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name:      ordinalPodName(v1alpha1.TiCDCMemberType, tc.GetName(), i),
							Namespace: metav1.NamespaceDefault,
							Labels:    label.New().Instance(tc.GetInstanceName()).TiCDC().Labels(),
						},
					})
				}

				// mock status of captures
				cdcControl := m.deps.CDCControl.(*controller.FakeTiCDCControl)
				cdcControl.GetStatusFn = func(tc *v1alpha1.TidbCluster, ordinal int32) (*controller.CaptureStatus, error) {
					if ordinal == 1 {
						return nil, fmt.Errorf("mock err")
					}

					return &controller.CaptureStatus{}, nil
				}
			},
			errExpectFn: errExpectNil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				// check status of statefulset
				g.Expect(tc.Status.TiCDC.StatefulSet.Replicas).To(Equal(int32(3)))
				// check status of captures
				g.Expect(tc.Status.TiCDC.Captures).To(HaveLen(3))
				for podName, capture := range tc.Status.TiCDC.Captures {
					if podName == ordinalPodName(v1alpha1.TiCDCMemberType, tc.GetName(), 1) {
						g.Expect(capture.Ready).To(BeFalse())
					} else {
						g.Expect(capture.Ready).To(BeTrue())
					}
				}
				// check synced
				g.Expect(tc.Status.TiCDC.Synced).To(BeFalse())
			},
		},
	}

	for i := range tests {
		t.Logf(tests[i].name)
		testFn(&tests[i], t)
	}
}

func TestGetNewTiCDCStatefulSet(t *testing.T) {
	tests := []struct {
		name    string
		tc      v1alpha1.TidbCluster
		testSts func(sts *apps.StatefulSet)
	}{
		{
			name: "TiCDC spec storageVolumes",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiCDC: &v1alpha1.TiCDCSpec{StorageVolumes: []v1alpha1.StorageVolume{
						{
							Name:        "sort-dir",
							StorageSize: "2Gi",
							MountPath:   "/var/lib/sort-dir",
						},
					}},
				},
			},
			testSts: func(sts *apps.StatefulSet) {
				g := NewGomegaWithT(t)
				q, _ := resource.ParseQuantity("2Gi")
				g.Expect(sts.Spec.VolumeClaimTemplates).To(Equal([]corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: v1alpha1.TiCDCMemberType.String() + "-sort-dir",
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
				index := 0
				g.Expect(sts.Spec.Template.Spec.Containers[0].VolumeMounts[index]).To(Equal(corev1.VolumeMount{
					Name:      fmt.Sprintf("%s-%s", v1alpha1.TiCDCMemberType, "sort-dir"),
					MountPath: "/var/lib/sort-dir",
				}))
			},
		},
		{
			name: "TiCDC additional volumes",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiCDC: &v1alpha1.TiCDCSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							AdditionalVolumes: []corev1.Volume{{Name: "test", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}},
						},
					},
				},
			},
			testSts: testAdditionalVolumes(t, []corev1.Volume{{Name: "test", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sts, _ := getNewTiCDCStatefulSet(&tt.tc, nil)
			tt.testSts(sts)
		})
	}
}

func newFakeTiCDCMemberManager() (*ticdcMemberManager, *controller.FakeStatefulSetControl, *controller.FakeTiDBControl, *fakeIndexers) {
	fakeDeps := controller.NewFakeDependencies()
	tmm := &ticdcMemberManager{
		deps:              fakeDeps,
		scaler:            NewTiCDCScaler(fakeDeps),
		suspender:         suspender.NewFakeSuspender(),
		podVolumeModifier: &volumes.FakePodVolumeModifier{},
	}
	tmm.statefulSetIsUpgradingFn = ticdcStatefulSetIsUpgrading
	indexers := &fakeIndexers{
		pod:    fakeDeps.KubeInformerFactory.Core().V1().Pods().Informer().GetIndexer(),
		tc:     fakeDeps.InformerFactory.Pingcap().V1alpha1().TidbClusters().Informer().GetIndexer(),
		svc:    fakeDeps.KubeInformerFactory.Core().V1().Services().Informer().GetIndexer(),
		eps:    fakeDeps.KubeInformerFactory.Core().V1().Endpoints().Informer().GetIndexer(),
		secret: fakeDeps.KubeInformerFactory.Core().V1().Secrets().Informer().GetIndexer(),
		set:    fakeDeps.KubeInformerFactory.Apps().V1().StatefulSets().Informer().GetIndexer(),
	}
	setControl := fakeDeps.StatefulSetControl.(*controller.FakeStatefulSetControl)
	tidbControl := fakeDeps.TiDBControl.(*controller.FakeTiDBControl)
	return tmm, setControl, tidbControl, indexers
}

func newTidbClusterForCDC() *v1alpha1.TidbCluster {
	return &v1alpha1.TidbCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TidbCluster",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("test"),
		},
		Spec: v1alpha1.TidbClusterSpec{
			TiCDC: &v1alpha1.TiCDCSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image: v1alpha1.TiCDCMemberType.String(),
				},
				ResourceRequirements: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
				},
				Replicas: 3,
			},
		},
	}
}
