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

package autoscaler

import (
	"fmt"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/autoscaler/autoscaler/query"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	v1alpha1listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	appslisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
)

type autoScalerManager struct {
	kubecli   kubernetes.Interface
	cli       versioned.Interface
	tcLister  v1alpha1listers.TidbClusterLister
	tmLister  v1alpha1listers.TidbMonitorLister
	tcControl controller.TidbClusterControlInterface
	pdControl pdapi.PDControlInterface
	taLister  v1alpha1listers.TidbClusterAutoScalerLister
	stsLister appslisters.StatefulSetLister
	recorder  record.EventRecorder
}

func NewAutoScalerManager(
	kubecli kubernetes.Interface,
	cli versioned.Interface,
	informerFactory informers.SharedInformerFactory,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	recorder record.EventRecorder) *autoScalerManager {
	tcLister := informerFactory.Pingcap().V1alpha1().TidbClusters().Lister()
	return &autoScalerManager{
		kubecli:   kubecli,
		cli:       cli,
		tcLister:  tcLister,
		tcControl: controller.NewRealTidbClusterControl(cli, tcLister, recorder),
		pdControl: pdapi.NewDefaultPDControl(kubecli),
		taLister:  informerFactory.Pingcap().V1alpha1().TidbClusterAutoScalers().Lister(),
		tmLister:  informerFactory.Pingcap().V1alpha1().TidbMonitors().Lister(),
		stsLister: kubeInformerFactory.Apps().V1().StatefulSets().Lister(),
		recorder:  recorder,
	}
}

func (am *autoScalerManager) Sync(tac *v1alpha1.TidbClusterAutoScaler) error {
	if tac.DeletionTimestamp != nil {
		return nil
	}

	tcName := tac.Spec.Cluster.Name
	// When Namespace in TidbClusterRef is omitted, we take tac's namespace as default
	if len(tac.Spec.Cluster.Namespace) < 1 {
		tac.Spec.Cluster.Namespace = tac.Namespace
	}

	tc, err := am.tcLister.TidbClusters(tac.Spec.Cluster.Namespace).Get(tcName)
	if err != nil {
		if errors.IsNotFound(err) {
			// Target TidbCluster Ref is deleted, empty the auto-scaling status
			// And delete all autoscaling clusters
			requirement, _ := labels.NewRequirement(label.AutoInstanceLabelKey, selection.Equals, []string{tac.Name})
			err := am.cli.PingcapV1alpha1().TidbClusters(tac.Spec.Cluster.Namespace).DeleteCollection(nil, v1.ListOptions{
				LabelSelector: labels.NewSelector().Add(*requirement).String(),
			})
			if err != nil {
				klog.Errorf("cannot delete orphan autoscaling clusters for tac[%s/%s], err: %v", tac.Namespace, tac.Name, err)
			}
			return nil
		}
		return err
	}

	defaultTAC(tac, tc)
	if err := validateTAC(tac); err != nil {
		klog.Errorf("invalid spec tac[%s/%s]: %s", tac.Namespace, tac.Name, err.Error())
		return nil
	}

	oldTc := tc.DeepCopy()
	if err := am.syncAutoScaling(tc, tac); err != nil {
		return err
	}

	return am.updateAutoScaling(oldTc, tac)
}

func (am *autoScalerManager) syncExternal(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler, component v1alpha1.MemberType) error {
	switch component {
	case v1alpha1.TiDBMemberType:
		targetReplicas, err := query.ExternalService(tc, v1alpha1.TiDBMemberType, tac.Spec.TiDB.External.Endpoint, am.kubecli)
		if err != nil {
			klog.Errorf("tac[%s/%s] 's query to the external endpoint got error: %v", tac.Namespace, tac.Name, err)
			return err
		}

		if tc.Spec.TiDB.Replicas == targetReplicas {
			return nil
		}

		updated := tc.DeepCopy()
		updated.Spec.TiDB.Replicas = targetReplicas
		if _, err = am.tcControl.UpdateTidbCluster(updated, &updated.Status, &tc.Status); err != nil {
			return err
		}
	case v1alpha1.TiKVMemberType:
		targetReplicas, err := query.ExternalService(tc, v1alpha1.TiKVMemberType, tac.Spec.TiKV.External.Endpoint, am.kubecli)
		if err != nil {
			klog.Errorf("tac[%s/%s] 's query to the external endpoint got error: %v", tac.Namespace, tac.Name, err)
			return err
		}

		if tc.Spec.TiKV.Replicas == targetReplicas {
			return nil
		}

		updated := tc.DeepCopy()
		updated.Spec.TiKV.Replicas = targetReplicas
		if _, err = am.tcControl.UpdateTidbCluster(updated, &updated.Status, &tc.Status); err != nil {
			return err
		}
	}

	return nil
}

