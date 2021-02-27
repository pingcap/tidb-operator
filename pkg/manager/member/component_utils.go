// Copyright 2021 PingCAP, Inc.
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

package member

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/pkg/util"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
)

func getComponentDesiredOrdinals(context *ComponentContext) sets.Int32 {
	tc := context.tc
	component := context.component

	switch component {
	case label.PDLabelVal:
		return tc.PDStsDesiredOrdinals(true)
	case label.TiKVLabelVal:
		return tc.TiKVStsDesiredOrdinals(true)
	case label.TiDBLabelVal:
		return tc.TiDBStsDesiredOrdinals(true)
	case label.TiFlashLabelVal:
		return tc.TiFlashStsDesiredOrdinals(true)
	}

	return sets.Int32{}
}

func getComponentLabel(context *ComponentContext, instanceName string) label.Label {
	component := context.component

	var componentLabel label.Label
	switch component {
	case label.PDLabelVal:
		componentLabel = label.New().Instance(instanceName).PD()
	case label.TiKVLabelVal:
		componentLabel = label.New().Instance(instanceName).TiKV()
	case label.TiFlashLabelVal:
		componentLabel = label.New().Instance(instanceName).TiFlash()
	case label.TiDBLabelVal:
		componentLabel = label.New().Instance(instanceName).TiDB()
	case label.TiCDCLabelVal:
		componentLabel = label.New().Instance(instanceName).TiCDC()
	case label.PumpLabelVal:
		componentLabel = label.New().Instance(instanceName).Pump()
	}
	return componentLabel
}

func getComponentUpdataRevision(context *ComponentContext) string {
	tc := context.tc
	component := context.component

	var componentUpdateRevision string
	switch component {
	case label.PDLabelVal:
		componentUpdateRevision = tc.Status.PD.StatefulSet.UpdateRevision
	case label.TiKVLabelVal:
		componentUpdateRevision = tc.Status.TiKV.StatefulSet.UpdateRevision
	case label.TiFlashLabelVal:
		componentUpdateRevision = tc.Status.TiFlash.StatefulSet.UpdateRevision
	case label.TiDBLabelVal:
		componentUpdateRevision = tc.Status.TiDB.StatefulSet.UpdateRevision
	case label.TiCDCLabelVal:
		componentUpdateRevision = tc.Status.TiCDC.StatefulSet.UpdateRevision
	case label.PumpLabelVal:
		componentUpdateRevision = tc.Status.Pump.StatefulSet.UpdateRevision
	}
	return componentUpdateRevision
}

func getComponentMemberName(context *ComponentContext) string {
	tc := context.tc
	component := context.component

	tcName := tc.GetName()
	var componentMemberName string
	switch component {
	case label.PDLabelVal:
		componentMemberName = controller.PDMemberName(tcName)
	case label.TiKVLabelVal:
		componentMemberName = controller.TiKVMemberName(tcName)
	case label.TiFlashLabelVal:
		componentMemberName = controller.TiFlashMemberName(tcName)
	case label.TiDBLabelVal:
		componentMemberName = controller.TiDBMemberName(tcName)
	case label.TiCDCLabelVal:
		componentMemberName = controller.TiCDCMemberName(tcName)
	case label.PumpLabelVal:
		componentMemberName = controller.PumpMemberName(tcName)
	}
	return componentMemberName
}

func syncNewComponentStatefulset(context *ComponentContext) error {
	tc := context.tc
	component := context.component

	switch component {
	case label.PDLabelVal:
		tc.Status.PD.StatefulSet = &apps.StatefulSetStatus{}
	case label.TiKVLabelVal:
		tc.Status.TiKV.StatefulSet = &apps.StatefulSetStatus{}
	case label.TiDBLabelVal:
		tc.Status.TiDB.StatefulSet = &apps.StatefulSetStatus{}
	case label.TiFlashLabelVal:
		tc.Status.TiFlash.StatefulSet = &apps.StatefulSetStatus{}
	case label.TiCDCLabelVal:
		tc.Status.TiCDC.StatefulSet = &apps.StatefulSetStatus{}
	case label.PumpLabelVal:
		tc.Status.Pump.StatefulSet = &apps.StatefulSetStatus{}
	}
	return nil
}

