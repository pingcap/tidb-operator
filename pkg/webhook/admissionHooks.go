package webhook

import (
	"github.com/pingcap/tidb-operator/pkg/webhook/util"
	"time"

	"github.com/pingcap/advanced-statefulset/pkg/apis/apps/v1alpha1/helper"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/pkg/webhook/pod"
	admission "k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"

	asclientset "github.com/pingcap/advanced-statefulset/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	core "k8s.io/api/core/v1"
	kubeinformers "k8s.io/client-go/informers"
	event "k8s.io/client-go/kubernetes/typed/core/v1"
)

func (a *PodAdmissionHook) ValidatingResource() (plural schema.GroupVersionResource, singular string) {
	return schema.GroupVersionResource{
			Group:    "admission.tidb.pingcap.com",
			Version:  "v1alpha1",
			Resource: "podadmissionreviews",
		},
		"PodAdmissionReview"
}

func (a *PodAdmissionHook) Validate(ar *admission.AdmissionRequest) *admission.AdmissionResponse {
	if !a.initialized {
		return &admission.AdmissionResponse{
			Allowed: false,
		}
	}
	if ar.Kind.Kind != "Pod" {
		klog.Infof("success to %v %s[%s/%s]", ar.Operation, ar.Kind.Kind, ar.Name, ar.Namespace)
		return util.ARSuccess()
	}
	return a.podAC.AdmitPods(ar)
}

// any special initialization goes here
func (a *PodAdmissionHook) Initialize(cfg *rest.Config, stopCh <-chan struct{}) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	cli, err := versioned.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("failed to create Clientset: %v", err)
	}

	var kubeCli kubernetes.Interface
	kubeCli, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("failed to get kubernetes Clientset: %v", err)
	}
	asCli, err := asclientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("failed to get advanced-statefulset Clientset: %v", err)
	}

	if features.DefaultFeatureGate.Enabled(features.AdvancedStatefulSet) {
		// If AdvancedStatefulSet is enabled, we hijack the Kubernetes client to use
		// AdvancedStatefulSet.
		kubeCli = helper.NewHijackClient(kubeCli, asCli)
	}

	informerFactory := informers.NewSharedInformerFactory(cli, controller.ResyncDuration)
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeCli, controller.ResyncDuration)

	// init pdControl
	pdControl := pdapi.NewDefaultPDControl(kubeCli)

	// init recorder
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&event.EventSinkImpl{
		Interface: event.New(kubeCli.CoreV1().RESTClient()).Events("")})
	recorder := eventBroadcaster.NewRecorder(v1alpha1.Scheme, core.EventSource{Component: "tidbcluster"})

	pc := pod.NewPodAdmissionControl(kubeCli, cli, pdControl, informerFactory, kubeInformerFactory, recorder, []string{}, 3*time.Minute)
	a.podAC = pc
	informerFactory.Start(stopCh)
	kubeInformerFactory.Start(stopCh)

	// Wait for all started informers' cache were synced.
	for v, synced := range informerFactory.WaitForCacheSync(wait.NeverStop) {
		if !synced {
			klog.Fatalf("error syncing informer for %v", v)
		}
	}
	for v, synced := range kubeInformerFactory.WaitForCacheSync(wait.NeverStop) {
		if !synced {
			klog.Fatalf("error syncing informer for %v", v)
		}
	}
	a.initialized = true
	klog.Info("webhook initialized successfully")
	return nil
}
