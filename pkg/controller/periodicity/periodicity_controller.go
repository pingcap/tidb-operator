package periodicity

import (
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	v1alpha1listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	eventv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	appslisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
)

type Controller struct {
	stsLister          appslisters.StatefulSetLister
	tcLister           v1alpha1listers.TidbClusterLister
	statefulSetControl controller.StatefulSetControlInterface
}

func NewController(
	kubeCli kubernetes.Interface,
	informerFactory informers.SharedInformerFactory,
	kubeInformerFactory kubeinformers.SharedInformerFactory) *Controller {

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&eventv1.EventSinkImpl{
		Interface: eventv1.New(kubeCli.CoreV1().RESTClient()).Events("")})
	recorder := eventBroadcaster.NewRecorder(v1alpha1.Scheme, corev1.EventSource{Component: "periodiciy-controller"})
	stsLister := kubeInformerFactory.Apps().V1().StatefulSets().Lister()

	return &Controller{
		tcLister:           informerFactory.Pingcap().V1alpha1().TidbClusters().Lister(),
		statefulSetControl: controller.NewRealStatefuSetControl(kubeCli, stsLister, recorder),
		stsLister:          stsLister,
	}

}

func (c *Controller) Run() {
	klog.Infof("Start to running periodicity job")
	var errs []error
	if controller.PodWebhookEnabled {
		if err := c.syncStatefulSetTimeStamp(); err != nil {
			errs = append(errs, err)
		}
	}
	klog.Errorf("error happened in periodicity controller,err:%v", errors.NewAggregate(errs))
}

// refer: https://github.com/pingcap/tidb-operator/pull/1875
func (c *Controller) syncStatefulSetTimeStamp() error {
	selector, err := label.New().Selector()
	if err != nil {
		return err
	}
	stsList, err := c.stsLister.List(selector)
	if err != nil {
		return err
	}
	var errs []error
	for _, sts := range stsList {
		// If there is any error during our sts annotation updating, we just collect the error
		// and continue to next sts
		if sts.Annotations == nil {
			sts.Annotations = map[string]string{}
		}
		if sts.Labels == nil {
			sts.Labels = map[string]string{}
		}
		tcName, ok := sts.Labels[label.InstanceLabelKey]
		if !ok {
			continue
		}
		tc, err := c.tcLister.TidbClusters(sts.Namespace).Get(tcName)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		sts.Annotations[label.AnnStsLastSyncTimestamp] = time.Now().Format(time.RFC3339)
		newSts, err := c.statefulSetControl.UpdateStatefulSet(tc, sts)
		if err != nil {
			klog.Errorf("failed to update statefulset %q, error: %v", sts.Name, err)
			errs = append(errs, err)
		}
		klog.Infof("newSts[%s], annotation value=%v", newSts.Name, newSts.Annotations)
	}
	return errors.NewAggregate(errs)
}