func syncExistedComponentStatefulset(context *ComponentContext, set *apps.StatefulSet) error {
	tc := context.tc
	component := context.component
	switch component {
	case label.PDLabelVal:
		tc.Status.PD.StatefulSet = &set.Status
	case label.TiKVLabelVal:
		tc.Status.TiKV.StatefulSet = &set.Status
	case label.TiFlashLabelVal:
		tc.Status.TiFlash.StatefulSet = &set.Status
	case label.TiDBLabelVal:
		tc.Status.TiDB.StatefulSet = &set.Status
	case label.TiCDCLabelVal:
		tc.Status.TiCDC.StatefulSet = &set.Status
	case label.PumpLabelVal:
		tc.Status.Pump.StatefulSet = &set.Status
	}
	return nil
}

func syncComponentImage(context *ComponentContext, set *apps.StatefulSet) error {
	tc := context.tc
	component := context.component

	switch component {
	case label.PDLabelVal:
		tc.Status.PD.Image = ""
		c := filterContainer(set, "pd")
		if c != nil {
			tc.Status.PD.Image = c.Image
		}
	case label.TiKVLabelVal:
		tc.Status.TiKV.Image = ""
		c := filterContainer(set, "tikv")
		if c != nil {
			tc.Status.TiKV.Image = c.Image
		}
	case label.TiFlashLabelVal:
		tc.Status.TiFlash.Image = ""
		c := filterContainer(set, "tiflash")
		if c != nil {
			tc.Status.TiFlash.Image = c.Image
		}
	case label.TiDBLabelVal:
		tc.Status.TiDB.Image = ""
		c := filterContainer(set, "tidb")
		if c != nil {
			tc.Status.TiDB.Image = c.Image
		}
	}
	return nil
}

func getComponentMemberType(context *ComponentContext) v1alpha1.MemberType {
	component := context.component

	var memberType v1alpha1.MemberType
	switch component {
	case label.PDLabelVal:
		memberType = v1alpha1.PDMemberType
	case label.TiKVLabelVal:
		memberType = v1alpha1.TiKVMemberType
	case label.TiFlashLabelVal:
		memberType = v1alpha1.TiFlashMemberType
	case label.TiDBLabelVal:
		memberType = v1alpha1.TiDBMemberType
	}

	return memberType
}

func ComponentClusterVersionGreaterThanOrEqualTo4(version string) (bool, error) {
	v, err := semver.NewVersion(version)
	if err != nil {
		return true, err
	}

	return v.Major() >= 4, nil
}

func collectUnjoinedPDMembers(context *ComponentContext, set *apps.StatefulSet, pdStatus map[string]v1alpha1.PDMember) error {
	tc := context.tc
	dependencies := context.dependencies

	podSelector, podSelectErr := metav1.LabelSelectorAsSelector(set.Spec.Selector)
	if podSelectErr != nil {
		return podSelectErr
	}
	pods, podErr := dependencies.PodLister.Pods(tc.Namespace).List(podSelector)
	if podErr != nil {
		return fmt.Errorf("collectUnjoinedMembers: failed to list pods for cluster %s/%s, selector %s, error %v", tc.GetNamespace(), tc.GetName(), set.Spec.Selector, podErr)
	}
	for _, pod := range pods {
		var joined = false
		for pdName := range pdStatus {
			ordinal, err := util.GetOrdinalFromPodName(pod.Name)
			if err != nil {
				return fmt.Errorf("unexpected pod name %q: %v", pod.Name, err)
			}
			if strings.EqualFold(PdName(tc.Name, ordinal, tc.Namespace, tc.Spec.ClusterDomain), pdName) {
				joined = true
				break
			}
		}
		if !joined {
			if tc.Status.PD.UnjoinedMembers == nil {
				tc.Status.PD.UnjoinedMembers = map[string]v1alpha1.UnjoinedMember{}
			}
			ordinal, err := util.GetOrdinalFromPodName(pod.Name)
			if err != nil {
				return err
			}
			pvcName := ordinalPVCName(v1alpha1.PDMemberType, controller.PDMemberName(tc.Name), ordinal)
			pvc, err := dependencies.PVCLister.PersistentVolumeClaims(tc.Namespace).Get(pvcName)
			if err != nil {
				return fmt.Errorf("collectUnjoinedMembers: failed to get pvc %s of cluster %s/%s, error %v", pvcName, tc.GetNamespace(), tc.GetName(), err)
			}
			tc.Status.PD.UnjoinedMembers[pod.Name] = v1alpha1.UnjoinedMember{
				PodName:   pod.Name,
				PVCUID:    pvc.UID,
				CreatedAt: metav1.Now(),
			}
		} else {
			if tc.Status.PD.UnjoinedMembers != nil {
				delete(tc.Status.PD.UnjoinedMembers, pod.Name)
			}
		}
	}
	return nil
}

