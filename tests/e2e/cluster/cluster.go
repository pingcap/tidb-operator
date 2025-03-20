// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cluster

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/semver/v3"
	_ "github.com/go-sql-driver/mysql"

	//nolint: stylecheck // too many changes, refactor later
	. "github.com/onsi/ginkgo/v2"
	//nolint: stylecheck // too many changes, refactor later
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	cacheddiscovery "k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/tests/e2e/config"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/data"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/k8s"
	utiltidb "github.com/pingcap/tidb-operator/tests/e2e/utils/tidb"
)

const (
	createClusterTimeout = 10 * time.Minute
	createClusterPolling = 5 * time.Second

	deleteClusterTimeout = 10 * time.Minute
	deleteClusterPolling = 5 * time.Second

	suspendResumePolling = 5 * time.Second

	logLevelConfig = "log.level = 'warn'"
)

func initK8sClient() (kubernetes.Interface, client.Client, *rest.Config) {
	restConfig, err := k8s.LoadConfig()
	Expect(err).NotTo(HaveOccurred())

	clientset, err := kubernetes.NewForConfig(restConfig)
	Expect(err).NotTo(HaveOccurred())

	scheme := runtime.NewScheme()
	metav1.AddToGroupVersion(scheme, schema.GroupVersion{Version: "v1"})
	Expect(clientgoscheme.AddToScheme(scheme)).To(Succeed())
	Expect(v1alpha1.Install(scheme)).To(Succeed())

	// also init a controller-runtime client
	k8sClient, err := client.New(restConfig, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	return clientset, k8sClient, restConfig
}

func LoadClientRawConfig() (clientcmdapi.Config, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{ClusterDefaults: clientcmd.ClusterDefaults}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides).RawConfig()
}

