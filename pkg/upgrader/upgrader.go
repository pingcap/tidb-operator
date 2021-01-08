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

	"github.com/pingcap/tidb-operator/pkg/util/conversion"

	kruiseappsv1beta1 "github.com/openkruise/kruise-api/apps/v1beta1"
	kruiseclientset "github.com/openkruise/kruise-api/client/clientset/versioned"
	asappsv1 "github.com/pingcap/advanced-statefulset/client/apis/apps/v1"
	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	asclientset "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned"
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
	kubeCli       kubernetes.Interface
	cli           versioned.Interface
	asCli         asclientset.Interface
	kruiseCli     kruiseclientset.Interface
	kruiseEnabled bool
	ns            string
}

var _ Interface = &upgrader{}

func (u *upgrader) Upgrade() error {
	var stsToMigrate []appsv1.StatefulSet
	var astsToMigrate []asappsv1.StatefulSet
	var kruiseStsToMigrate []kruiseappsv1beta1.StatefulSet

	stsToMigrate, err := u.getStsToMigrate()
	if err != nil {
		return err
	}
	astsToMigrate, err = u.getAstsToMigrate()
	if err != nil {
		return err
	}
	kruiseStsToMigrate, err = u.getKruiseAstsToMigrate()
	if err != nil {
		return err
	}

	if features.DefaultFeatureGate.Enabled(features.AdvancedStatefulSet) {
		if !u.kruiseEnabled && len(kruiseStsToMigrate) > 0 {
			// The upgrader cannot migrate Kruise Advanced StatefulSets to PingCAP Advanced StatefulSets
			// automatically right now.
			// TODO try our best to allow users to revert AdvancedStatefulSet feature automaticaly
			return fmt.Errorf("Upgrader: found %d Kruise Advanced StatefulSets owned by TidbCluster, the operator can't run with KruiseAdvancedStatefulSet feature disabled", len(kruiseStsToMigrate))
		}
		err = u.doUpgrade(stsToMigrate, astsToMigrate)
		if err != nil {
			return err
		}
	} else {
		isAstsSupported, err := utildiscovery.IsAPIGroupSupported(u.kubeCli.Discovery(), asappsv1.GroupName)
		if err != nil {
			return err
		}
		isKruiseSupported, err := utildiscovery.IsAPIGroupSupported(u.kubeCli.Discovery(), "apps.kruise.io")
		if err != nil {
			return err
		}
		if !isAstsSupported && !isKruiseSupported {
			klog.Infof("Upgrader: APIGroup %s and %s are not registered, skip checking Advanced Statfulset", asappsv1.GroupName, "apps.kruise.io")
			return nil
		}

		if len(astsToMigrate) <= 0 && len(kruiseStsToMigrate) <= 0 {
			klog.Infof("Upgrader: found %d PingCAP Advanced StatefulSets and %d kruise Advanced StatefulSets owned by TidbCluster, nothing need to do", len(astsToMigrate), len(kruiseStsToMigrate))
			return nil
		}
		// The upgrader cannot migrate Advanced StatefulSets to Kubernetes
		// StatefulSets automatically right now.
		// TODO try our best to allow users to revert AdvancedStatefulSet feature automaticaly
		return fmt.Errorf("Upgrader: found %d PingCAP Advanced StatefulSets and %d kruise Advanced StatefulSets owned by TidbCluster, the operator can't run with AdvancedStatefulSet feature disabled", len(astsToMigrate), len(kruiseStsToMigrate))
	}
	return nil
}

func (u *upgrader) getStsToMigrate() ([]appsv1.StatefulSet, error) {
	stsList, err := u.kubeCli.AppsV1().StatefulSets(u.ns).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	stsToMigrate := make([]appsv1.StatefulSet, 0)
	tidbClusters := make([]*v1alpha1.TidbCluster, 0)
	for _, sts := range stsList.Items {
		if ok, tcRef := util.IsOwnedByTidbCluster(&sts); ok {
			stsToMigrate = append(stsToMigrate, sts)
			tc, err := u.cli.PingcapV1alpha1().TidbClusters(sts.Namespace).Get(tcRef.Name, metav1.GetOptions{})
			if err != nil && !apierrors.IsNotFound(err) {
				return nil, err
			}
			if tc != nil {
				tidbClusters = append(tidbClusters, tc)
			}
		}
	}

	// Check if relavant TidbClusters have delete slots annotations set.
	for _, tc := range tidbClusters {
		// Existing delete slots annotations must be removed first. This is
		// a safety check to ensure no pods are affected in upgrading
		// process.
		if anns := deleteSlotAnns(tc); len(anns) > 0 {
			return nil, fmt.Errorf("Upgrader: TidbCluster %s/%s has delete slot annotations %v, please remove them before enabling AdvancedStatefulSet feature", tc.Namespace, tc.Name, anns)
		}
	}

	klog.Infof("Upgrader: found %d Kubernetes StatefulSets owned by TidbCluster", len(stsToMigrate))
	return stsToMigrate, nil
}

