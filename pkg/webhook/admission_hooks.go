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

	"github.com/pingcap/advanced-statefulset/pkg/apis/apps/v1alpha1/helper"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/pkg/webhook/pod"
	"github.com/pingcap/tidb-operator/pkg/webhook/util"
	admission "k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"

	asappsv1alpha1 "github.com/pingcap/advanced-statefulset/pkg/apis/apps/v1alpha1"
	asclientset "github.com/pingcap/advanced-statefulset/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/webhook/statefulset"
	core "k8s.io/api/core/v1"
	kubeinformers "k8s.io/client-go/informers"
	event "k8s.io/client-go/kubernetes/typed/core/v1"
)

type AdmissionHook struct {
	lock                     sync.RWMutex
	initialized              bool
	podAC                    *pod.PodAdmissionControl
	stsAC                    *statefulset.StatefulSetAdmissionControl
	ExtraServiceAccounts     string
	EvictRegionLeaderTimeout time.Duration
}

func (a *AdmissionHook) ValidatingResource() (plural schema.GroupVersionResource, singular string) {
	return schema.GroupVersionResource{
			Group:    "admission.tidb.pingcap.com",
			Version:  "v1alpha1",
			Resource: "podadmissionreviews",
		},
		"PodAdmissionReview"
}

func (a *AdmissionHook) Validate(ar *admission.AdmissionRequest) *admission.AdmissionResponse {
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
			expectedGroup = asappsv1alpha1.GroupName
		}
		if expectedGroup != ar.Kind.Group {
			return a.unknownAdmissionRequest(ar)
		}
		return a.stsAC.AdmitStatefulSets(ar)
	default:
		return a.unknownAdmissionRequest(ar)
	}
}

// any special initialization goes here
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

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeCli, controller.ResyncDuration)

	// init pdControl
	pdControl := pdapi.NewDefaultPDControl(kubeCli)

	// init recorder
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&event.EventSinkImpl{
		Interface: event.New(kubeCli.CoreV1().RESTClient()).Events("")})
	recorder := eventBroadcaster.NewRecorder(v1alpha1.Scheme, core.EventSource{Component: "tidbcluster"})

	pc := pod.NewPodAdmissionControl(kubeCli, cli, pdControl, kubeInformerFactory, recorder, strings.Split(a.ExtraServiceAccounts, ","), a.EvictRegionLeaderTimeout)
	a.podAC = pc
	kubeInformerFactory.Start(stopCh)

	// Wait for all started informers' cache were synced.
	for _, synced := range kubeInformerFactory.WaitForCacheSync(wait.NeverStop) {
		if !synced {
			return err
		}
	}
	klog.Info("pod admission webhook initialized successfully")
	a.stsAC = statefulset.NewStatefulSetAdmissionControl(cli)
	klog.Info("statefulset admission webhook initialized successfully")
	a.initialized = true
	return nil
}

func (a *AdmissionHook) unknownAdmissionRequest(ar *admission.AdmissionRequest) *admission.AdmissionResponse {
	klog.Infof("success to %v %s[%s/%s]", ar.Operation, ar.Kind.Kind, ar.Name, ar.Namespace)
	return util.ARSuccess()
}
