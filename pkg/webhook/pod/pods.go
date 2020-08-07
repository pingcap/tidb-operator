// Copyright 2019 PingCAP, Inc.
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

package pod

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/openshift/generic-admission-server/pkg/apiserver"
	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	asclientset "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	v1alpha1listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/label"
	memberUtils "github.com/pingcap/tidb-operator/pkg/manager/member"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/pkg/webhook/util"
	admission "k8s.io/api/admission/v1beta1"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	eventv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
)

type PodAdmissionControl struct {
	lock           sync.RWMutex
	initialized    bool
	resyncDuration time.Duration
	// kubernetes client interface
	kubeCli kubernetes.Interface
	// operator client interface
	operatorCli versioned.Interface
	// pd Control
	pdControl pdapi.PDControlInterface
	// the map of the service account from the request which should be checked by webhook
	serviceAccounts sets.String
	// tc lister
	tcLister v1alpha1listers.TidbClusterLister
	// tikv group lister
	tikvGroupLister v1alpha1listers.TiKVGroupLister
	// recorder to send event
	recorder record.EventRecorder
}

var _ apiserver.ValidatingAdmissionHook = &PodAdmissionControl{}
var _ apiserver.MutatingAdmissionHook = &PodAdmissionControl{}

const (
	stsControllerServiceAccounts = "system:serviceaccount:kube-system:statefulset-controller"
	podDeleteMsgPattern          = "pod [%s] deleted"
	pdScaleInReason              = "PDScaleIn"
	pdUpgradeReason              = "PDUpgrade"
	tikvScaleInReason            = "TiKVScaleIn"
	tikvUpgradeReason            = "TiKVUpgrade"
)

var (
	AstsControllerServiceAccounts string
)

func NewPodAdmissionControl(extraServiceAccounts []string, evictRegionLeaderTimeout time.Duration, resyncDuration time.Duration) *PodAdmissionControl {
	serviceAccounts := sets.NewString(stsControllerServiceAccounts)
	for _, sa := range extraServiceAccounts {
		serviceAccounts.Insert(sa)
	}
	if features.DefaultFeatureGate.Enabled(features.AdvancedStatefulSet) {
		serviceAccounts.Insert(AstsControllerServiceAccounts)
	}
	EvictLeaderTimeout = evictRegionLeaderTimeout
	return &PodAdmissionControl{
		serviceAccounts: serviceAccounts,
		resyncDuration:  resyncDuration,
	}
}

func (w *PodAdmissionControl) ValidatingResource() (plural schema.GroupVersionResource, singular string) {
	return schema.GroupVersionResource{
			Group:    "admission.tidb.pingcap.com",
			Version:  "v1alpha1",
			Resource: "podvalidations",
		},
		"podvalidation"
}

func (w *PodAdmissionControl) MutatingResource() (plural schema.GroupVersionResource, singular string) {
	return schema.GroupVersionResource{
			Group:    "admission.tidb.pingcap.com",
			Version:  "v1alpha1",
			Resource: "podmutations",
		},
		"podmutation"
}

// admitPayload used to simply the param to make each function more easier
type admitPayload struct {
	// the pod for admission request object
	pod *core.Pod
	// the ownerStatefulSet for target tidb component pod
	ownerStatefulSet *apps.StatefulSet
	// the owner controller for target tidb component pod
	controller runtime.Object
	// the pdClient for target tc
	pdClient pdapi.PDClient
	// description for pod's controller
	controllerDesc controllerDesc
}

func (pc *PodAdmissionControl) Admit(ar *admission.AdmissionRequest) *admission.AdmissionResponse {
	pc.lock.RLock()
	defer pc.lock.RUnlock()
	if !pc.initialized {
		return &admission.AdmissionResponse{
			Allowed: false,
		}
	}

	if ar.Operation != admission.Create && ar.Operation != admission.Update {
		return util.ARSuccess()
	}
	return pc.mutatePod(ar)
}