// ComponentShouldRecover checks whether we should perform recovery operation.
func ComponentShouldRecover(context *ComponentContext) bool {
	tc := context.tc
	component := context.component
	dependencies := context.dependencies

	var pdMembers map[string]v1alpha1.PDMember
	var stores map[string]v1alpha1.TiKVStore
	var tidbMembers map[string]v1alpha1.TiDBMember
	var pdFailureMembers map[string]v1alpha1.PDFailureMember
	var failureStores map[string]v1alpha1.TiKVFailureStore
	var tidbFailureMembers map[string]v1alpha1.TiDBFailureMember
	var ordinals sets.Int32
	var podPrefix string

	switch component {
	case label.TiKVLabelVal:
		stores = tc.Status.TiKV.Stores
		failureStores = tc.Status.TiKV.FailureStores
		if failureStores == nil {
			return false
		}
		ordinals = tc.TiKVStsDesiredOrdinals(true)
		podPrefix = controller.TiKVMemberName(tc.Name)
	case label.TiFlashLabelVal:
		stores = tc.Status.TiFlash.Stores
		failureStores = tc.Status.TiFlash.FailureStores
		if failureStores == nil {
			return false
		}
		ordinals = tc.TiFlashStsDesiredOrdinals(true)
		podPrefix = controller.TiFlashMemberName(tc.Name)
	case label.PDLabelVal:
		pdMembers = tc.Status.PD.Members
		pdFailureMembers = tc.Status.PD.FailureMembers
		if pdFailureMembers == nil {
			return false
		}
		ordinals = tc.PDStsDesiredOrdinals(true)
		podPrefix = controller.PDMemberName(tc.Name)
	case label.TiDBLabelVal:
		tidbMembers = tc.Status.TiDB.Members
		tidbFailureMembers = tc.Status.TiDB.FailureMembers
		if tidbFailureMembers == nil {
			return false
		}
		ordinals = tc.TiDBStsDesiredOrdinals(true)
		podPrefix = controller.TiDBMemberName(tc.Name)
	default:
		klog.Warningf("Unexpected component %s for %s/%s in shouldRecover", component, tc.Namespace, tc.Name)
		return false
	}

	// If all desired replicas (excluding failover pods) of tidb cluster are
	// healthy, we can perform our failover recovery operation.
	// Note that failover pods may fail (e.g. lack of resources) and we don't care
	// about them because we're going to delete them.
	for ordinal := range ordinals {
		name := fmt.Sprintf("%s-%d", podPrefix, ordinal)
		pod, err := dependencies.PodLister.Pods(tc.Namespace).Get(name)
		if err != nil {
			klog.Errorf("pod %s/%s does not exist: %v", tc.Namespace, name, err)
			return false
		}
		if !podutil.IsPodReady(pod) {
			return false
		}
		var exist bool
		switch component {
		case label.TiKVLabelVal, label.TiFlashLabelVal:
			for _, v := range stores {
				if v.PodName == pod.Name {
					exist = true
					if v.State != v1alpha1.TiKVStateUp {
						return false
					}
				}
			}
		case label.PDLabelVal:
			for pdName, pdMember := range pdMembers {
				if strings.Split(pdName, ".")[0] == pod.Name {
					if !pdMember.Health {
						return false
					}
					exist = true
					break
				}
			}
		case label.TiDBLabelVal:
			status, ok := tidbMembers[pod.Name]
			if !ok || !status.Health {
				return false
			}
		}

		if !exist && component != label.TiDBLabelVal {
			return false
		}
	}
	return true
}