// nolint: goconst // just for test
var _ = Describe("TiDB Cluster", func() {
	var (
		clientSet   kubernetes.Interface
		k8sClient   client.Client
		restConfig  *rest.Config
		fw          k8s.PortForwarder
		yamlApplier *k8s.YAMLApplier

		ns  *corev1.Namespace
		tc  *v1alpha1.Cluster
		ctx = context.Background()
	)

	BeforeEach(func() {
		// TODO: only run once
		clientRawConfig, err := LoadClientRawConfig()
		Expect(err).To(BeNil())
		fw, err = k8s.NewPortForwarder(ctx, config.NewSimpleRESTClientGetter(&clientRawConfig))
		Expect(err).To(BeNil())

		clientSet, k8sClient, restConfig = initK8sClient()
		_ = restConfig

		dynamicClient := dynamic.NewForConfigOrDie(restConfig)
		cachedDiscovery := cacheddiscovery.NewMemCacheClient(clientSet.Discovery())
		restmapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscovery)
		restmapper.Reset()
		yamlApplier = k8s.NewYAMLApplier(dynamicClient, restmapper)
		ns = data.NewNamespace()
		tc = data.NewCluster(ns.Name, "tc")

		By(fmt.Sprintf("Creating a Cluster in namespace %s", tc.Namespace))
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())
		Expect(k8sClient.Create(ctx, tc)).To(Succeed())

		By(fmt.Sprintf("Waiting for the Cluster in namespace %s to be ready", tc.Namespace))
		Eventually(func(g Gomega) {
			var tcGet v1alpha1.Cluster
			g.Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: tc.Namespace, Name: tc.Name}, &tcGet)).To(Succeed())
			g.Expect(tcGet.Status.ObservedGeneration).To(Equal(tcGet.Generation))
		}).WithTimeout(createClusterTimeout).WithPolling(createClusterPolling).Should(Succeed())

		DeferCleanup(func() {
			// uncomment the following lines to keep the resources for debugging
			//if CurrentSpecReport().Failed() {
			//	return
			//}
			By(fmt.Sprintf("Deleting the Cluster in namespace %s", tc.Namespace))
			Expect(k8sClient.Delete(ctx, tc)).To(Succeed())

			By(fmt.Sprintf("Checking the Cluster in namespace %s has been deleted", tc.Namespace))
			Eventually(func(g Gomega) {
				err := k8sClient.Get(ctx, client.ObjectKey{Namespace: tc.Namespace, Name: tc.Name}, tc)
				g.Expect(errors.IsNotFound(err)).To(BeTrue())
			}).WithTimeout(deleteClusterTimeout).WithPolling(deleteClusterPolling).Should(Succeed())

			By(fmt.Sprintf("Checking resources have already been deleted in namespace %s", tc.Namespace))
			var podList corev1.PodList
			Expect(k8sClient.List(ctx, &podList, client.InNamespace(tc.Namespace), client.MatchingLabels{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyCluster:   tc.Name,
			})).To(Succeed())
			Expect(len(podList.Items)).To(Equal(0))
			var cmList corev1.ConfigMapList
			Expect(k8sClient.List(ctx, &cmList, client.InNamespace(tc.Namespace), client.MatchingLabels{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyCluster:   tc.Name,
			})).To(Succeed())
			Expect(len(cmList.Items)).To(Equal(0))
			var pvcList corev1.PersistentVolumeClaimList
			Expect(k8sClient.List(ctx, &pvcList, client.InNamespace(tc.Namespace), client.MatchingLabels{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyCluster:   tc.Name,
			})).To(Succeed())
			Expect(len(pvcList.Items)).To(Equal(0))

			var svcList corev1.ServiceList
			Expect(k8sClient.List(ctx, &svcList, client.InNamespace(tc.Namespace), client.MatchingLabels{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyCluster:   tc.Name,
			})).To(Succeed())
			Expect(len(svcList.Items)).To(Equal(0))

			By(fmt.Sprintf("Deleting the Namespace %s", ns.Name))
			Expect(k8sClient.Delete(ctx, ns)).To(Succeed())

			By("Checking namespace can be deleted")
			Eventually(func(g Gomega) {
				err := k8sClient.Get(ctx, client.ObjectKey{Name: ns.Name}, ns)
				g.Expect(errors.IsNotFound(err)).To(BeTrue())
			}).WithTimeout(deleteClusterTimeout).WithPolling(deleteClusterPolling).Should(Succeed())
		})
	})
	PContext("Evenly scheduling", func() {
		It("should create a tidb cluster that all components have 3 nodes", func() {
			pdg := data.NewPDGroup(ns.Name, "pdg", tc.Name, ptr.To(int32(3)), func(g *v1alpha1.PDGroup) {
				g.Spec.SchedulePolicies = []v1alpha1.SchedulePolicy{
					{
						Type: v1alpha1.SchedulePolicyTypeEvenlySpread,
						EvenlySpread: &v1alpha1.SchedulePolicyEvenlySpread{
							Topologies: []v1alpha1.ScheduleTopology{
								{
									Topology: v1alpha1.Topology{
										"zone": "zone-a",
									},
								},
								{
									Topology: v1alpha1.Topology{
										"zone": "zone-b",
									},
								},
								{
									Topology: v1alpha1.Topology{
										"zone": "zone-c",
									},
								},
							},
						},
					},
				}
			})
			kvg := data.NewTiKVGroup(ns.Name, "kvg", tc.Name, ptr.To(int32(3)), func(g *v1alpha1.TiKVGroup) {
				g.Spec.SchedulePolicies = []v1alpha1.SchedulePolicy{
					{
						Type: v1alpha1.SchedulePolicyTypeEvenlySpread,
						EvenlySpread: &v1alpha1.SchedulePolicyEvenlySpread{
							Topologies: []v1alpha1.ScheduleTopology{
								{
									Topology: v1alpha1.Topology{
										"zone": "zone-a",
									},
								},
								{
									Topology: v1alpha1.Topology{
										"zone": "zone-b",
									},
								},
								{
									Topology: v1alpha1.Topology{
										"zone": "zone-c",
									},
								},
							},
						},
					},
				}
			})
			dbg := data.NewTiDBGroup(ns.Name, "dbg", tc.Name, ptr.To(int32(3)), func(g *v1alpha1.TiDBGroup) {
				g.Spec.SchedulePolicies = []v1alpha1.SchedulePolicy{
					{
						Type: v1alpha1.SchedulePolicyTypeEvenlySpread,
						EvenlySpread: &v1alpha1.SchedulePolicyEvenlySpread{
							Topologies: []v1alpha1.ScheduleTopology{
								{
									Topology: v1alpha1.Topology{
										"zone": "zone-a",
									},
								},
								{
									Topology: v1alpha1.Topology{
										"zone": "zone-b",
									},
								},
								{
									Topology: v1alpha1.Topology{
										"zone": "zone-c",
									},
								},
							},
						},
					},
				}
			})
			flashg := data.NewTiFlashGroup(ns.Name, "flashg", tc.Name, ptr.To(int32(3)), func(g *v1alpha1.TiFlashGroup) {
				g.Spec.SchedulePolicies = []v1alpha1.SchedulePolicy{
					{
						Type: v1alpha1.SchedulePolicyTypeEvenlySpread,
						EvenlySpread: &v1alpha1.SchedulePolicyEvenlySpread{
							Topologies: []v1alpha1.ScheduleTopology{
								{
									Topology: v1alpha1.Topology{
										"zone": "zone-a",
									},
								},
								{
									Topology: v1alpha1.Topology{
										"zone": "zone-b",
									},
								},
								{
									Topology: v1alpha1.Topology{
										"zone": "zone-c",
									},
								},
							},
						},
					},
				}
			})
			cdcg := data.NewTiCDCGroup(ns.Name, "cdcg", tc.Name, ptr.To(int32(3)), func(g *v1alpha1.TiCDCGroup) {
				g.Spec.SchedulePolicies = []v1alpha1.SchedulePolicy{
					{
						Type: v1alpha1.SchedulePolicyTypeEvenlySpread,
						EvenlySpread: &v1alpha1.SchedulePolicyEvenlySpread{
							Topologies: []v1alpha1.ScheduleTopology{
								{
									Topology: v1alpha1.Topology{
										"zone": "zone-a",
									},
								},
								{
									Topology: v1alpha1.Topology{
										"zone": "zone-b",
									},
								},
								{
									Topology: v1alpha1.Topology{
										"zone": "zone-c",
									},
								},
							},
						},
					},
				}
			})

			Expect(k8sClient.Create(ctx, pdg)).To(Succeed())
			Expect(k8sClient.Create(ctx, kvg)).To(Succeed())
			Expect(k8sClient.Create(ctx, dbg)).To(Succeed())
			Expect(k8sClient.Create(ctx, flashg)).To(Succeed())
			Expect(k8sClient.Create(ctx, cdcg)).To(Succeed())

			nodes := &corev1.NodeList{}
			Expect(k8sClient.List(ctx, nodes)).Should(Succeed())
			nodeToZone := map[string]int{}
			for _, node := range nodes.Items {
				switch node.Labels["zone"] {
				case "zone-a":
					nodeToZone[node.Name] = 0
				case "zone-b":
					nodeToZone[node.Name] = 1
				case "zone-c":
					nodeToZone[node.Name] = 2
				}
			}
			By("Checking the status of the cluster and the connection to the TiDB service")
			Eventually(func(g Gomega) {
				tcGet, ready := utiltidb.IsClusterReady(k8sClient, tc.Name, tc.Namespace)
				g.Expect(ready).To(BeTrue())

				// TODO: move cluster status check into `IsClusterReady`?
				for _, compStatus := range tcGet.Status.Components {
					switch compStatus.Kind {
					case v1alpha1.ComponentKindPD, v1alpha1.ComponentKindTiKV, v1alpha1.ComponentKindTiDB,
						v1alpha1.ComponentKindTiFlash, v1alpha1.ComponentKindTiCDC:
						g.Expect(compStatus.Replicas).To(Equal(int32(3)))
					default:
						g.Expect(compStatus.Replicas).To(BeZero())
					}
				}

				g.Expect(utiltidb.AreAllInstancesReady(k8sClient, pdg,
					[]*v1alpha1.TiKVGroup{kvg}, []*v1alpha1.TiDBGroup{dbg},
					[]*v1alpha1.TiFlashGroup{flashg}, []*v1alpha1.TiCDCGroup{cdcg})).To(Succeed())

				pods := &corev1.PodList{}
				Expect(k8sClient.List(ctx, pods, client.InNamespace(tc.Namespace))).Should(Succeed())
				var components [5][3]int
				for _, pod := range pods.Items {
					index, ok := nodeToZone[pod.Spec.NodeName]
					g.Expect(ok).To(BeTrue())
					switch pod.Labels[v1alpha1.LabelKeyComponent] {
					case v1alpha1.LabelValComponentPD:
						components[0][index] += 1
					case v1alpha1.LabelValComponentTiDB:
						components[1][index] += 1
					case v1alpha1.LabelValComponentTiKV:
						components[2][index] += 1
					case v1alpha1.LabelValComponentTiFlash:
						components[3][index] += 1
					case v1alpha1.LabelValComponentTiCDC:
						components[4][index] += 1
					default:
						Fail("unexpected component " + pod.Labels[v1alpha1.LabelKeyComponent])
					}
				}
				g.Expect(components).To(Equal([5][3]int{{1, 1, 1}, {1, 1, 1}, {1, 1, 1}, {1, 1, 1}, {1, 1, 1}}))
			}).WithTimeout(createClusterTimeout).WithPolling(createClusterPolling).Should(Succeed())
		})
	})

	Context("Basic", func() {
		It("should create a minimal tidb cluster", func() {
			pdg := data.NewPDGroup(ns.Name, "pdg", tc.Name, ptr.To(int32(1)), nil)
			kvg := data.NewTiKVGroup(ns.Name, "kvg", tc.Name, ptr.To(int32(1)), nil)
			dbg := data.NewTiDBGroup(ns.Name, "dbg", tc.Name, ptr.To(int32(1)), nil)
			flashg := data.NewTiFlashGroup(ns.Name, "flashg", tc.Name, ptr.To(int32(1)), nil)
			cdcg := data.NewTiCDCGroup(ns.Name, "cdcg", tc.Name, ptr.To(int32(1)), nil)

			Expect(k8sClient.Create(ctx, pdg)).To(Succeed())
			Expect(k8sClient.Create(ctx, kvg)).To(Succeed())
			Expect(k8sClient.Create(ctx, dbg)).To(Succeed())
			Expect(k8sClient.Create(ctx, flashg)).To(Succeed())
			Expect(k8sClient.Create(ctx, cdcg)).To(Succeed())

			By("Checking the status of the cluster and the connection to the TiDB service")
			Eventually(func(g Gomega) {
				tcGet, ready := utiltidb.IsClusterReady(k8sClient, tc.Name, tc.Namespace)
				g.Expect(ready).To(BeTrue())

				// TODO: move cluster status check into `IsClusterReady`?
				for _, compStatus := range tcGet.Status.Components {
					switch compStatus.Kind {
					case v1alpha1.ComponentKindPD, v1alpha1.ComponentKindTiKV, v1alpha1.ComponentKindTiDB,
						v1alpha1.ComponentKindTiFlash, v1alpha1.ComponentKindTiCDC:
						g.Expect(compStatus.Replicas).To(Equal(int32(1)))
					default:
						g.Expect(compStatus.Replicas).To(BeZero())
					}
				}

				g.Expect(utiltidb.AreAllInstancesReady(k8sClient, pdg,
					[]*v1alpha1.TiKVGroup{kvg}, []*v1alpha1.TiDBGroup{dbg},
					[]*v1alpha1.TiFlashGroup{flashg}, []*v1alpha1.TiCDCGroup{cdcg})).To(Succeed())

				g.Expect(utiltidb.IsTiDBConnectable(ctx, k8sClient, fw,
					tc.Namespace, tc.Name, dbg.Name, "root", "", "")).To(Succeed())
			}).WithTimeout(createClusterTimeout).WithPolling(createClusterPolling).Should(Succeed())

			By("Checking log tailer sidercar container")
			var tidbPodList corev1.PodList
			Expect(k8sClient.List(ctx, &tidbPodList, client.InNamespace(tc.Namespace), client.MatchingLabels{
				v1alpha1.LabelKeyCluster: tc.Name,
				v1alpha1.LabelKeyGroup:   dbg.Name,
			})).To(Succeed())
			Expect(len(tidbPodList.Items)).To(Equal(1))
			tidbPod := tidbPodList.Items[0]
			Expect(tidbPod.Spec.InitContainers).To(HaveLen(1)) // sidercar container in `initContainers`
			Expect(tidbPod.Spec.InitContainers[0].Name).To(Equal(v1alpha1.ContainerNameTiDBSlowLog))

			var tiflashPodList corev1.PodList
			Expect(k8sClient.List(ctx, &tiflashPodList, client.InNamespace(tc.Namespace), client.MatchingLabels{
				v1alpha1.LabelKeyCluster: tc.Name,
				v1alpha1.LabelKeyGroup:   flashg.Name,
			})).To(Succeed())
			Expect(len(tiflashPodList.Items)).To(Equal(1))
			tiflashPod := tiflashPodList.Items[0]
			Expect(tiflashPod.Spec.InitContainers).To(HaveLen(2))
			Expect(tiflashPod.Spec.InitContainers[0].Name).To(Equal(v1alpha1.ContainerNameTiFlashServerLog))
			Expect(tiflashPod.Spec.InitContainers[1].Name).To(Equal(v1alpha1.ContainerNameTiFlashErrorLog))
		})

		It("should suspend and resume the tidb cluster", func() {
			checkSuspendCondtion := func(g Gomega, condStatus metav1.ConditionStatus) {
				var pdGroup v1alpha1.PDGroup
				g.Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: "pdg"}, &pdGroup)).To(Succeed())
				g.Expect(meta.IsStatusConditionPresentAndEqual(pdGroup.Status.Conditions, v1alpha1.CondSuspended, condStatus)).To(BeTrue())

				var tikvGroup v1alpha1.TiKVGroup
				g.Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: "kvg"}, &tikvGroup)).To(Succeed())
				g.Expect(meta.IsStatusConditionPresentAndEqual(tikvGroup.Status.Conditions, v1alpha1.CondSuspended, condStatus)).To(BeTrue())

				var tidbGroup v1alpha1.TiDBGroup
				g.Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: "dbg"}, &tidbGroup)).To(Succeed())
				g.Expect(meta.IsStatusConditionPresentAndEqual(tidbGroup.Status.Conditions, v1alpha1.CondSuspended, condStatus)).To(BeTrue())

				var flashGroup v1alpha1.TiFlashGroup
				g.Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: "flashg"}, &flashGroup)).To(Succeed())
				g.Expect(meta.IsStatusConditionPresentAndEqual(
					flashGroup.Status.Conditions, v1alpha1.CondSuspended, condStatus)).To(BeTrue())

				var cdcGroup v1alpha1.TiCDCGroup
				g.Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: "cdcg"}, &cdcGroup)).To(Succeed())
				g.Expect(meta.IsStatusConditionPresentAndEqual(
					cdcGroup.Status.Conditions, v1alpha1.CondSuspended, condStatus)).To(BeTrue())

				var pdList v1alpha1.PDList
				g.Expect(k8sClient.List(ctx, &pdList, client.InNamespace(tc.Namespace), client.MatchingLabels{
					v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
					v1alpha1.LabelKeyCluster:   tc.Name,
					v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
					v1alpha1.LabelKeyGroup:     pdGroup.Name,
				})).To(Succeed())
				g.Expect(len(pdList.Items)).To(Equal(1))
				pd := pdList.Items[0]
				g.Expect(pd.Spec.Cluster.Name).To(Equal(tc.Name))
				g.Expect(meta.IsStatusConditionPresentAndEqual(pd.Status.Conditions, v1alpha1.CondSuspended, condStatus)).To(BeTrue())

				var tikvList v1alpha1.TiKVList
				g.Expect(k8sClient.List(ctx, &tikvList, client.InNamespace(tc.Namespace), client.MatchingLabels{
					v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
					v1alpha1.LabelKeyCluster:   tc.Name,
					v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiKV,
					v1alpha1.LabelKeyGroup:     tikvGroup.Name,
				})).To(Succeed())
				g.Expect(len(tikvList.Items)).To(Equal(1))
				tikv := tikvList.Items[0]
				g.Expect(tikv.Spec.Cluster.Name).To(Equal(tc.Name))
				g.Expect(meta.IsStatusConditionPresentAndEqual(tikv.Status.Conditions, v1alpha1.CondSuspended, condStatus)).To(BeTrue())

				var tidbList v1alpha1.TiDBList
				g.Expect(k8sClient.List(ctx, &tidbList, client.InNamespace(tc.Namespace), client.MatchingLabels{
					v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
					v1alpha1.LabelKeyCluster:   tc.Name,
					v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiDB,
					v1alpha1.LabelKeyGroup:     tidbGroup.Name,
				})).To(Succeed())
				g.Expect(len(tidbList.Items)).To(Equal(1))
				tidb := tidbList.Items[0]
				g.Expect(tidb.Spec.Cluster.Name).To(Equal(tc.Name))
				g.Expect(meta.IsStatusConditionPresentAndEqual(tidb.Status.Conditions, v1alpha1.CondSuspended, condStatus)).To(BeTrue())

				var flashList v1alpha1.TiFlashList
				g.Expect(k8sClient.List(ctx, &flashList, client.InNamespace(tc.Namespace), client.MatchingLabels{
					v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
					v1alpha1.LabelKeyCluster:   tc.Name,
					v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiFlash,
					v1alpha1.LabelKeyGroup:     flashGroup.Name,
				})).To(Succeed())
				g.Expect(len(flashList.Items)).To(Equal(1))
				flash := flashList.Items[0]
				g.Expect(flash.Spec.Cluster.Name).To(Equal(tc.Name))
				g.Expect(meta.IsStatusConditionPresentAndEqual(flash.Status.Conditions, v1alpha1.CondSuspended, condStatus)).To(BeTrue())

				var cdcList v1alpha1.TiCDCList
				g.Expect(k8sClient.List(ctx, &cdcList, client.InNamespace(tc.Namespace), client.MatchingLabels{
					v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
					v1alpha1.LabelKeyCluster:   tc.Name,
					v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiCDC,
					v1alpha1.LabelKeyGroup:     cdcGroup.Name,
				})).To(Succeed())
				g.Expect(len(cdcList.Items)).To(Equal(1))
				cdc := cdcList.Items[0]
				g.Expect(cdc.Spec.Cluster.Name).To(Equal(tc.Name))
				g.Expect(meta.IsStatusConditionPresentAndEqual(cdc.Status.Conditions, v1alpha1.CondSuspended, condStatus)).To(BeTrue())

				var tcGet v1alpha1.Cluster
				g.Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: tc.Namespace, Name: tc.Name}, &tcGet)).To(Succeed())
				g.Expect(meta.IsStatusConditionPresentAndEqual(tcGet.Status.Conditions, v1alpha1.ClusterCondSuspended, condStatus)).To(BeTrue())
			}

			pdg := data.NewPDGroup(ns.Name, "pdg", tc.Name, ptr.To(int32(1)), nil)
			kvg := data.NewTiKVGroup(ns.Name, "kvg", tc.Name, ptr.To(int32(1)), nil)
			dbg := data.NewTiDBGroup(ns.Name, "dbg", tc.Name, ptr.To(int32(1)), nil)
			flashg := data.NewTiFlashGroup(ns.Name, "flashg", tc.Name, ptr.To(int32(1)), nil)
			cdcg := data.NewTiCDCGroup(ns.Name, "cdcg", tc.Name, ptr.To(int32(1)), nil)

			Expect(k8sClient.Create(ctx, pdg)).To(Succeed())
			Expect(k8sClient.Create(ctx, kvg)).To(Succeed())
			Expect(k8sClient.Create(ctx, dbg)).To(Succeed())
			Expect(k8sClient.Create(ctx, flashg)).To(Succeed())
			Expect(k8sClient.Create(ctx, cdcg)).To(Succeed())

			By("Waiting for the cluster to be ready")
			var tcGet *v1alpha1.Cluster
			Eventually(func(g Gomega) {
				var ready bool
				tcGet, ready = utiltidb.IsClusterReady(k8sClient, tc.Name, tc.Namespace)
				g.Expect(ready).To(BeTrue())

				g.Expect(utiltidb.AreAllInstancesReady(k8sClient, pdg,
					[]*v1alpha1.TiKVGroup{kvg}, []*v1alpha1.TiDBGroup{dbg},
					[]*v1alpha1.TiFlashGroup{flashg}, []*v1alpha1.TiCDCGroup{cdcg})).To(Succeed())
			}).WithTimeout(createClusterTimeout).WithPolling(createClusterPolling).Should(Succeed())

			By("Checking suspend condition is False")
			checkSuspendCondtion(Default, metav1.ConditionFalse)

			By("Suspending the TiDB cluster")
			tcGet.Spec.SuspendAction = &v1alpha1.SuspendAction{SuspendCompute: true}
			Expect(k8sClient.Update(ctx, tcGet)).To(Succeed())

			By("Checking suspend condition is True")
			Eventually(func(g Gomega) {
				checkSuspendCondtion(g, metav1.ConditionTrue)
			}).WithTimeout(2 * time.Minute).WithPolling(suspendResumePolling).Should(Succeed())

			By("Checking all Pods are deleted")
			podList, err := clientSet.CoreV1().Pods(tc.Namespace).List(ctx, metav1.ListOptions{})
			Expect(err).To(BeNil())
			Expect(len(podList.Items)).To(BeZero())

			By("Resuming the TiDB cluster")
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: tc.Namespace, Name: tc.Name}, tcGet)).To(Succeed())
			tcGet.Spec.SuspendAction = nil
			Expect(k8sClient.Update(ctx, tcGet)).To(Succeed())

			By("Checking suspend condition is False")
			Eventually(func(g Gomega) {
				checkSuspendCondtion(g, metav1.ConditionFalse)
			}).WithTimeout(time.Minute).WithPolling(suspendResumePolling).Should(Succeed())

			By("Checking the Pods are re-created")
			Eventually(func(g Gomega) {
				podList, err = clientSet.CoreV1().Pods(tc.Namespace).List(ctx, metav1.ListOptions{})
				g.Expect(err).To(BeNil())
				g.Expect(len(podList.Items)).To(Equal(5))
			}).WithTimeout(time.Minute).WithPolling(createClusterPolling).Should(Succeed())

			By("Checking the cluster is ready")
			Eventually(func(g Gomega) {
				_, ready := utiltidb.IsClusterReady(k8sClient, tc.Name, tc.Namespace)
				g.Expect(ready).To(BeTrue())

				g.Expect(utiltidb.AreAllInstancesReady(k8sClient, pdg,
					[]*v1alpha1.TiKVGroup{kvg}, []*v1alpha1.TiDBGroup{dbg},
					[]*v1alpha1.TiFlashGroup{flashg}, []*v1alpha1.TiCDCGroup{cdcg})).To(Succeed())
			}).WithTimeout(createClusterTimeout).WithPolling(createClusterPolling).Should(Succeed())

			By("Suspend the TiDB cluster again before deleting it")
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: tc.Namespace, Name: tc.Name}, tcGet)).To(Succeed())
			tcGet.Spec.SuspendAction = &v1alpha1.SuspendAction{SuspendCompute: true}
			Expect(k8sClient.Update(ctx, tcGet)).To(Succeed())

			By("Checking suspend condition is True")
			Eventually(func(g Gomega) {
				checkSuspendCondtion(g, metav1.ConditionTrue)
			}).WithTimeout(2 * time.Minute).WithPolling(suspendResumePolling).Should(Succeed())
		})

		It("should be able to scale in/out", func() {
			pdg := data.NewPDGroup(ns.Name, "pdg", tc.Name, ptr.To(int32(1)), nil)
			kvg := data.NewTiKVGroup(ns.Name, "kvg", tc.Name, ptr.To(int32(1)), nil)
			dbg := data.NewTiDBGroup(ns.Name, "dbg", tc.Name, ptr.To(int32(1)), nil)
			Expect(k8sClient.Create(ctx, pdg)).To(Succeed())
			Expect(k8sClient.Create(ctx, kvg)).To(Succeed())
			Expect(k8sClient.Create(ctx, dbg)).To(Succeed())

			By("Waiting for the cluster to be ready")
			Eventually(func(g Gomega) {
				_, ready := utiltidb.IsClusterReady(k8sClient, tc.Name, tc.Namespace)
				g.Expect(ready).To(BeTrue())

				g.Expect(utiltidb.AreAllInstancesReady(k8sClient, pdg,
					[]*v1alpha1.TiKVGroup{kvg}, []*v1alpha1.TiDBGroup{dbg}, nil, nil)).To(Succeed())
			}).WithTimeout(createClusterTimeout).WithPolling(createClusterPolling).Should(Succeed())

			By("Recording the pod's UID")
			listOpts := metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=%s,%s=%s",
					v1alpha1.LabelKeyCluster, tc.Name, v1alpha1.LabelKeyGroup, dbg.Name),
			}
			podList, err := clientSet.CoreV1().Pods(tc.Namespace).List(ctx, listOpts)
			Expect(err).To(BeNil())
			Expect(len(podList.Items)).To(Equal(1))
			originalPodName, originalPodUID := podList.Items[0].Name, podList.Items[0].UID

			By("scale out tidb from 1 to 2")
			var dbgGet v1alpha1.TiDBGroup
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: tc.Namespace, Name: dbg.Name}, &dbgGet)).To(Succeed())
			dbgGet.Spec.Replicas = ptr.To(int32(2))
			Expect(k8sClient.Update(ctx, &dbgGet)).To(Succeed())

			podNameToUID := make(map[string]types.UID, 2)
			Eventually(func(g Gomega) {
				g.Expect(utiltidb.AreAllTiDBHealthy(k8sClient, &dbgGet)).To(Succeed())
				podList, err = clientSet.CoreV1().Pods(tc.Namespace).List(ctx, listOpts)
				g.Expect(err).To(BeNil())
				g.Expect(len(podList.Items)).To(Equal(2))

				// Should not recreate the pod
				for _, pod := range podList.Items {
					podNameToUID[pod.Name] = pod.UID
				}
				g.Expect(podNameToUID[originalPodName]).To(Equal(originalPodUID))
			}).WithTimeout(createClusterTimeout).WithPolling(createClusterPolling).Should(Succeed())

			By("scale in tidb from 2 to 1")
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: tc.Namespace, Name: dbg.Name}, &dbgGet)).To(Succeed())
			dbgGet.Spec.Replicas = ptr.To(int32(1))
			Expect(k8sClient.Update(ctx, &dbgGet)).To(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(utiltidb.AreAllTiDBHealthy(k8sClient, &dbgGet)).To(Succeed())
				podList, err = clientSet.CoreV1().Pods(tc.Namespace).List(ctx, listOpts)
				g.Expect(err).To(BeNil())
				g.Expect(len(podList.Items)).To(Equal(1))
				// Should not recreate the pod
				g.Expect(podNameToUID[podList.Items[0].Name]).To(Equal(podList.Items[0].UID))
			}).WithTimeout(createClusterTimeout).WithPolling(createClusterPolling).Should(Succeed())
		})

		It("should be able to increase the volume size without restarting", func() {
			pdg := data.NewPDGroup(ns.Name, "pdg", tc.Name, ptr.To(int32(1)), nil)
			kvg := data.NewTiKVGroup(ns.Name, "kvg", tc.Name, ptr.To(int32(1)), func(kvg *v1alpha1.TiKVGroup) {
				for i := range kvg.Spec.Template.Spec.Volumes {
					vol := &kvg.Spec.Template.Spec.Volumes[i]
					if vol.Name == "data" {
						vol.Storage = data.StorageSizeGi2quantity(2)
						break
					}
				}
			})
			dbg := data.NewTiDBGroup(ns.Name, "dbg", tc.Name, ptr.To(int32(1)), nil)
			Expect(k8sClient.Create(ctx, pdg)).To(Succeed())
			Expect(k8sClient.Create(ctx, kvg)).To(Succeed())
			Expect(k8sClient.Create(ctx, dbg)).To(Succeed())

			By("Waiting for the cluster to be ready")
			Eventually(func(g Gomega) {
				_, ready := utiltidb.IsClusterReady(k8sClient, tc.Name, tc.Namespace)
				g.Expect(ready).To(BeTrue())
				g.Expect(utiltidb.AreAllInstancesReady(k8sClient, pdg,
					[]*v1alpha1.TiKVGroup{kvg}, []*v1alpha1.TiDBGroup{dbg}, nil, nil)).To(Succeed())
			}).WithTimeout(createClusterTimeout).WithPolling(createClusterPolling).Should(Succeed())

			scList, err := clientSet.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
			Expect(err).To(BeNil())
			for _, sc := range scList.Items {
				if sc.Annotations["storageclass.kubernetes.io/is-default-class"] == "true" &&
					(sc.AllowVolumeExpansion == nil || !*sc.AllowVolumeExpansion) {
					Skip("default storage class does not support volume expansion")
				}
			}

			By("Checking the pvc size")
			listOpts := metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=%s,%s=%s",
					v1alpha1.LabelKeyCluster, tc.Name, v1alpha1.LabelKeyGroup, kvg.Name),
			}
			pvcList, err := clientSet.CoreV1().PersistentVolumeClaims(tc.Namespace).List(ctx, listOpts)
			Expect(err).To(BeNil())
			Expect(len(pvcList.Items)).To(Equal(1))
			Expect(pvcList.Items[0].Status.Capacity.Storage().Equal(data.StorageSizeGi2quantity(2))).To(BeTrue())

			By("Recording the pod's UID")
			podList, err := clientSet.CoreV1().Pods(tc.Namespace).List(ctx, listOpts)
			Expect(err).To(BeNil())
			Expect(len(podList.Items)).To(Equal(1))
			originalPodName, originalPodUID := podList.Items[0].Name, podList.Items[0].UID

			By("Increasing the tikv's volume size from 2Gi to 5Gi")
			var kvgGet v1alpha1.TiKVGroup
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: tc.Namespace, Name: kvg.Name}, &kvgGet)).To(Succeed())
			Expect(len(kvgGet.Spec.Template.Spec.Volumes)).To(Equal(1))
			kvgGet.Spec.Template.Spec.Volumes[0].Storage = data.StorageSizeGi2quantity(5)
			Expect(k8sClient.Update(ctx, &kvgGet)).To(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(utiltidb.AreAllInstancesReady(k8sClient, pdg,
					[]*v1alpha1.TiKVGroup{kvg}, []*v1alpha1.TiDBGroup{dbg}, nil, nil)).To(Succeed())
				podList, err = clientSet.CoreV1().Pods(tc.Namespace).List(ctx, listOpts)
				g.Expect(err).To(BeNil())
				g.Expect(len(podList.Items)).To(Equal(1))
				g.Expect(podList.Items[0].Name).To(Equal(originalPodName))
				g.Expect(podList.Items[0].UID).To(Equal(originalPodUID))

				pvcList, err := clientSet.CoreV1().PersistentVolumeClaims(tc.Namespace).List(ctx, listOpts)
				Expect(err).To(BeNil())
				Expect(len(pvcList.Items)).To(Equal(1))
				Expect(pvcList.Items[0].Status.Capacity.Storage()).To(Equal(data.StorageSizeGi2quantity(5)))
			}).WithTimeout(createClusterTimeout).WithPolling(createClusterPolling).Should(Succeed())
		})

		It("should not recreate pods for labels/annotations modification", func() {
			pdg := data.NewPDGroup(ns.Name, "pdg", tc.Name, ptr.To(int32(1)), nil)
			kvg := data.NewTiKVGroup(ns.Name, "kvg", tc.Name, ptr.To(int32(1)), nil)
			dbg := data.NewTiDBGroup(ns.Name, "dbg", tc.Name, ptr.To(int32(1)), nil)
			Expect(k8sClient.Create(ctx, pdg)).To(Succeed())
			Expect(k8sClient.Create(ctx, kvg)).To(Succeed())
			Expect(k8sClient.Create(ctx, dbg)).To(Succeed())

			By("Waiting for the cluster to be ready")
			Eventually(func(g Gomega) {
				_, ready := utiltidb.IsClusterReady(k8sClient, tc.Name, tc.Namespace)
				g.Expect(ready).To(BeTrue())
				g.Expect(utiltidb.AreAllInstancesReady(k8sClient, pdg,
					[]*v1alpha1.TiKVGroup{kvg}, []*v1alpha1.TiDBGroup{dbg}, nil, nil)).To(Succeed())
			}).WithTimeout(createClusterTimeout).WithPolling(createClusterPolling).Should(Succeed())

			By("Recording the pod's UID")
			listOpts := metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=%s,%s=%s",
					v1alpha1.LabelKeyCluster, tc.Name, v1alpha1.LabelKeyGroup, dbg.Name),
			}
			podList, err := clientSet.CoreV1().Pods(tc.Namespace).List(ctx, listOpts)
			Expect(err).To(BeNil())
			Expect(len(podList.Items)).To(Equal(1))
			originalPodName, originalPodUID := podList.Items[0].Name, podList.Items[0].UID

			By("Changing labels and annotations")
			var dbgGet v1alpha1.TiDBGroup
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: tc.Namespace, Name: dbg.Name}, &dbgGet)).To(Succeed())
			if dbgGet.Spec.Template.Labels == nil {
				dbgGet.Spec.Template.Labels = map[string]string{}
			}
			dbgGet.Spec.Template.Labels["foo"] = "bar"
			if dbgGet.Spec.Template.Annotations == nil {
				dbgGet.Spec.Template.Annotations = map[string]string{}
			}
			dbgGet.Spec.Template.Annotations["foo"] = "bar"
			Expect(k8sClient.Update(ctx, &dbgGet)).To(Succeed())

			By("Checking the pods are not recreated")
			Eventually(func(g Gomega) {
				var pod corev1.Pod
				g.Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: tc.Namespace, Name: originalPodName}, &pod)).To(Succeed())
				g.Expect(pod.UID).To(Equal(originalPodUID))
				g.Expect(pod.Labels["foo"]).To(Equal("bar"))
				g.Expect(pod.Annotations["foo"]).To(Equal("bar"))
				g.Expect(pod.Status.Phase).To(Equal(corev1.PodRunning))
			}).WithTimeout(createClusterTimeout).WithPolling(createClusterPolling).Should(Succeed())
		})
	})

	Context("Rolling Update", func() {
		It("should NOT perform a rolling update", func() {
			pdg := data.NewPDGroup(ns.Name, "pdg", tc.Name, ptr.To(int32(1)), nil)
			kvg := data.NewTiKVGroup(ns.Name, "kvg", tc.Name, ptr.To(int32(3)), func(tk *v1alpha1.TiKVGroup) {
				tk.Spec.Template.Spec.UpdateStrategy.Config = v1alpha1.ConfigUpdateStrategyHotReload
			})
			dbg := data.NewTiDBGroup(ns.Name, "dbg", tc.Name, ptr.To(int32(1)), nil)
			Expect(k8sClient.Create(ctx, pdg)).To(Succeed())
			Expect(k8sClient.Create(ctx, kvg)).To(Succeed())
			Expect(k8sClient.Create(ctx, dbg)).To(Succeed())

			By("Waiting for the cluster to be ready")
			Eventually(func(g Gomega) {
				_, ready := utiltidb.IsClusterReady(k8sClient, tc.Name, tc.Namespace)
				g.Expect(ready).To(BeTrue())
				g.Expect(utiltidb.AreAllInstancesReady(k8sClient, pdg,
					[]*v1alpha1.TiKVGroup{kvg}, []*v1alpha1.TiDBGroup{dbg}, nil, nil)).To(Succeed())
			}).WithTimeout(createClusterTimeout).WithPolling(createClusterPolling).Should(Succeed())

			By("Checking the initial config")
			listOpts := metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=%s,%s=%s",
					v1alpha1.LabelKeyCluster, tc.Name, v1alpha1.LabelKeyGroup, kvg.Name),
			}
			assertConfigMaps := func(expectedLogLevel string) {
				cms, err := clientSet.CoreV1().ConfigMaps(tc.Namespace).List(ctx, listOpts)
				Expect(err).NotTo(HaveOccurred())
				Expect(cms.Items).To(HaveLen(3))
				for _, cm := range cms.Items {
					Expect(cm.Data).To(ContainElement(ContainSubstring("level = '%s'", expectedLogLevel)))
				}
			}

			By("Recording tikv pods' UID before changes")
			podMap, err := k8s.GetPodUIDMap(ctx, clientSet, tc.Namespace, listOpts)
			Expect(err).NotTo(HaveOccurred())
			Expect(podMap).To(HaveLen(3))

			testUpdateWithoutRolling := func(updateName string, modifySpec func(*v1alpha1.TiKVGroup)) {
				By(fmt.Sprintf("Applying %s change", updateName))
				var kvgGet v1alpha1.TiKVGroup
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(kvg), &kvgGet)).To(Succeed())
				modifySpec(&kvgGet)
				Expect(k8sClient.Update(ctx, &kvgGet)).To(Succeed())

				By(fmt.Sprintf("Verifying no rolling update after %s change", updateName))
				Consistently(func(g Gomega) {
					_, ready := utiltidb.IsClusterReady(k8sClient, tc.Name, tc.Namespace)
					g.Expect(ready).To(BeTrue())
					g.Expect(utiltidb.AreAllInstancesReady(k8sClient, pdg,
						[]*v1alpha1.TiKVGroup{kvg}, []*v1alpha1.TiDBGroup{dbg}, nil, nil)).To(Succeed())

					// Ensure pods are not re-created
					currentPodMap, err := k8s.GetPodUIDMap(ctx, clientSet, tc.Namespace, listOpts)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(currentPodMap).To(Equal(podMap))
				}).WithTimeout(3 * time.Minute).WithPolling(createClusterPolling).Should(Succeed())
			}

			testUpdateWithoutRolling("config", func(tk *v1alpha1.TiKVGroup) {
				tk.Spec.Template.Spec.Config = logLevelConfig
			})
			assertConfigMaps("warn")

			testUpdateWithoutRolling("annotation", func(tk *v1alpha1.TiKVGroup) {
				if tk.Spec.Template.Annotations == nil {
					tk.Spec.Template.Annotations = make(map[string]string)
				}
				tk.Spec.Template.Annotations["foo"] = "bar"
			})
		})

		It("pd/tikv/tiflash/ticdc: should perform a rolling update for config change", func() {
			pdg := data.NewPDGroup(ns.Name, "pd", tc.Name, ptr.To(int32(3)), nil)
			kvg := data.NewTiKVGroup(ns.Name, "tikv", tc.Name, ptr.To(int32(3)), nil)
			dbg := data.NewTiDBGroup(ns.Name, "tidb", tc.Name, ptr.To(int32(1)), nil)
			flashg := data.NewTiFlashGroup(ns.Name, "flash", tc.Name, ptr.To(int32(3)), nil)
			cdcg := data.NewTiCDCGroup(ns.Name, "cdc", tc.Name, ptr.To(int32(3)), nil)
			Expect(k8sClient.Create(ctx, pdg)).To(Succeed())
			Expect(k8sClient.Create(ctx, kvg)).To(Succeed())
			Expect(k8sClient.Create(ctx, dbg)).To(Succeed())
			Expect(k8sClient.Create(ctx, flashg)).To(Succeed())
			Expect(k8sClient.Create(ctx, cdcg)).To(Succeed())

			By("Waiting for the cluster to be ready")
			Eventually(func(g Gomega) {
				_, ready := utiltidb.IsClusterReady(k8sClient, tc.Name, tc.Namespace)
				g.Expect(ready).To(BeTrue())
				g.Expect(utiltidb.AreAllInstancesReady(k8sClient, pdg,
					[]*v1alpha1.TiKVGroup{kvg}, []*v1alpha1.TiDBGroup{dbg},
					[]*v1alpha1.TiFlashGroup{flashg}, []*v1alpha1.TiCDCGroup{cdcg})).To(Succeed())
			}).WithTimeout(createClusterTimeout).WithPolling(createClusterPolling).Should(Succeed())

			groupNames := []string{pdg.Name, kvg.Name, flashg.Name, cdcg.Name}
			outerCtx, cancel := context.WithCancel(ctx)
			defer cancel()
			for _, groupName := range groupNames {
				By("Checking the logic of rolling update for " + groupName)
				listOpts := metav1.ListOptions{
					LabelSelector: fmt.Sprintf("%s=%s,%s=%s",
						v1alpha1.LabelKeyCluster, tc.Name, v1alpha1.LabelKeyGroup, groupName),
				}

				By("collecting the events of pods for verifying rolling update")
				watchCtx, cancel := context.WithCancel(outerCtx)
				podWatcher, err := clientSet.CoreV1().Pods(tc.Namespace).Watch(watchCtx, listOpts)
				Expect(err).NotTo(HaveOccurred())
				type podInfo struct {
					name         string
					uid          string
					creationTime metav1.Time
					deletionTime metav1.Time
				}

				podMap := map[string]podInfo{}
				wg := sync.WaitGroup{}
				wg.Add(1)

				go func() {
					defer GinkgoRecover()
					for {
						select {
						case <-watchCtx.Done():
							GinkgoWriter.Println("podWatcher is stopped")
							podWatcher.Stop()
							wg.Done()
							return

						case event := <-podWatcher.ResultChan():
							pod, isPod := event.Object.(*corev1.Pod)
							if !isPod {
								continue
							}
							info, ok := podMap[string(pod.UID)]
							if !ok {
								info = podInfo{
									name:         pod.Name,
									uid:          string(pod.UID),
									creationTime: pod.CreationTimestamp,
								}
							}
							if !pod.DeletionTimestamp.IsZero() && pod.DeletionGracePeriodSeconds != nil && *pod.DeletionGracePeriodSeconds == 0 {
								info.deletionTime = *pod.DeletionTimestamp
							}
							podMap[string(pod.UID)] = info
						}
					}
				}()

				By("Changing the spec")
				cfg := "log.level = 'debug'"
				var updateTime time.Time
				switch groupName {
				case "pd":
					var pdgGet v1alpha1.PDGroup
					Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: tc.Namespace, Name: groupName}, &pdgGet)).To(Succeed())
					pdgGet.Spec.Template.Spec.Config = v1alpha1.ConfigFile(cfg)
					updateTime = time.Now()
					Expect(k8sClient.Update(ctx, &pdgGet)).To(Succeed())
				case "tikv":
					var kvgGet v1alpha1.TiKVGroup
					Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: tc.Namespace, Name: groupName}, &kvgGet)).To(Succeed())
					kvgGet.Spec.Template.Spec.Config = v1alpha1.ConfigFile(cfg)
					updateTime = time.Now()
					Expect(k8sClient.Update(ctx, &kvgGet)).To(Succeed())
				case "flash":
					cfg = "logger.level = 'debug'"
					var flashgGet v1alpha1.TiFlashGroup
					Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: tc.Namespace, Name: groupName}, &flashgGet)).To(Succeed())
					flashgGet.Spec.Template.Spec.Config = v1alpha1.ConfigFile(cfg)
					updateTime = time.Now()
					Expect(k8sClient.Update(ctx, &flashgGet)).To(Succeed())
				case "cdc":
					cfg = "log-level = 'debug'"
					var cdcgGet v1alpha1.TiCDCGroup
					Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: tc.Namespace, Name: groupName}, &cdcgGet)).To(Succeed())
					cdcgGet.Spec.Template.Spec.Config = v1alpha1.ConfigFile(cfg)
					updateTime = time.Now()
					Expect(k8sClient.Update(ctx, &cdcgGet)).To(Succeed())
				}

				Eventually(func(g Gomega) {
					_, ready := utiltidb.IsClusterReady(k8sClient, tc.Name, tc.Namespace)
					g.Expect(ready).To(BeTrue())
					g.Expect(utiltidb.AreAllInstancesReady(k8sClient, pdg,
						[]*v1alpha1.TiKVGroup{kvg}, []*v1alpha1.TiDBGroup{dbg},
						[]*v1alpha1.TiFlashGroup{flashg}, []*v1alpha1.TiCDCGroup{cdcg})).To(Succeed())

					podList, err := clientSet.CoreV1().Pods(tc.Namespace).List(ctx, listOpts)
					g.Expect(err).To(BeNil())
					g.Expect(len(podList.Items)).To(Equal(3))
					for _, pod := range podList.Items {
						// Ensure pods are re-created
						g.Expect(pod.Status.StartTime).ShouldNot(BeNil())
						g.Expect(pod.Status.StartTime.After(updateTime)).To(BeTrue())
					}
					// g.Expect(k8s.CheckRollingRestartLogic(eventSlice)).To(BeTrue())
				}).WithTimeout(createClusterTimeout).WithPolling(createClusterPolling).Should(Succeed())
				cancel()

				wg.Wait()

				infos := []podInfo{}
				for _, v := range podMap {
					infos = append(infos, v)
				}
				slices.SortFunc(infos, func(a podInfo, b podInfo) int {
					if a.deletionTime.IsZero() && b.deletionTime.IsZero() {
						return a.creationTime.Compare(b.creationTime.Time)
					}
					if a.deletionTime.IsZero() {
						return a.creationTime.Compare(b.deletionTime.Time)
					}
					if b.deletionTime.IsZero() {
						return a.deletionTime.Compare(b.creationTime.Time)
					}
					return a.deletionTime.Compare(b.deletionTime.Time)
				})
				for _, info := range infos {
					if info.deletionTime.IsZero() {
						GinkgoWriter.Printf("%v(%v) created at %s\n", info.name, info.uid, info.creationTime)
					} else {
						GinkgoWriter.Printf("%v(%v) created at %s, deleted at %s\n", info.name, info.uid, info.creationTime, info.deletionTime)
					}
				}
				Expect(len(infos)).To(Equal(6))
				Expect(infos[0].name).To(Equal(infos[1].name))
				Expect(infos[2].name).To(Equal(infos[3].name))
				Expect(infos[4].name).To(Equal(infos[5].name))
			}
		})

		It("should perform a rolling update when the restart annotation is added", func() {
			pdg := data.NewPDGroup(ns.Name, "pdg", tc.Name, ptr.To(int32(1)), nil)
			kvg := data.NewTiKVGroup(ns.Name, "kvg", tc.Name, ptr.To(int32(3)), nil)
			dbg := data.NewTiDBGroup(ns.Name, "dbg", tc.Name, ptr.To(int32(1)), nil)
			Expect(k8sClient.Create(ctx, pdg)).To(Succeed())
			Expect(k8sClient.Create(ctx, kvg)).To(Succeed())
			Expect(k8sClient.Create(ctx, dbg)).To(Succeed())

			By("Waiting for the cluster to be ready")
			Eventually(func(g Gomega) {
				_, ready := utiltidb.IsClusterReady(k8sClient, tc.Name, tc.Namespace)
				g.Expect(ready).To(BeTrue())
				g.Expect(utiltidb.AreAllInstancesReady(k8sClient, pdg,
					[]*v1alpha1.TiKVGroup{kvg}, []*v1alpha1.TiDBGroup{dbg}, nil, nil)).To(Succeed())
			}).WithTimeout(createClusterTimeout).WithPolling(createClusterPolling).Should(Succeed())

			By("collecting the events of pods for verifying rolling update")
			listOpts := metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=%s,%s=%s",
					v1alpha1.LabelKeyCluster, tc.Name, v1alpha1.LabelKeyGroup, kvg.Name),
			}
			watchCtx, cancel := context.WithCancel(ctx)
			podWatcher, err := clientSet.CoreV1().Pods(tc.Namespace).Watch(watchCtx, listOpts)
			Expect(err).NotTo(HaveOccurred())
			type podInfo struct {
				name         string
				uid          string
				creationTime metav1.Time
				deletionTime metav1.Time
			}

			podMap := map[string]podInfo{}
			wg := sync.WaitGroup{}
			wg.Add(1)

			go func() {
				defer GinkgoRecover()
				for {
					select {
					case <-watchCtx.Done():
						GinkgoWriter.Println("podWatcher is stopped")
						podWatcher.Stop()
						wg.Done()
						return

					case event := <-podWatcher.ResultChan():
						pod, isPod := event.Object.(*corev1.Pod)
						if !isPod {
							continue
						}
						info, ok := podMap[string(pod.UID)]
						if !ok {
							info = podInfo{
								name:         pod.Name,
								uid:          string(pod.UID),
								creationTime: pod.CreationTimestamp,
							}
						}
						if !pod.DeletionTimestamp.IsZero() && pod.DeletionGracePeriodSeconds != nil && *pod.DeletionGracePeriodSeconds == 0 {
							info.deletionTime = *pod.DeletionTimestamp
						}
						podMap[string(pod.UID)] = info
					}
				}
			}()

			By("Add the restart annotation to TiKVGroup")
			var updateTime time.Time
			var kvgGet v1alpha1.TiKVGroup
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: tc.Namespace, Name: kvg.Name}, &kvgGet)).To(Succeed())
			if kvgGet.Spec.Template.Annotations == nil {
				kvgGet.Spec.Template.Annotations = map[string]string{}
			}
			kvgGet.Spec.Template.Annotations[metav1alpha1.RestartAnnotationKey] = "test"
			updateTime = time.Now()
			Expect(k8sClient.Update(ctx, &kvgGet)).To(Succeed())

			Eventually(func(g Gomega) {
				_, ready := utiltidb.IsClusterReady(k8sClient, tc.Name, tc.Namespace)
				g.Expect(ready).To(BeTrue())
				g.Expect(utiltidb.AreAllInstancesReady(k8sClient, pdg,
					[]*v1alpha1.TiKVGroup{kvg}, []*v1alpha1.TiDBGroup{dbg}, nil, nil)).To(Succeed())

				podList, err := clientSet.CoreV1().Pods(tc.Namespace).List(ctx, listOpts)
				g.Expect(err).To(BeNil())
				g.Expect(len(podList.Items)).To(Equal(3))
				for _, pod := range podList.Items {
					// Ensure pods are re-created
					g.Expect(pod.Status.StartTime).ShouldNot(BeNil())
					g.Expect(pod.Status.StartTime.After(updateTime)).To(BeTrue())
				}
			}).WithTimeout(createClusterTimeout).WithPolling(createClusterPolling).Should(Succeed())
			cancel()
			wg.Wait()

			var infos []podInfo
			for _, v := range podMap {
				infos = append(infos, v)
			}
			slices.SortFunc(infos, func(a podInfo, b podInfo) int {
				if a.deletionTime.IsZero() && b.deletionTime.IsZero() {
					return a.creationTime.Compare(b.creationTime.Time)
				}
				if a.deletionTime.IsZero() {
					return a.creationTime.Compare(b.deletionTime.Time)
				}
				if b.deletionTime.IsZero() {
					return a.deletionTime.Compare(b.creationTime.Time)
				}
				return a.deletionTime.Compare(b.deletionTime.Time)
			})
			for _, info := range infos {
				if info.deletionTime.IsZero() {
					GinkgoWriter.Printf("%v(%v) created at %s\n", info.name, info.uid, info.creationTime)
				} else {
					GinkgoWriter.Printf("%v(%v) created at %s, deleted at %s\n", info.name, info.uid, info.creationTime, info.deletionTime)
				}
			}
			Expect(len(infos)).To(Equal(6))
			Expect(infos[0].name).To(Equal(infos[1].name))
			Expect(infos[2].name).To(Equal(infos[3].name))
			Expect(infos[4].name).To(Equal(infos[5].name))

			By("Update the restart annotation to TiKVGroup")
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: tc.Namespace, Name: kvg.Name}, &kvgGet)).To(Succeed())
			if kvgGet.Spec.Template.Annotations == nil {
				kvgGet.Spec.Template.Annotations = map[string]string{}
			}
			kvgGet.Spec.Template.Annotations[metav1alpha1.RestartAnnotationKey] = "test-again"
			updateTime = time.Now()
			Expect(k8sClient.Update(ctx, &kvgGet)).To(Succeed())
			Eventually(func(g Gomega) {
				_, ready := utiltidb.IsClusterReady(k8sClient, tc.Name, tc.Namespace)
				g.Expect(ready).To(BeTrue())
				g.Expect(utiltidb.AreAllInstancesReady(k8sClient, pdg,
					[]*v1alpha1.TiKVGroup{kvg}, []*v1alpha1.TiDBGroup{dbg}, nil, nil)).To(Succeed())

				podList, err := clientSet.CoreV1().Pods(tc.Namespace).List(ctx, listOpts)
				g.Expect(err).To(BeNil())
				g.Expect(len(podList.Items)).To(Equal(3))
				for _, pod := range podList.Items {
					// Ensure pods are re-created
					g.Expect(pod.Status.StartTime).ShouldNot(BeNil())
					g.Expect(pod.Status.StartTime.After(updateTime)).To(BeTrue())
				}
			}).WithTimeout(createClusterTimeout).WithPolling(createClusterPolling).Should(Succeed())
		})
	})

	Context("Version Upgrade", func() {
		When("use the default policy", func() {
			It("should wait for pd upgrade to complete before upgrading tikv", func() {
				pdg := data.NewPDGroup(ns.Name, "pdg", tc.Name, ptr.To(int32(3)), nil)
				kvg := data.NewTiKVGroup(ns.Name, "kvg", tc.Name, ptr.To(int32(3)), nil)
				dbg := data.NewTiDBGroup(ns.Name, "dbg", tc.Name, ptr.To(int32(1)), nil)
				Expect(k8sClient.Create(ctx, pdg)).To(Succeed())
				Expect(k8sClient.Create(ctx, kvg)).To(Succeed())
				Expect(k8sClient.Create(ctx, dbg)).To(Succeed())

				By("Waiting for the cluster to be ready")
				Eventually(func(g Gomega) {
					_, ready := utiltidb.IsClusterReady(k8sClient, tc.Name, tc.Namespace)
					g.Expect(ready).To(BeTrue())

					g.Expect(utiltidb.AreAllInstancesReady(k8sClient, pdg,
						[]*v1alpha1.TiKVGroup{kvg}, []*v1alpha1.TiDBGroup{dbg}, []*v1alpha1.TiFlashGroup{}, nil)).To(Succeed())
				}).WithTimeout(createClusterTimeout).WithPolling(createClusterPolling).Should(Succeed())

				By("Checking the version of tikv group")
				var kvgGet v1alpha1.TiKVGroup
				Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: tc.Namespace, Name: kvg.Name}, &kvgGet)).To(Succeed())
				oldVersion := kvgGet.Spec.Template.Spec.Version
				Expect(oldVersion).NotTo(BeEmpty())
				Expect(kvgGet.Status.Version).To(Equal(oldVersion))
				v, err := semver.NewVersion(oldVersion)
				Expect(err).To(BeNil())
				newVersion := "v" + v.IncMinor().String()

				By(fmt.Sprintf("Updating the version of the tikv group from %s to %s", oldVersion, newVersion))
				kvgGet.Spec.Template.Spec.Version = newVersion
				Expect(k8sClient.Update(ctx, &kvgGet)).To(Succeed())

				Consistently(func(g Gomega) {
					_, ready := utiltidb.IsClusterReady(k8sClient, tc.Name, tc.Namespace)
					g.Expect(ready).To(BeTrue())
					g.Expect(utiltidb.AreAllInstancesReady(k8sClient, pdg,
						[]*v1alpha1.TiKVGroup{kvg}, []*v1alpha1.TiDBGroup{dbg}, []*v1alpha1.TiFlashGroup{}, nil)).To(Succeed())

					// tikv should not be upgraded
					g.Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: tc.Namespace, Name: kvg.Name}, &kvgGet)).To(Succeed())
					g.Expect(kvgGet.Spec.Template.Spec.Version).To(Equal(newVersion))
					g.Expect(kvgGet.Status.Version).To(Equal(oldVersion))
				}).WithTimeout(1 * time.Minute).WithPolling(createClusterPolling).Should(Succeed())

				By("Pause the reconciliation")
				var tcGet v1alpha1.Cluster
				Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: tc.Namespace, Name: tc.Name}, &tcGet)).To(Succeed())
				tcGet.Spec.Paused = true
				Expect(k8sClient.Update(ctx, &tcGet)).To(Succeed())

				By("Upgrading the pd group")
				var pdgGet v1alpha1.PDGroup
				Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: tc.Namespace, Name: pdg.Name}, &pdgGet)).To(Succeed())
				Expect(pdgGet.Spec.Template.Spec.Version).To(Equal(oldVersion))
				Expect(pdgGet.Status.Version).To(Equal(oldVersion))
				pdgGet.Spec.Template.Spec.Version = newVersion
				Expect(k8sClient.Update(ctx, &pdgGet)).To(Succeed())

				By("Checking the version of pd group when paused")
				Consistently(func(g Gomega) {
					g.Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: tc.Namespace, Name: pdg.Name}, &pdgGet)).To(Succeed())
					g.Expect(pdgGet.Spec.Template.Spec.Version).To(Equal(newVersion))
					g.Expect(pdgGet.Status.Version).To(Equal(oldVersion))
				}).WithTimeout(1 * time.Minute).WithPolling(createClusterPolling).Should(Succeed())

				By("Resume the reconciliation")
				tcGet.Spec.Paused = false
				Expect(k8sClient.Update(ctx, &tcGet)).To(Succeed())

				By("Checking the version of pd and tikv group after resuming")
				Eventually(func(g Gomega) {
					_, ready := utiltidb.IsClusterReady(k8sClient, tc.Name, tc.Namespace)
					g.Expect(ready).To(BeTrue())
					g.Expect(utiltidb.AreAllInstancesReady(k8sClient, pdg,
						[]*v1alpha1.TiKVGroup{kvg}, []*v1alpha1.TiDBGroup{dbg}, []*v1alpha1.TiFlashGroup{}, nil)).To(Succeed())

					// pd and tikv should be upgraded
					g.Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: tc.Namespace, Name: pdg.Name}, &pdgGet)).To(Succeed())
					g.Expect(pdgGet.Spec.Template.Spec.Version).To(Equal(newVersion))
					g.Expect(pdgGet.Status.Version).To(Equal(newVersion))
					var pdPods corev1.PodList
					g.Expect(k8sClient.List(ctx, &pdPods, client.InNamespace(tc.Namespace),
						client.MatchingLabels{v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD}))
					for _, pod := range pdPods.Items {
						for _, container := range pod.Spec.Containers {
							if container.Name == v1alpha1.ContainerNamePD {
								g.Expect(strings.Contains(container.Image, newVersion)).To(BeTrue())
							}
						}
					}

					g.Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: tc.Namespace, Name: kvg.Name}, &kvgGet)).To(Succeed())
					g.Expect(kvgGet.Spec.Template.Spec.Version).To(Equal(newVersion))
					g.Expect(kvgGet.Status.Version).To(Equal(newVersion))
					g.Expect(k8sClient.List(ctx, &pdPods, client.InNamespace(tc.Namespace),
						client.MatchingLabels{v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiKV}))
					for _, pod := range pdPods.Items {
						for _, container := range pod.Spec.Containers {
							if container.Name == v1alpha1.ContainerNameTiKV {
								g.Expect(strings.Contains(container.Image, newVersion)).To(BeTrue())
							}
						}
					}
				}).WithTimeout(createClusterTimeout).WithPolling(createClusterPolling).Should(Succeed())
			})
		})
	})

	Context("TLS", func() {
		It("should enable TLS for MySQL Client and between TiDB components", func() {
			By("Installing the certificates")
			Expect(InstallTiDBIssuer(ctx, yamlApplier, ns.Name, tc.Name)).To(Succeed())
			Expect(InstallTiDBCertificates(ctx, yamlApplier, ns.Name, tc.Name, "dbg")).To(Succeed())
			Expect(InstallTiDBComponentsCertificates(ctx, yamlApplier, ns.Name, tc.Name, "pdg", "kvg", "dbg", "flashg", "cdcg")).To(Succeed())

			By("Enabling TLS for TiDB components")
			var tcGet v1alpha1.Cluster
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: tc.Namespace, Name: tc.Name}, &tcGet)).To(Succeed())
			tcGet.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
			Expect(k8sClient.Update(ctx, &tcGet)).To(Succeed())

			By("Creating the components with TLS client enabled")
			pdg := data.NewPDGroup(ns.Name, "pdg", tc.Name, ptr.To(int32(1)), func(_ *v1alpha1.PDGroup) {})
			kvg := data.NewTiKVGroup(ns.Name, "kvg", tc.Name, ptr.To(int32(1)), func(_ *v1alpha1.TiKVGroup) {})
			dbg := data.NewTiDBGroup(ns.Name, "dbg", tc.Name, ptr.To(int32(1)), func(group *v1alpha1.TiDBGroup) {
				group.Spec.Template.Spec.Security = &v1alpha1.TiDBSecurity{
					TLS: &v1alpha1.TiDBTLS{
						MySQL: &v1alpha1.TLS{
							Enabled: true,
						},
					},
				}
			})
			flashg := data.NewTiFlashGroup(ns.Name, "flashg", tc.Name, ptr.To(int32(1)), nil)
			cdcg := data.NewTiCDCGroup(ns.Name, "cdcg", tc.Name, ptr.To(int32(1)), nil)
			Expect(k8sClient.Create(ctx, pdg)).To(Succeed())
			Expect(k8sClient.Create(ctx, kvg)).To(Succeed())
			Expect(k8sClient.Create(ctx, dbg)).To(Succeed())
			Expect(k8sClient.Create(ctx, flashg)).To(Succeed())
			Expect(k8sClient.Create(ctx, cdcg)).To(Succeed())

			By("Checking the status of the cluster and the connection to the TiDB service")
			Eventually(func(g Gomega) {
				tcGet, ready := utiltidb.IsClusterReady(k8sClient, tc.Name, tc.Namespace)
				g.Expect(ready).To(BeTrue())
				g.Expect(len(tcGet.Status.PD)).NotTo(BeZero())
				for _, compStatus := range tcGet.Status.Components {
					switch compStatus.Kind {
					case v1alpha1.ComponentKindPD, v1alpha1.ComponentKindTiKV, v1alpha1.ComponentKindTiDB,
						v1alpha1.ComponentKindTiFlash, v1alpha1.ComponentKindTiCDC:
						g.Expect(compStatus.Replicas).To(Equal(int32(1)))
					default:
						g.Expect(compStatus.Replicas).To(BeZero())
					}
				}

				checkComponent := func(groupName, componentName string, expectedReplicas *int32) {
					listOptions := metav1.ListOptions{
						LabelSelector: fmt.Sprintf("%s=%s,%s=%s",
							v1alpha1.LabelKeyCluster, tc.Name, v1alpha1.LabelKeyGroup, groupName),
					}
					podList, err := clientSet.CoreV1().Pods(tc.Namespace).List(ctx, listOptions)
					g.Expect(err).To(BeNil())
					g.Expect(len(podList.Items)).To(Equal(int(*expectedReplicas)))
					for _, pod := range podList.Items {
						g.Expect(pod.Status.Phase).To(Equal(corev1.PodRunning))

						// check for mTLS
						g.Expect(pod.Spec.Volumes).To(ContainElement(corev1.Volume{
							Name: v1alpha1.VolumeNameClusterTLS,
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									// TODO(liubo02): extract to a namer pkg
									SecretName:  groupName + "-" + componentName + "-cluster-secret",
									DefaultMode: ptr.To(int32(420)),
								},
							},
						}))
						g.Expect(pod.Spec.Containers[0].VolumeMounts).To(ContainElement(corev1.VolumeMount{
							Name:      v1alpha1.VolumeNameClusterTLS,
							MountPath: fmt.Sprintf("/var/lib/%s-tls", componentName),
							ReadOnly:  true,
						}))

						switch componentName {
						case v1alpha1.LabelValComponentTiDB:
							// check for TiDB server & mysql client TLS
							g.Expect(pod.Spec.Volumes).To(ContainElement(corev1.Volume{
								Name: v1alpha1.VolumeNameMySQLTLS,
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										// TODO(liubo02): extract to a namer pkg
										SecretName:  dbg.Name + "-tidb-server-secret",
										DefaultMode: ptr.To(int32(420)),
									},
								},
							}))
							g.Expect(pod.Spec.Containers[0].VolumeMounts).To(ContainElement(corev1.VolumeMount{
								Name:      v1alpha1.VolumeNameMySQLTLS,
								MountPath: v1alpha1.DirPathMySQLTLS,
								ReadOnly:  true,
							}))
						}
					}
				}
				checkComponent(pdg.Name, v1alpha1.LabelValComponentPD, pdg.Spec.Replicas)
				checkComponent(kvg.Name, v1alpha1.LabelValComponentTiKV, kvg.Spec.Replicas)
				checkComponent(dbg.Name, v1alpha1.LabelValComponentTiDB, dbg.Spec.Replicas)
				checkComponent(flashg.Name, v1alpha1.LabelValComponentTiFlash, flashg.Spec.Replicas)
				checkComponent(cdcg.Name, v1alpha1.LabelValComponentTiCDC, cdcg.Spec.Replicas)

				// TODO(liubo02): extract to a common namer pkg
				g.Expect(utiltidb.IsTiDBConnectable(ctx, k8sClient, fw,
					tc.Namespace, tc.Name, dbg.Name, "root", "", dbg.Name+"-tidb-client-secret")).To(Succeed())
			}).WithTimeout(createClusterTimeout).WithPolling(createClusterPolling).Should(Succeed())

			// TODO: version upgrade test
		})
	})

	Context("TiDB Feature", func() {
		It("should set store labels for TiKV and TiFlash, set server lables for TiDB", func() {
			By("Creating the components with location labels")
			pdg := data.NewPDGroup(ns.Name, "pdg", tc.Name, ptr.To(int32(1)), func(group *v1alpha1.PDGroup) {
				group.Spec.Template.Spec.Config = `[replication]
location-labels = ["region", "zone", "host"]`
			})
			kvg := data.NewTiKVGroup(ns.Name, "kvg", tc.Name, ptr.To(int32(1)), nil)
			dbg := data.NewTiDBGroup(ns.Name, "dbg", tc.Name, ptr.To(int32(1)), nil)
			flashg := data.NewTiFlashGroup(ns.Name, "flashg", tc.Name, ptr.To(int32(1)), nil)
			Expect(k8sClient.Create(ctx, pdg)).To(Succeed())
			Expect(k8sClient.Create(ctx, kvg)).To(Succeed())
			Expect(k8sClient.Create(ctx, dbg)).To(Succeed())
			Expect(k8sClient.Create(ctx, flashg)).To(Succeed())

			By("Checking the status of the cluster and the connection to the TiDB service")
			Eventually(func(g Gomega) {
				tcGet, ready := utiltidb.IsClusterReady(k8sClient, tc.Name, tc.Namespace)
				g.Expect(ready).To(BeTrue())
				g.Expect(len(tcGet.Status.PD)).NotTo(BeZero())
				for _, compStatus := range tcGet.Status.Components {
					switch compStatus.Kind {
					case v1alpha1.ComponentKindPD, v1alpha1.ComponentKindTiKV, v1alpha1.ComponentKindTiDB, v1alpha1.ComponentKindTiFlash:
						g.Expect(compStatus.Replicas).To(Equal(int32(1)))
					default:
						g.Expect(compStatus.Replicas).To(BeZero())
					}
				}

				g.Expect(utiltidb.AreAllInstancesReady(k8sClient, pdg,
					[]*v1alpha1.TiKVGroup{kvg}, []*v1alpha1.TiDBGroup{dbg}, []*v1alpha1.TiFlashGroup{flashg}, nil)).To(Succeed())
				g.Expect(utiltidb.IsTiDBConnectable(ctx, k8sClient, fw,
					tc.Namespace, tc.Name, dbg.Name, "root", "", "")).To(Succeed())
			}).WithTimeout(createClusterTimeout).WithPolling(createClusterPolling).Should(Succeed())

			By("Checking the store labels and server labels")
			// TODO: extract it to a common utils
			svcName := dbg.Name + "-tidb"
			dsn, cancel, err := utiltidb.PortForwardAndGetTiDBDSN(fw, tc.Namespace, svcName, "root", "", "test", "charset=utf8mb4")
			Expect(err).To(BeNil())
			defer cancel()
			db, err := sql.Open("mysql", dsn)
			Expect(err).To(BeNil())
			defer db.Close()

			Eventually(func(g Gomega) { // retry as the labels may need some time to be set
				plRows, err := db.QueryContext(ctx, "SHOW PLACEMENT LABELS")
				g.Expect(err).To(BeNil())
				defer plRows.Close()
				plKeyToValues := make(map[string]string)
				for plRows.Next() {
					var key, values string
					err = plRows.Scan(&key, &values)
					g.Expect(err).To(BeNil())
					plKeyToValues[key] = values
				}
				g.Expect(plKeyToValues).To(HaveKey("zone")) // short name for `topology.kubernetes.io/zone`
				g.Expect(plKeyToValues).To(HaveKey("host")) // short name for `kubernetes.io/hostname`

				storeRows, err := db.QueryContext(ctx, "SELECT store_id,label FROM INFORMATION_SCHEMA.TIKV_STORE_STATUS")
				g.Expect(err).To(BeNil())
				defer storeRows.Close()
				storeKeyToValues := make(map[int]string)
				for storeRows.Next() {
					var storeID int
					var label string
					err = storeRows.Scan(&storeID, &label)
					g.Expect(err).To(BeNil())
					storeKeyToValues[storeID] = label
				}
				g.Expect(storeKeyToValues).To(HaveLen(2)) // TiKV and TiFlash
				for _, v := range storeKeyToValues {
					g.Expect(v).To(ContainSubstring(`"key": "zone"`))
					g.Expect(v).To(ContainSubstring(`"key": "host"`))
				}

				serverRows, err := db.QueryContext(ctx, "SELECT IP,LABELS FROM INFORMATION_SCHEMA.TIDB_SERVERS_INFO")
				g.Expect(err).To(BeNil())
				defer serverRows.Close()
				serverKeyToValues := make(map[string]string)
				for serverRows.Next() {
					var ip, labels string
					err = serverRows.Scan(&ip, &labels)
					g.Expect(err).To(BeNil())
					serverKeyToValues[ip] = labels
				}
				g.Expect(serverKeyToValues).To(HaveLen(1))
				for _, v := range serverKeyToValues {
					g.Expect(v).To(ContainSubstring(`zone=`))
					g.Expect(v).To(ContainSubstring(`host=`))
				}
			}).WithTimeout(time.Minute).WithPolling(5 * time.Second).Should(Succeed())
		})

		It("should enable readiness probe", func() {
			By("Creating the components with readiness probe enabled")
			pdg := data.NewPDGroup(ns.Name, "pdg", tc.Name, ptr.To(int32(1)), nil)
			kvg := data.NewTiKVGroup(ns.Name, "kvg", tc.Name, ptr.To(int32(1)), nil)
			dbg := data.NewTiDBGroup(ns.Name, "dbg", tc.Name, ptr.To(int32(1)), nil)
			Expect(k8sClient.Create(ctx, pdg)).To(Succeed())
			Expect(k8sClient.Create(ctx, kvg)).To(Succeed())
			Expect(k8sClient.Create(ctx, dbg)).To(Succeed())

			By("Checking the status of the cluster and the connection to the TiDB service")
			Eventually(func(g Gomega) {
				tcGet, ready := utiltidb.IsClusterReady(k8sClient, tc.Name, tc.Namespace)
				g.Expect(ready).To(BeTrue())
				g.Expect(len(tcGet.Status.PD)).NotTo(BeZero())
				for _, compStatus := range tcGet.Status.Components {
					switch compStatus.Kind {
					case v1alpha1.ComponentKindPD, v1alpha1.ComponentKindTiKV, v1alpha1.ComponentKindTiDB:
						g.Expect(compStatus.Replicas).To(Equal(int32(1)))
					default:
						g.Expect(compStatus.Replicas).To(BeZero())
					}
				}

				g.Expect(utiltidb.AreAllInstancesReady(k8sClient, pdg,
					[]*v1alpha1.TiKVGroup{kvg}, []*v1alpha1.TiDBGroup{dbg}, nil, nil)).To(Succeed())
				g.Expect(utiltidb.IsTiDBConnectable(ctx, k8sClient, fw,
					tc.Namespace, tc.Name, dbg.Name, "root", "", "")).To(Succeed())
			}).WithTimeout(createClusterTimeout).WithPolling(createClusterPolling).Should(Succeed())

			By("Checking the readiness probe in the Pods")
			var podList corev1.PodList
			Expect(k8sClient.List(ctx, &podList, client.InNamespace(tc.Namespace))).To(Succeed())
			for _, pod := range podList.Items {
				switch pod.Labels[v1alpha1.LabelKeyComponent] {
				case v1alpha1.LabelValComponentTiDB: // TiDB enables Readiness probe by default
					Expect(pod.Spec.Containers).To(HaveLen(1))
					Expect(pod.Spec.Containers[0].ReadinessProbe).NotTo(BeNil())
					Expect(pod.Spec.Containers[0].ReadinessProbe.TimeoutSeconds).To(Equal(int32(1)))
					Expect(pod.Spec.Containers[0].ReadinessProbe.FailureThreshold).To(Equal(int32(3)))
					Expect(pod.Spec.Containers[0].ReadinessProbe.SuccessThreshold).To(Equal(int32(1)))
					Expect(pod.Spec.Containers[0].ReadinessProbe.InitialDelaySeconds).To(Equal(int32(10)))
					Expect(pod.Spec.Containers[0].ReadinessProbe.PeriodSeconds).To(Equal(int32(10)))
					Expect(pod.Spec.Containers[0].ReadinessProbe.TCPSocket).NotTo(BeNil())
				}
			}
		})

		It("should be able to gracefully shutdown tidb", func() {
			pdg := data.NewPDGroup(ns.Name, "pdg", tc.Name, ptr.To(int32(1)), nil)
			kvg := data.NewTiKVGroup(ns.Name, "kvg", tc.Name, ptr.To(int32(1)), nil)
			dbg := data.NewTiDBGroup(ns.Name, "dbg", tc.Name, ptr.To(int32(3)), func(group *v1alpha1.TiDBGroup) {
				group.Spec.Template.Spec.Config = "graceful-wait-before-shutdown = 60"
			})
			Expect(k8sClient.Create(ctx, pdg)).To(Succeed())
			Expect(k8sClient.Create(ctx, kvg)).To(Succeed())
			Expect(k8sClient.Create(ctx, dbg)).To(Succeed())

			By("Waiting for the cluster to be ready")
			// TODO: extract it to a common utils
			svcName := dbg.Name + "-tidb"
			var clusterIP string
			Eventually(func(g Gomega) {
				_, ready := utiltidb.IsClusterReady(k8sClient, tc.Name, tc.Namespace)
				g.Expect(ready).To(BeTrue())
				g.Expect(utiltidb.AreAllInstancesReady(k8sClient, pdg,
					[]*v1alpha1.TiKVGroup{kvg}, []*v1alpha1.TiDBGroup{dbg}, nil, nil)).To(Succeed())
				svc, err := clientSet.CoreV1().Services(tc.Namespace).Get(ctx, svcName, metav1.GetOptions{})
				g.Expect(err).To(BeNil())
				clusterIP = svc.Spec.ClusterIP
			}).WithTimeout(createClusterTimeout).WithPolling(createClusterPolling).Should(Succeed())

			By("Create a Job to connect to the TiDB cluster to run transactions")
			jobName := "testing-workload-job"
			job := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      jobName,
					Namespace: tc.Namespace,
				},
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app": jobName,
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "testing-workload",
									Image: "pingcap/testing-workload:latest",
									Args: []string{
										"--action", "workload",
										"--host", clusterIP,
										"--duration", "8",
										"--max-connections", "30",
									},
									ImagePullPolicy: corev1.PullIfNotPresent,
								},
							},
							RestartPolicy: corev1.RestartPolicyNever,
						},
					},
					BackoffLimit: ptr.To[int32](0),
				},
			}
			_, err := clientSet.BatchV1().Jobs(tc.Namespace).Create(ctx, job, metav1.CreateOptions{})
			Expect(err).To(BeNil())

			By("Ensure the job pod is running")
			var jobPodName string
			Eventually(func(g Gomega) {
				pods, err := clientSet.CoreV1().Pods(tc.Namespace).List(ctx, metav1.ListOptions{
					LabelSelector: fmt.Sprintf("app=%s", jobName),
				})
				g.Expect(err).To(BeNil())
				g.Expect(len(pods.Items)).To(Equal(1))
				g.Expect(pods.Items[0].Status.Phase).To(Equal(corev1.PodRunning))
				jobPodName = pods.Items[0].Name
			}).WithTimeout(time.Minute).WithPolling(createClusterPolling).Should(Succeed())

			By("Rolling restart TiDB")
			var dbgGet v1alpha1.TiDBGroup
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: tc.Namespace, Name: dbg.Name}, &dbgGet)).To(Succeed())
			dbgGet.Spec.Template.Spec.Config = "log.level = 'warn'"
			Expect(k8sClient.Update(ctx, &dbgGet)).To(Succeed())

			Eventually(func(g Gomega) {
				_, ready := utiltidb.IsClusterReady(k8sClient, tc.Name, tc.Namespace)
				g.Expect(ready).To(BeTrue())
				g.Expect(utiltidb.AreAllInstancesReady(k8sClient, pdg,
					[]*v1alpha1.TiKVGroup{kvg}, []*v1alpha1.TiDBGroup{dbg}, nil, nil)).To(Succeed())
				jobGet, err := clientSet.BatchV1().Jobs(tc.Namespace).Get(ctx, jobName, metav1.GetOptions{})
				g.Expect(err).To(BeNil())
				if jobGet.Status.Failed > 0 {
					// print the logs if the job failed
					req := clientSet.CoreV1().Pods(tc.Namespace).GetLogs(jobPodName, &corev1.PodLogOptions{})
					podLogs, err := req.Stream(ctx)
					g.Expect(err).To(BeNil())
					defer podLogs.Close()

					buf := new(bytes.Buffer)
					_, _ = io.Copy(buf, podLogs)
					GinkgoWriter.Println(buf.String())
					Fail("job failed")
				}
				g.Expect(jobGet.Status.Succeeded).To(BeNumerically("==", 1))
			}).WithTimeout(createClusterTimeout).WithPolling(createClusterPolling).Should(Succeed())
		})
	})

	Context("Overlay", func() {
		It("should be able to overlay the pod's terminationGracePeriodSeconds", func() {
			pdg := data.NewPDGroup(ns.Name, "pdg", tc.Name, ptr.To(int32(1)), nil)
			kvg := data.NewTiKVGroup(ns.Name, "kvg", tc.Name, ptr.To(int32(1)), nil)
			dbg := data.NewTiDBGroup(ns.Name, "dbg", tc.Name, ptr.To(int32(1)), nil)
			Expect(k8sClient.Create(ctx, pdg)).To(Succeed())
			Expect(k8sClient.Create(ctx, kvg)).To(Succeed())
			Expect(k8sClient.Create(ctx, dbg)).To(Succeed())

			By("Waiting for the cluster to be ready")
			Eventually(func(g Gomega) {
				_, ready := utiltidb.IsClusterReady(k8sClient, tc.Name, tc.Namespace)
				g.Expect(ready).To(BeTrue())
				g.Expect(utiltidb.AreAllInstancesReady(k8sClient, pdg,
					[]*v1alpha1.TiKVGroup{kvg}, []*v1alpha1.TiDBGroup{dbg}, nil, nil)).To(Succeed())
			}).WithTimeout(createClusterTimeout).WithPolling(createClusterPolling).Should(Succeed())

			By("Recording the terminationGracePeriodSeconds before overlay")
			listOpts := metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=%s,%s=%s",
					v1alpha1.LabelKeyCluster, tc.Name, v1alpha1.LabelKeyGroup, dbg.Name),
			}
			pods, err := clientSet.CoreV1().Pods(tc.Namespace).List(ctx, listOpts)
			Expect(err).To(BeNil())
			Expect(len(pods.Items)).To(Equal(1))
			beforeTermSeconds := pods.Items[0].Spec.TerminationGracePeriodSeconds
			Expect(beforeTermSeconds).NotTo(BeNil())
			beforeUID := pods.Items[0].UID

			By("Overlaying the terminationGracePeriodSeconds")
			var dbgGet v1alpha1.TiDBGroup
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: tc.Namespace, Name: dbg.Name}, &dbgGet)).To(Succeed())
			overlaySeconds := *beforeTermSeconds + 5
			dbgGet.Spec.Template.Spec.Overlay = &v1alpha1.Overlay{
				Pod: &v1alpha1.PodOverlay{
					Spec: &corev1.PodSpec{TerminationGracePeriodSeconds: &overlaySeconds},
				},
			}
			Expect(k8sClient.Update(ctx, &dbgGet)).To(Succeed())

			Eventually(func(g Gomega) {
				_, ready := utiltidb.IsClusterReady(k8sClient, tc.Name, tc.Namespace)
				g.Expect(ready).To(BeTrue())
				g.Expect(utiltidb.AreAllInstancesReady(k8sClient, pdg,
					[]*v1alpha1.TiKVGroup{kvg}, []*v1alpha1.TiDBGroup{dbg}, nil, nil)).To(Succeed())

				pods, err := clientSet.CoreV1().Pods(tc.Namespace).List(ctx, listOpts)
				Expect(err).To(BeNil())
				Expect(len(pods.Items)).To(Equal(1))
				Expect(pods.Items[0].UID).NotTo(Equal(beforeUID))
				Expect(pods.Items[0].Spec.TerminationGracePeriodSeconds).NotTo(BeNil())
				Expect(*pods.Items[0].Spec.TerminationGracePeriodSeconds).To(Equal(overlaySeconds))
			})
		})
	})
})