func (pc *PodAdmissionControl) Validate(ar *admission.AdmissionRequest) *admission.AdmissionResponse {
	pc.lock.RLock()
	defer pc.lock.RUnlock()
	if !pc.initialized {
		return &admission.AdmissionResponse{
			Allowed: false,
		}
	}

	name := ar.Name
	namespace := ar.Namespace
	operation := ar.Operation
	serviceAccount := ar.UserInfo.Username
	klog.Infof("receive %s pod[%s/%s] by sa[%s]", operation, namespace, name, serviceAccount)

	if !pc.serviceAccounts.Has(serviceAccount) {
		klog.Infof("Request was not sent by known controlled ServiceAccounts, admit to %s pod [%s/%s]", operation, namespace, name)
		return util.ARSuccess()
	}

	switch operation {
	case admission.Delete:
		return pc.admitDeletePods(name, namespace)
	case admission.Create:
		return pc.AdmitCreatePods(ar)
	default:
		klog.Infof("Admit to %s pod[%s/%s]", operation, namespace, name)
		return util.ARSuccess()
	}

}

// Webhook server receive request to delete pod
// if this pod wasn't member of tidbcluster, just let the request pass.
// Otherwise, we check tidbcluster and statefulset which own this pod whether they were existed.
// If either of them were deleted,we would let this request pass,
// otherwise we will check it decided by component type.
func (pc *PodAdmissionControl) admitDeletePods(name, namespace string) *admission.AdmissionResponse {

	klog.Infof("receive admission to %s pod[%s/%s]", "delete", namespace, name)

	// We would update pod annotations if they were deleted member by admission controller,
	// so we shall find this pod from apiServer considering getting latest pod info.
	pod, err := pc.kubeCli.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		klog.Infof("failed to find pod[%s/%s] during delete it,admit to delete", namespace, name)
		return util.ARSuccess()
	}

	l := label.Label(pod.Labels)

	if !l.IsManagedByTiDBOperator() {
		klog.Infof("pod[%s/%s] is not managed by TiDB-Operator,admit to create", namespace, name)
		return util.ARSuccess()
	}

	if !(l.IsPD() || l.IsTiKV() || l.IsTiDB()) {
		klog.Infof("pod[%s/%s] is not TiDB component,admit to delete", namespace, name)
		return util.ARSuccess()
	}

	ownerStatefulSet, err := getOwnerStatefulSetForTiDBComponent(pod, pc.kubeCli)
	if err != nil {
		if errors.IsNotFound(err) || err.Error() == fmt.Sprintf(failToFindTidbComponentOwnerStatefulset, namespace, name) {
			klog.Infof("owner statefulset for pod[%s/%s] is deleted,admit to delete pod", namespace, name)
			return util.ARSuccess()
		}
		klog.Infof("failed to get owner statefulset for pod[%s/%s],refuse to delete pod", namespace, name)
		return util.ARFail(fmt.Errorf("failed to get owner statefulset for pod[%s/%s],refuse to delete pod", namespace, name))
	}

	// When AdvancedStatefulSet is enabled, the ordinal of the last pod in the statefulset could be a non-zero number,
	// so we let the deleting request of the last pod pass when spec.replicas <= 1 and status.replicas equals 1
	if *ownerStatefulSet.Spec.Replicas <= 1 && ownerStatefulSet.Status.Replicas == 1 {
		klog.Infof("statefulset[%s/%s] only have one pod[%s/%s],admit to delete it.", namespace, ownerStatefulSet.Name, namespace, name)
		return util.ARSuccess()
	}

	if l.IsPD() {
		return pc.processAdmitDeletePDPod(pod, ownerStatefulSet)
	} else if l.IsTiKV() {
		return pc.processAdmitDeleteTiKVPod(pod, ownerStatefulSet)
	}

	klog.Infof("[%s/%s] is admit to be deleted", namespace, name)
	return util.ARSuccess()
}

func (pc *PodAdmissionControl) processAdmitDeletePDPod(pod *core.Pod, ownerStatefulSet *apps.StatefulSet) *admission.AdmissionResponse {
	name := pod.Name
	namespace := pod.Namespace
	tcName, exist := pod.Labels[label.InstanceLabelKey]
	if !exist {
		klog.Errorf("pod[%s/%s] has no label: %s", namespace, name, label.InstanceLabelKey)
		return util.ARSuccess()
	}

	tc, err := pc.tcLister.TidbClusters(namespace).Get(tcName)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.Infof("tc[%s/%s] had been deleted,admit to delete pod[%s/%s]", namespace, tcName, namespace, name)
			return util.ARSuccess()
		}
		klog.Errorf("failed get tc[%s/%s],refuse to delete pod[%s/%s]", namespace, tcName, namespace, name)
		return util.ARFail(err)
	}
	// Force Upgraded,Admit to Upgrade
	if memberUtils.NeedForceUpgrade(tc) {
		klog.Infof("tc[%s/%s] is force upgraded, admit to delete pod[%s/%s]", namespace, tcName, namespace, name)
		return util.ARSuccess()
	}
	payload := &admitPayload{
		pod:              pod,
		controller:       tc,
		ownerStatefulSet: ownerStatefulSet,
		pdClient:         pc.pdControl.GetPDClient(pdapi.Namespace(namespace), tcName, tc.IsTLSClusterEnabled()),
	}
	return pc.admitDeletePdPods(payload)
}

