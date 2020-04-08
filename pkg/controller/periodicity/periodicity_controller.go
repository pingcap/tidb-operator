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

// Package periodicity dedicate the periodicity controller.
// This controller updates StatefulSets managed by our operator periodically.
// This is necessary when the pod admission webhook is used. Because we will
// deny pod deletion requests if the pod is not ready for deletion. However,
// retry duration on StatefulSet in its controller grows exponentially on
// failures. So we need to update StatefulSets to trigger events, then they
// will be put into the process queue of StatefulSet controller constantly.
// Refer to https://github.com/pingcap/tidb-operator/pull/1875 and
// https://github.com/pingcap/tidb-operator/issues/1846 for more details.
package periodicity

import (
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	v1alpha1listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
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

func (c *Controller) Run(stopCh <-chan struct{}) {
	klog.Infof("Staring periodicity controller")
	defer klog.Infof("Shutting down periodicity controller")
	wait.Until(c.run, time.Minute, stopCh)
}

func (c *Controller) run() {
	var errs []error
	if err := c.syncStatefulSetTimeStamp(); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		klog.Errorf("error happened in periodicity controller,err:%v", errors.NewAggregate(errs))
	}
}

// in this sync function, we update all stateful sets the operator managed and log errors
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
		ok, tcRef := util.IsOwnedByTidbCluster(sts)
		if !ok {
			continue
		}
		tc, err := c.tcLister.TidbClusters(sts.Namespace).Get(tcRef.Name)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if sts.Annotations == nil {
			sts.Annotations = map[string]string{}
		}
		sts.Annotations[label.AnnStsLastSyncTimestamp] = time.Now().Format(time.RFC3339)
		newSts, err := c.statefulSetControl.UpdateStatefulSet(tc, sts)
		if err != nil {
			klog.Errorf("failed to update statefulset %q, error: %v", sts.Name, err)
			errs = append(errs, err)
		}
		klog.Infof("successfully updated statefulset %q", newSts.Name)
	}
	return errors.NewAggregate(errs)
}