func getTiFlashStore(store *pdapi.StoreInfo) *v1alpha1.TiKVStore {
	if store.Store == nil || store.Status == nil {
		return nil
	}
	storeID := fmt.Sprintf("%d", store.Store.GetId())
	ip := strings.Split(store.Store.GetAddress(), ":")[0]
	podName := strings.Split(ip, ".")[0]

	return &v1alpha1.TiKVStore{
		ID:                storeID,
		PodName:           podName,
		IP:                ip,
		LeaderCount:       int32(store.Status.LeaderCount),
		State:             store.Store.StateName,
		LastHeartbeatTime: metav1.Time{Time: store.Status.LastHeartbeatTS},
	}
}

func setStoreLabelsForTiKV(context *ComponentContext) (int, error) {
	tc := context.tc
	dependencies := context.dependencies

	ns := tc.GetNamespace()
	// for unit test
	setCount := 0

	if !tc.TiKVBootStrapped() {
		klog.Infof("TiKV of Cluster %s/%s is not bootstrapped yet, no need to set store labels", tc.Namespace, tc.Name)
		return setCount, nil
	}

	pdCli := controller.GetPDClient(dependencies.PDControl, tc)
	storesInfo, err := pdCli.GetStores()
	if err != nil {
		return setCount, err
	}

	config, err := pdCli.GetConfig()
	if err != nil {
		return setCount, err
	}

	locationLabels := []string(config.Replication.LocationLabels)
	if locationLabels == nil {
		return setCount, nil
	}

	pattern, err := regexp.Compile(fmt.Sprintf(tikvStoreLimitPattern, tc.Name, tc.Name, tc.Namespace, controller.FormatClusterDomainForRegex(tc.Spec.ClusterDomain)))
	if err != nil {
		return -1, err
	}
	for _, store := range storesInfo.Stores {
		// In theory, the external tikv can join the cluster, and the operator would only manage the internal tikv.
		// So we check the store owner to make sure it.
		if store.Store != nil && !pattern.Match([]byte(store.Store.Address)) {
			continue
		}
		status := getTiKVStore(store)
		if status == nil {
			continue
		}
		podName := status.PodName

		pod, err := dependencies.PodLister.Pods(ns).Get(podName)
		if err != nil {
			return setCount, fmt.Errorf("setStoreLabelsForTiKV: failed to get pods %s for cluster %s/%s, error: %s", podName, ns, tc.GetName(), err)
		}

		nodeName := pod.Spec.NodeName
		ls, err := getNodeLabels(context, nodeName, locationLabels)
		if err != nil || len(ls) == 0 {
			klog.Warningf("node: [%s] has no node labels, skipping set store labels for Pod: [%s/%s]", nodeName, ns, podName)
			continue
		}

		if !storeLabelsEqualNodeLabels(store.Store.Labels, ls) {
			set, err := pdCli.SetStoreLabels(store.Store.Id, ls)
			if err != nil {
				msg := fmt.Sprintf("failed to set labels %v for store (id: %d, pod: %s/%s): %v ",
					ls, store.Store.Id, ns, podName, err)
				dependencies.Recorder.Event(tc, corev1.EventTypeWarning, FailedSetStoreLabels, msg)
				continue
			}
			if set {
				setCount++
				klog.Infof("pod: [%s/%s] set labels: %v successfully", ns, podName, ls)
			}
		}
	}

	return setCount, nil
}

func getNodeLabels(context *ComponentContext, nodeName string, storeLabels []string) (map[string]string, error) {
	dependencies := context.dependencies

	node, err := dependencies.NodeLister.Get(nodeName)
	if err != nil {
		return nil, err
	}
	labels := map[string]string{}
	ls := node.GetLabels()
	for _, storeLabel := range storeLabels {
		if value, found := ls[storeLabel]; found {
			labels[storeLabel] = value
			continue
		}

		// TODO after pd supports storeLabel containing slash character, these codes should be deleted
		if storeLabel == "host" {
			if host, found := ls[corev1.LabelHostname]; found {
				labels[storeLabel] = host
			}
		}

	}
	return labels, nil
}

func storeLabelsEqualNodeLabels(storeLabels []*metapb.StoreLabel, nodeLabels map[string]string) bool {
	ls := map[string]string{}
	for _, label := range storeLabels {
		key := label.GetKey()
		if _, ok := nodeLabels[key]; ok {
			val := label.GetValue()
			ls[key] = val
		}
	}
	return reflect.DeepEqual(ls, nodeLabels)
}
