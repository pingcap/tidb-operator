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

package webhook

import (
	"strings"
	"sync"
	"time"

	"github.com/openshift/generic-admission-server/pkg/apiserver"
	asappsv1 "github.com/pingcap/advanced-statefulset/client/apis/apps/v1"
	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	asclientset "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
<<<<<<< HEAD
=======
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
>>>>>>> 6a96869... webhook: Register Strategy AdmissionHook at separate API endpoint (#3047)
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/pkg/webhook/pod"
	"github.com/pingcap/tidb-operator/pkg/webhook/statefulset"
	"github.com/pingcap/tidb-operator/pkg/webhook/util"
	admission "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	eventv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
)

// AdmissionHook implements both ValidatingAdmissionHook and
// MutatingAdmissionHook interface.
type AdmissionHook struct {
	lock                     sync.RWMutex
	initialized              bool
	podAC                    *pod.PodAdmissionControl
	stsAC                    *statefulset.StatefulSetAdmissionControl
<<<<<<< HEAD
	strategyAC               *strategy.AdmissionWebhook
=======
	ResyncDuration           time.Duration
>>>>>>> 6a96869... webhook: Register Strategy AdmissionHook at separate API endpoint (#3047)
	ExtraServiceAccounts     string
	EvictRegionLeaderTimeout time.Duration
}

var _ apiserver.ValidatingAdmissionHook = &AdmissionHook{}
var _ apiserver.MutatingAdmissionHook = &AdmissionHook{}

func (a *AdmissionHook) ValidatingResource() (plural schema.GroupVersionResource, singular string) {
	return schema.GroupVersionResource{
			Group:    "admission.tidb.pingcap.com",
			Version:  "v1alpha1",
			Resource: "admissionreviews",
		},
		"PodAdmissionReview"
}

func (a *AdmissionHook) Validate(ar *admission.AdmissionRequest) *admission.AdmissionResponse {
	a.lock.RLock()
	defer a.lock.RUnlock()
	if !a.initialized {
		return &admission.AdmissionResponse{
			Allowed: false,
		}
	}

	switch ar.Kind.Kind {
	case "Pod":
		if "" != ar.Kind.Group {
			return a.unknownAdmissionRequest(ar)
		}
		return a.podAC.AdmitPods(ar)
	case "StatefulSet":
		expectedGroup := "apps"
		if features.DefaultFeatureGate.Enabled(features.AdvancedStatefulSet) {
			expectedGroup = asappsv1.GroupName
		}
		if expectedGroup != ar.Kind.Group {
			return a.unknownAdmissionRequest(ar)
		}
		return a.stsAC.AdmitStatefulSets(ar)
	default:
		return a.unknownAdmissionRequest(ar)
	}
}

func (a *AdmissionHook) MutatingResource() (plural schema.GroupVersionResource, singular string) {
	return schema.GroupVersionResource{
			Group:    "admission.tidb.pingcap.com",
			Version:  "v1alpha1",
			Resource: "mutatingreviews",
		},
		"GenericMutatingReview"
}

func (a *AdmissionHook) Admit(ar *admission.AdmissionRequest) *admission.AdmissionResponse {
	a.lock.RLock()
	defer a.lock.RUnlock()

	name := ar.Name
	namespace := ar.Namespace
	kind := ar.Kind.Kind
	klog.Infof("receive mutation request for %s[%s/%s]", kind, namespace, name)

	switch ar.Kind.Kind {
	case "Pod":
		if "" != ar.Kind.Group {
			return a.unknownAdmissionRequest(ar)
		}
		return a.podAC.MutatePods(ar)
	default:
		return a.unknownAdmissionRequest(ar)
	}
}

// Initialize implements AdmissionHook.Initialize interface. It's is called as
// a post-start hook.
func (a *AdmissionHook) Initialize(cfg *rest.Config, stopCh <-chan struct{}) error {
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

	// init pdControl
	pdControl := pdapi.NewDefaultPDControl(kubeCli)

	// init recorder
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&eventv1.EventSinkImpl{
		Interface: eventv1.New(kubeCli.CoreV1().RESTClient()).Events("")})
	recorder := eventBroadcaster.NewRecorder(v1alpha1.Scheme, corev1.EventSource{Component: "tidb-admission-controller"})

<<<<<<< HEAD
	pc := pod.NewPodAdmissionControl(kubeCli, cli, pdControl, strings.Split(a.ExtraServiceAccounts, ","), a.EvictRegionLeaderTimeout, recorder)
	a.podAC = pc
=======
	// informer factory
	informerFactory := informers.NewSharedInformerFactoryWithOptions(cli, a.ResyncDuration)

	a.podAC = pod.NewPodAdmissionControl(kubeCli, cli, pdControl, strings.Split(a.ExtraServiceAccounts, ","), a.EvictRegionLeaderTimeout, informerFactory, recorder)
>>>>>>> 6a96869... webhook: Register Strategy AdmissionHook at separate API endpoint (#3047)
	klog.Info("pod admission webhook initialized successfully")
	a.stsAC = statefulset.NewStatefulSetAdmissionControl(cli)
	klog.Info("statefulset admission webhook initialized successfully")

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

func (a *AdmissionHook) unknownAdmissionRequest(ar *admission.AdmissionRequest) *admission.AdmissionResponse {
	klog.Infof("success to %v %s[%s/%s]", ar.Operation, ar.Kind.Kind, ar.Namespace, ar.Name)
	return util.ARSuccess()
}
