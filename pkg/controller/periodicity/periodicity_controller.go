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
	"encoding/json"
	"time"

	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/util"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
)

type Controller struct {
	deps *controller.Dependencies
}

func NewController(deps *controller.Dependencies) *Controller {
	return &Controller{
		deps: deps,
	}
}

func (c *Controller) Run(_ int, stopCh <-chan struct{}) {
	klog.Info("Staring periodicity controller")
	defer klog.Info("Shutting down periodicity controller")
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
	stsList, err := c.deps.StatefulSetLister.List(selector)
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
		_, err := c.deps.TiDBClusterLister.TidbClusters(sts.Namespace).Get(tcRef.Name)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		patchAnnotations := map[string]string{}
		patchAnnotations[label.AnnStsLastSyncTimestamp] = time.Now().Format(time.RFC3339)
		var mergePatch []byte
		mergePatch, err = json.Marshal(map[string]interface{}{
			"metadata": map[string]interface{}{
				"annotations": patchAnnotations,
			},
		})
		if err != nil {
			errs = append(errs, err)
			continue
		}
		_, err = c.deps.KubeClientset.AppsV1().StatefulSets(sts.Namespace).Patch(sts.Name, types.MergePatchType, mergePatch)
		if err != nil {
			klog.Errorf("sts[%s/%s] patch timestamp failed, error: %v", sts.Namespace, sts.Name, err.Error())
			errs = append(errs, err)
			continue
		}
	}
	return errors.NewAggregate(errs)
}
