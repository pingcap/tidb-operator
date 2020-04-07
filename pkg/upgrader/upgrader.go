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

package upgrader

import (
	"fmt"

	asappsv1 "github.com/pingcap/advanced-statefulset/pkg/apis/apps/v1"
	"github.com/pingcap/advanced-statefulset/pkg/apis/apps/v1/helper"
	asclientset "github.com/pingcap/advanced-statefulset/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/util"
	utildiscovery "github.com/pingcap/tidb-operator/pkg/util/discovery"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

// Interface represents the interface of upgrader.
type Interface interface {
	// Upgrade runs in the boot process and performs the function of upgrade
	// for the current operator.
	//
	// Implemention of this interface must be idempotent. If it fails, the
	// program can be restarted to retry again without affecting current
	// Kubernetes cluster and TiDB clusters.
	//
	// Note that it's possible that it cann't finish its job when you're trying
	// to upgrade a newer operator to an older one or enable/disable some
	// irreversible features.
	Upgrade() error
}

type upgrader struct {
	kubeCli kubernetes.Interface
	cli     versioned.Interface
	asCli   asclientset.Interface
	ns      string
}

var _ Interface = &upgrader{}

func (u *upgrader) Upgrade() error {
	if features.DefaultFeatureGate.Enabled(features.AdvancedStatefulSet) {
		klog.Infof("Upgrader: migrating Kubernetes StatefulSets to Advanced StatefulSets")
		stsList, err := u.kubeCli.AppsV1().StatefulSets(u.ns).List(metav1.ListOptions{})
		if err != nil {
			return err
		}
		stsToMigrate := make([]appsv1.StatefulSet, 0)
		tidbClusters := make([]*v1alpha1.TidbCluster, 0)
		for _, sts := range stsList.Items {
			if ok, tcRef := util.IsOwnedByTidbCluster(&sts); ok {
				stsToMigrate = append(stsToMigrate, sts)
				tc, err := u.cli.PingcapV1alpha1().TidbClusters(sts.Namespace).Get(tcRef.Name, metav1.GetOptions{})
				if err != nil && !apierrors.IsNotFound(err) {
					return err
				}
				if tc != nil {
					tidbClusters = append(tidbClusters, tc)
				}
			}
		}
		if len(stsToMigrate) <= 0 {
			klog.Infof("Upgrader: found 0 Kubernetes StatefulSets owned by TidbCluster, nothing need to do")
			return nil
		}
		klog.Infof("Upgrader: %d Kubernetes Statfulsets owned by TidbCluster should be migrated to Advanced Statefulsets", len(stsToMigrate))
		// Check if relavant TidbClusters have delete slots annotations set.
		for _, tc := range tidbClusters {
			// Existing delete slots annotations must be removed first. This is
			// a safety check to ensure no pods are affected in upgrading
			// process.
			if anns := deleteSlotAnns(tc); len(anns) > 0 {
				return fmt.Errorf("Upgrader: TidbCluster %s/%s has delete slot annotations %v, please remove them before enabling AdvancedStatefulSet feature", tc.Namespace, tc.Name, anns)
			}
		}
		klog.Infof("Upgrader: found %d Kubernetes StatefulSets owned by TidbCluster, trying to migrate one by one", len(stsToMigrate))
		for _, sts := range stsToMigrate {
			_, err := helper.Upgrade(u.kubeCli, u.asCli, &sts)
			if err != nil {
				return err
			}
			klog.Infof("Upgrader: successfully migrated Kubernetes StatefulSet %s/%s", sts.Namespace, sts.Name)
		}
	} else {
		if isSupported, err := utildiscovery.IsAPIGroupSupported(u.kubeCli.Discovery(), asappsv1.GroupName); err != nil {
			return err
		} else if !isSupported {
			klog.Infof("Upgrader: APIGroup %s is not registered, skip checking Advanced Statfulset", asappsv1.GroupName)
			return nil
		}
		stsList, err := u.asCli.AppsV1().StatefulSets(u.ns).List(metav1.ListOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				klog.Infof("Upgrader: Kubernetes server does't have Advanced StatefulSets resources, skip to revert")
				return nil
			}
			return err
		}
		stsToMigrate := make([]asappsv1.StatefulSet, 0)
		for _, sts := range stsList.Items {
			if ok, _ := util.IsOwnedByTidbCluster(&sts); ok {
				stsToMigrate = append(stsToMigrate, sts)
			}
		}
		if len(stsToMigrate) <= 0 {
			klog.Infof("Upgrader: found %d Advanced StatefulSets owned by TidbCluster, nothing need to do", len(stsToMigrate))
			return nil
		}
		// The upgrader cannot migrate Advanced StatefulSets to Kubernetes
		// StatefulSets automatically right now.
		// TODO try our best to allow users to revert AdvancedStatefulSet feature automaticaly
		return fmt.Errorf("Upgrader: found %d Advanced StatefulSets owned by TidbCluster, the operator cann't run with AdvancedStatefulSet feature disabled", len(stsToMigrate))
	}
	return nil
}

func deleteSlotAnns(tc *v1alpha1.TidbCluster) map[string]string {
	anns := make(map[string]string)
	if tc == nil || tc.Annotations == nil {
		return anns
	}
	for _, key := range []string{label.AnnPDDeleteSlots, label.AnnTiDBDeleteSlots, label.AnnTiKVDeleteSlots} {
		if v, ok := tc.Annotations[key]; ok {
			anns[key] = v
		}
	}
	return anns
}

func NewUpgrader(kubeCli kubernetes.Interface, cli versioned.Interface, asCli asclientset.Interface, ns string) Interface {
	return &upgrader{kubeCli, cli, asCli, ns}
}