func (pc *PodAdmissionControl) processAdmitDeleteTiKVPod(pod *core.Pod, ownerStatefulSet *apps.StatefulSet) *admission.AdmissionResponse {
	name := pod.Name
	namespace := pod.Namespace
	l := label.Label(pod.Labels)
	payload := &admitPayload{
		pod:              pod,
		ownerStatefulSet: ownerStatefulSet,
	}
	controllerName, exist := pod.Labels[label.InstanceLabelKey]
	if !exist {
		klog.Errorf("pod[%s/%s] has no label: %s", namespace, name, label.InstanceLabelKey)
		return util.ARSuccess()
	}

	if l.IsTidbClusterPod() {
		tcName := controllerName
		tc, err := pc.tcLister.TidbClusters(namespace).Get(tcName)
		if err != nil {
			if errors.IsNotFound(err) {
				klog.Infof("tc[%s/%s] had been deleted,admit to delete pod[%s/%s]", namespace, tcName, namespace, name)
				return util.ARSuccess()
			}
			klog.Errorf("failed get tc[%s/%s],refuse to delete pod[%s/%s]", namespace, tcName, namespace, name)
			return util.ARFail(err)
		}
		payload.pdClient = pc.pdControl.GetPDClient(pdapi.Namespace(namespace), tcName, tc.IsTLSClusterEnabled())
		payload.controller = tc
		payload.controllerDesc = controllerDesc{
			name:      tcName,
			namespace: namespace,
			kind:      v1alpha1.TiDBClusterKind,
		}
		return pc.admitDeleteTiKVPods(payload)
	} else if l.IsGroupPod() {
		tgName := controllerName
		tg, err := pc.tikvGroupLister.TiKVGroups(namespace).Get(tgName)
		if err != nil {
			if errors.IsNotFound(err) {
				klog.Infof("tikvgroup[%s/%s] had been deleted,admit to delete pod[%s/%s]", namespace, tgName, namespace, name)
				return util.ARSuccess()
			}
			klog.Errorf("failed get tikvgroup[%s/%s],refuse to delete pod[%s/%s]", namespace, tgName, namespace, name)
			return util.ARFail(err)
		}
		ownerTcName := tg.Spec.ClusterName
		tc, err := pc.tcLister.TidbClusters(namespace).Get(ownerTcName)
		if err != nil {
			// Event if the ownerTC is deleted, we won't delete the tikvgroup pod unless its owner controller is deleted
			klog.Errorf("failed get tc[%s/%s],refuse to delete pod[%s/%s]", namespace, ownerTcName, namespace, name)
			return util.ARFail(err)
		}
		payload.pdClient = pc.pdControl.GetPDClient(pdapi.Namespace(namespace), ownerTcName, tc.IsTLSClusterEnabled())
		payload.controller = tg
		payload.controllerDesc = controllerDesc{
			name:      tgName,
			namespace: namespace,
			kind:      v1alpha1.TiKVGroupKind,
		}
		return pc.admitDeleteTiKVPods(payload)
	}

	klog.Infof("tikv pod[%s/%s] is not managed by tidbcluster or tikvgroup, admit to be deleted", namespace, name)
	return util.ARSuccess()
}