func (u *upgrader) getAstsToMigrate() ([]asappsv1.StatefulSet, error) {
	astsList, err := u.asCli.AppsV1().StatefulSets(u.ns).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	astsToMigrate := make([]asappsv1.StatefulSet, 0)
	for _, asts := range astsList.Items {
		if ok, _ := util.IsOwnedByTidbCluster(&asts); ok {
			astsToMigrate = append(astsToMigrate, asts)
		}
	}

	klog.Infof("Upgrader: found %d PingCAP AdvancedStatefulSets owned by TidbCluster", len(astsToMigrate))
	return astsToMigrate, nil
}

func (u *upgrader) getKruiseAstsToMigrate() ([]kruiseappsv1beta1.StatefulSet, error) {
	kruiseStsList, err := u.kruiseCli.AppsV1beta1().StatefulSets(u.ns).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	kruiseAstsToMigrate := make([]kruiseappsv1beta1.StatefulSet, 0)
	for _, kAsts := range kruiseStsList.Items {
		if ok, _ := util.IsOwnedByTidbCluster(&kAsts); ok {
			kruiseAstsToMigrate = append(kruiseAstsToMigrate, kAsts)
		}
	}

	klog.Infof("Upgrader: found %d OpenKruise AdvancedStatefulSets owned by TidbCluster", len(kruiseAstsToMigrate))
	return kruiseAstsToMigrate, nil

}

func (u *upgrader) doUpgrade(stsToMigrate []appsv1.StatefulSet, astsToMigrate []asappsv1.StatefulSet) error {
	if u.kruiseEnabled {
		klog.Infof("Upgrader: migrating Kubernetes StatefulSets to OpenKruise Advanced StatefulSets")
		for _, sts := range stsToMigrate {
			_, err := conversion.UpgradeFromSts(u.kubeCli, u.kruiseCli, &sts)
			if err != nil {
				return err
			}
			klog.Infof("Upgrader: successfully migrated from Kubernetes StatefulSet %s/%s to Kruise Advanced StatefulSet", sts.Namespace, sts.Name)
		}
		klog.Infof("Upgrader: migrating PingCAP Advanced StatefulSets to OpenKruise Advanced StatefulSets")
		for _, asts := range astsToMigrate {
			_, err := conversion.UpgradeFromAsts(u.kubeCli, u.asCli, u.kruiseCli, &asts)
			if err != nil {
				return err
			}
			klog.Infof("Upgrader: successfully migrated from PingCAP Advanced StatefulSet %s/%s to Kruise Advanced StatefulSet", asts.Namespace, asts.Name)
		}
	} else {
		klog.Infof("Upgrader: migrating Kubernetes StatefulSets to PingCAP Advanced StatefulSets")
		for _, sts := range stsToMigrate {
			_, err := helper.Upgrade(u.kubeCli, u.asCli, &sts)
			if err != nil {
				return err
			}
			klog.Infof("Upgrader: successfully migrated from Kubernetes StatefulSet %s/%s to PingCAP Advanced StatefulSet", sts.Namespace, sts.Name)
		}
	}
	return nil
}

func deleteSlotAnns(tc *v1alpha1.TidbCluster) map[string]string {
	anns := make(map[string]string)
	if tc == nil || tc.Annotations == nil {
		return anns
	}
	for _, key := range []string{label.AnnPDDeleteSlots, label.AnnTiDBDeleteSlots, label.AnnTiKVDeleteSlots, label.AnnTiFlashDeleteSlots} {
		if v, ok := tc.Annotations[key]; ok {
			anns[key] = v
		}
	}
	return anns
}

func NewUpgrader(kubeCli kubernetes.Interface, cli versioned.Interface, asCli asclientset.Interface, kruiseCli kruiseclientset.Interface, kruiseEnabled bool, ns string) Interface {
	return &upgrader{kubeCli, cli, asCli, kruiseCli, kruiseEnabled, ns}
}