func (am *autoScalerManager) syncPD(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler, component v1alpha1.MemberType) error {
	strategy := autoscalerToStrategy(tac, component)
	// Request PD for auto-scaling plans
	plans, err := controller.GetPDClient(am.pdControl, tc).GetAutoscalingPlans(*strategy)
	if err != nil {
		klog.Errorf("cannot get auto-scaling plans for tac[%s/%s] err:%v", tac.Namespace, tac.Name, err)
		return err
	}

	// Apply auto-scaling plans
	if err := am.syncPlans(tc, tac, plans, component); err != nil {
		klog.Errorf("cannot apply autoscaling plans for tac[%s/%s] err:%v", tac.Namespace, tac.Name, err)
		return err
	}
	return nil
}

func (am *autoScalerManager) syncAutoScaling(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler) error {
	if tac.Spec.TiDB != nil {
		if tac.Spec.TiDB.External != nil {
			if err := am.syncExternal(tc, tac, v1alpha1.TiDBMemberType); err != nil {
				return err
			}
		} else {
			if err := am.syncPD(tc, tac, v1alpha1.TiDBMemberType); err != nil {
				return err
			}
		}
	}

	if tac.Spec.TiKV != nil {
		if tac.Spec.TiKV.External != nil {
			if err := am.syncExternal(tc, tac, v1alpha1.TiKVMemberType); err != nil {
				return err
			}
		} else {
			if err := am.syncPD(tc, tac, v1alpha1.TiKVMemberType); err != nil {
				return err
			}
		}
	}

	if err := am.syncMonitor(tc, tac); err != nil {
		return err
	}

	klog.Infof("tc[%s/%s]'s tac[%s/%s] synced", tc.Namespace, tc.Name, tac.Namespace, tac.Name)
	return nil
}

func (am *autoScalerManager) updateAutoScaling(oldTc *v1alpha1.TidbCluster,
	tac *v1alpha1.TidbClusterAutoScaler) error {
	if tac.Annotations == nil {
		tac.Annotations = map[string]string{}
	}
	now := time.Now()
	tac.Annotations[label.AnnLastSyncingTimestamp] = fmt.Sprintf("%d", now.Unix())
	return am.updateTidbClusterAutoScaler(tac)
}

func (am *autoScalerManager) updateTidbClusterAutoScaler(tac *v1alpha1.TidbClusterAutoScaler) error {

	ns := tac.GetNamespace()
	tacName := tac.GetName()
	oldTac := tac.DeepCopy()

	// don't wait due to limited number of clients, but backoff after the default number of steps
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var updateErr error
		_, updateErr = am.cli.PingcapV1alpha1().TidbClusterAutoScalers(ns).Update(tac)
		if updateErr == nil {
			klog.Infof("TidbClusterAutoScaler: [%s/%s] updated successfully", ns, tacName)
			return nil
		}
		klog.V(4).Infof("failed to update TidbClusterAutoScaler: [%s/%s], error: %v", ns, tacName, updateErr)
		if updated, err := am.taLister.TidbClusterAutoScalers(ns).Get(tacName); err == nil {
			// make a copy so we don't mutate the shared cache
			tac = updated.DeepCopy()
			tac.Annotations = oldTac.Annotations
			tac.Status = oldTac.Status
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated TidbClusterAutoScaler %s/%s from lister: %v", ns, tacName, err))
		}
		return updateErr
	})
	if err != nil {
		klog.Errorf("failed to update TidbClusterAutoScaler: [%s/%s], error: %v", ns, tacName, err)
	}
	return err
}