// Webhook server receive request to create pod
// if this pod wasn't member of tidbcluster, just let the request pass.
// Currently we only check with tikv pod
func (pc *PodAdmissionControl) AdmitCreatePods(ar *admission.AdmissionRequest) *admission.AdmissionResponse {
	pod := &core.Pod{}
	if err := json.Unmarshal(ar.Object.Raw, pod); err != nil {
		klog.Errorf("Could not unmarshal raw object: %v", err)
		return util.ARFail(err)
	}

	name := pod.Name
	namespace := pod.Namespace

	klog.Infof("receive admission to %s pod[%s/%s]", "create", namespace, name)

	l := label.Label(pod.Labels)

	if !l.IsManagedByTiDBOperator() {
		klog.Infof("pod[%s/%s] is not managed by TiDB-Operator,admit to create", namespace, name)
		return util.ARSuccess()
	}

	if !(l.IsPD() || l.IsTiKV() || l.IsTiDB()) {
		klog.Infof("pod[%s/%s] is not TiDB component,admit to create", namespace, name)
		return util.ARSuccess()
	}

	var ownerTc *v1alpha1.TidbCluster
	var err error
	if l.IsTidbClusterPod() {
		tcName, exist := pod.Labels[label.InstanceLabelKey]
		if !exist {
			return util.ARSuccess()
		}
		ownerTc, err = pc.tcLister.TidbClusters(namespace).Get(tcName)
		if err != nil {
			klog.Errorf("failed get tc[%s/%s],refuse to create pod[%s/%s],%v", namespace, tcName, namespace, name, err)
			return util.ARFail(err)
		}
	} else if l.IsGroupPod() {
		tgName, exist := pod.Labels[label.InstanceLabelKey]
		if !exist {
			return util.ARSuccess()
		}
		tg, err := pc.tikvGroupLister.TiKVGroups(namespace).Get(tgName)
		if err != nil {
			klog.Errorf("failed get tikvgroup[%s/%s],refuse to create pod[%s/%s],%v", namespace, tgName, namespace, name, err)
			return util.ARFail(err)
		}
		tcName := tg.Spec.ClusterName
		ownerTc, err = pc.tcLister.TidbClusters(namespace).Get(tcName)
		if err != nil {
			klog.Errorf("failed get tc[%s/%s],refuse to create pod[%s/%s],%v", namespace, tcName, namespace, name, err)
			return util.ARFail(err)
		}
	} else {
		klog.Infof("tikv pod[%s/%s] has unknown controller, admit to create", namespace, name)
		return util.ARSuccess()
	}

	if l.IsTiKV() {
		pdClient := pc.pdControl.GetPDClient(pdapi.Namespace(namespace), ownerTc.Name, ownerTc.IsTLSClusterEnabled())
		return pc.admitCreateTiKVPod(pod, pdClient)
	}

	return util.ARSuccess()
}

// Initialize implements AdmissionHook.Initialize interface. It's is called as
// a post-start hook.
func (a *PodAdmissionControl) Initialize(cfg *rest.Config, stopCh <-chan struct{}) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	cli, err := versioned.NewForConfig(cfg)
	if err != nil {
		return err
	}

	var kubeCli kubernetes.Interface
	kubeCli, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}
	asCli, err := asclientset.NewForConfig(cfg)
	if err != nil {
		return err
	}

	if features.DefaultFeatureGate.Enabled(features.AdvancedStatefulSet) {
		// If AdvancedStatefulSet is enabled, we hijack the Kubernetes client to use
		// AdvancedStatefulSet.
		kubeCli = helper.NewHijackClient(kubeCli, asCli)
	}

	a.kubeCli = kubeCli
	a.operatorCli = cli

	// init pdControl
	a.pdControl = pdapi.NewDefaultPDControl(kubeCli)

	// init recorder
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&eventv1.EventSinkImpl{
		Interface: eventv1.New(kubeCli.CoreV1().RESTClient()).Events("")})
	a.recorder = eventBroadcaster.NewRecorder(v1alpha1.Scheme, core.EventSource{Component: "tidb-admission-controller"})

	// informer factory
	informerFactory := informers.NewSharedInformerFactoryWithOptions(cli, a.resyncDuration)

	// initialize listers
	a.tcLister = informerFactory.Pingcap().V1alpha1().TidbClusters().Lister()
	a.tikvGroupLister = informerFactory.Pingcap().V1alpha1().TiKVGroups().Lister()

	// Start informer factories after all controller are initialized.
	informerFactory.Start(stopCh)

	// Wait for all started informers' cache were synced.
	for v, synced := range informerFactory.WaitForCacheSync(wait.NeverStop) {
		if !synced {
			klog.Fatalf("error syncing informer for %v", v)
		}
	}

	a.initialized = true
	return nil
}
