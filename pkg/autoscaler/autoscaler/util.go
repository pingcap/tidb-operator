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
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	operatorUtils "github.com/pingcap/tidb-operator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog"
	"k8s.io/utils/pointer"
)

var zeroQuantity = resource.MustParse("0")

// checkStsAutoScalingPrerequisites would check the sts status to ensure wouldn't happen during
// upgrading, scaling
func checkStsAutoScalingPrerequisites(set *appsv1.StatefulSet) bool {
	return !operatorUtils.IsStatefulSetUpgrading(set) && !operatorUtils.IsStatefulSetScaling(set)
}

// checkStsAutoScalingInterval would check whether there is enough interval duration between every two auto-scaling
func checkStsAutoScalingInterval(tac *v1alpha1.TidbClusterAutoScaler, intervalSeconds int32, memberType v1alpha1.MemberType) (bool, error) {
	lastAutoScalingTimestamp, existed := tac.Annotations[label.AnnTiDBLastAutoScalingTimestamp]
	if memberType == v1alpha1.TiKVMemberType {
		lastAutoScalingTimestamp, existed = tac.Annotations[label.AnnTiKVLastAutoScalingTimestamp]
	}
	if !existed {
		return true, nil
	}
	t, err := strconv.ParseInt(lastAutoScalingTimestamp, 10, 64)
	if err != nil {
		return false, fmt.Errorf("tac[%s/%s] parse last auto-scaling timestamp failed,err:%v", tac.Namespace, tac.Name, err)
	}
	if intervalSeconds > int32(time.Since(time.Unix(t, 0)).Seconds()) {
		return false, nil
	}
	return true, nil
}

// limitTargetReplicas would limit the calculated target replicas to ensure the min/max Replicas
func limitTargetReplicas(targetReplicas int32, tac *v1alpha1.TidbClusterAutoScaler, memberType v1alpha1.MemberType) int32 {
	var min, max int32
	switch memberType {
	case v1alpha1.TiKVMemberType:
		min, max = *tac.Spec.TiKV.MinReplicas, tac.Spec.TiKV.MaxReplicas
	case v1alpha1.TiDBMemberType:
		min, max = *tac.Spec.TiDB.MinReplicas, tac.Spec.TiDB.MaxReplicas
	default:
		return targetReplicas
	}
	if targetReplicas > max {
		return max
	}
	if targetReplicas < min {
		return min
	}
	return targetReplicas
}

func defaultResources(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler, component v1alpha1.MemberType) {
	typ := fmt.Sprintf("default_%s", component.String())
	resource := v1alpha1.AutoResource{}
	var requests corev1.ResourceList

	switch component {
	case v1alpha1.TiDBMemberType:
		requests = tc.Spec.TiDB.Requests
	case v1alpha1.TiKVMemberType:
		requests = tc.Spec.TiKV.Requests
	}

	for res, v := range requests {
		switch res {
		case corev1.ResourceCPU:
			resource.CPU = v
		case corev1.ResourceMemory:
			resource.Memory = v
		case corev1.ResourceStorage:
			resource.Storage = v
		}
	}

	switch component {
	case v1alpha1.TiDBMemberType:
		if tac.Spec.TiDB.Resources == nil {
			tac.Spec.TiDB.Resources = make(map[string]v1alpha1.AutoResource)
		}
		tac.Spec.TiDB.Resources[typ] = resource
	case v1alpha1.TiKVMemberType:
		if tac.Spec.TiKV.Resources == nil {
			tac.Spec.TiKV.Resources = make(map[string]v1alpha1.AutoResource)
		}
		tac.Spec.TiKV.Resources[typ] = resource
	}
}

func defaultResourceTypes(tac *v1alpha1.TidbClusterAutoScaler, rule *v1alpha1.AutoRule, component v1alpha1.MemberType) {
	resources := getSpecResources(tac, component)
	if len(rule.ResourceTypes) == 0 {
		for name := range resources {
			rule.ResourceTypes = append(rule.ResourceTypes, name)
		}
	}
}

func getBasicAutoScalerSpec(tac *v1alpha1.TidbClusterAutoScaler, component v1alpha1.MemberType) *v1alpha1.BasicAutoScalerSpec {
	switch component {
	case v1alpha1.TiDBMemberType:
		return &tac.Spec.TiDB.BasicAutoScalerSpec
	case v1alpha1.TiKVMemberType:
		return &tac.Spec.TiKV.BasicAutoScalerSpec
	}
	return nil
}

func getSpecResources(tac *v1alpha1.TidbClusterAutoScaler, component v1alpha1.MemberType) map[string]v1alpha1.AutoResource {
	switch component {
	case v1alpha1.TiDBMemberType:
		if tac.Spec.TiDB != nil {
			return tac.Spec.TiDB.Resources
		}
	case v1alpha1.TiKVMemberType:
		if tac.Spec.TiKV != nil {
			return tac.Spec.TiKV.Resources
		}
	}
	return nil
}

func defaultBasicAutoScaler(tac *v1alpha1.TidbClusterAutoScaler, component v1alpha1.MemberType) {
	spec := getBasicAutoScalerSpec(tac, component)

	if spec.MinReplicas == nil {
		spec.MinReplicas = pointer.Int32Ptr(1)
	}
	if spec.ScaleOutIntervalSeconds == nil {
		spec.ScaleOutIntervalSeconds = pointer.Int32Ptr(300)
	}
	if spec.ScaleInIntervalSeconds == nil {
		spec.ScaleInIntervalSeconds = pointer.Int32Ptr(500)
	}
	// If ExternalEndpoint is not provided, we would set default metrics
	if spec.External == nil && spec.MetricsTimeDuration == nil {
		spec.MetricsTimeDuration = pointer.StringPtr("3m")
	}

	if spec.External != nil {
		return
	}

	for res, rule := range spec.Rules {
		if res == corev1.ResourceCPU {
			if rule.MinThreshold == nil {
				rule.MinThreshold = pointer.Float64Ptr(0.1)
			}
		}
		defaultResourceTypes(tac, &rule, component)
		spec.Rules[res] = rule
	}
}

// If the minReplicas not set, the default value would be 1
// If the Metrics not set, the default metric will be set to 80% average CPU utilization.
// defaultTAC would default the omitted value
func defaultTAC(tac *v1alpha1.TidbClusterAutoScaler, tc *v1alpha1.TidbCluster) {
	if tac.Annotations == nil {
		tac.Annotations = map[string]string{}
	}

	// Construct default resource
	if tac.Spec.TiKV != nil && tac.Spec.TiKV.External == nil && len(tac.Spec.TiKV.Resources) == 0 {
		defaultResources(tc, tac, v1alpha1.TiKVMemberType)
	}

	if tac.Spec.TiDB != nil && tac.Spec.TiDB.External == nil && len(tac.Spec.TiDB.Resources) == 0 {
		defaultResources(tc, tac, v1alpha1.TiDBMemberType)
	}

	if tidb := tac.Spec.TiDB; tidb != nil {
		defaultBasicAutoScaler(tac, v1alpha1.TiDBMemberType)
	}

	if tikv := tac.Spec.TiKV; tikv != nil {
		defaultBasicAutoScaler(tac, v1alpha1.TiKVMemberType)
		for id, m := range tikv.Metrics {
			if m.Resource == nil || m.Resource.Name != corev1.ResourceStorage {
				continue
			}
			if m.LeastStoragePressurePeriodSeconds == nil {
				m.LeastStoragePressurePeriodSeconds = pointer.Int64Ptr(300)
			}
			if m.LeastRemainAvailableStoragePercent == nil {
				m.LeastRemainAvailableStoragePercent = pointer.Int64Ptr(10)
			}
			tikv.Metrics[id] = m
		}
	}

	if monitor := tac.Spec.Monitor; monitor != nil && len(monitor.Namespace) < 1 {
		monitor.Namespace = tac.Namespace
	}
}

func validateBasicAutoScalerSpec(tac *v1alpha1.TidbClusterAutoScaler, component v1alpha1.MemberType) error {
	spec := getBasicAutoScalerSpec(tac, component)

	if spec.External != nil {
		return nil
	}

	if len(spec.Rules) == 0 {
		return fmt.Errorf("no rules defined for component %s in %s/%s", component.String(), tac.Namespace, tac.Name)
	}
	resources := getSpecResources(tac, component)

	if component == v1alpha1.TiKVMemberType {
		for name, res := range resources {
			if res.Storage.Cmp(zeroQuantity) == 0 {
				return fmt.Errorf("resource %s defined for tikv does not have storage in %s/%s", name, tac.Namespace, tac.Name)
			}
		}
	}

	acceptableResources := map[corev1.ResourceName]struct{}{
		corev1.ResourceCPU: {},
	}

	checkCommon := func(res corev1.ResourceName, rule v1alpha1.AutoRule) error {
		if _, ok := acceptableResources[res]; !ok {
			return fmt.Errorf("unknown resource type %s of %s in %s/%s", res.String(), component.String(), tac.Namespace, tac.Name)
		}
		if rule.MaxThreshold > 1.0 || rule.MaxThreshold < 0.0 {
			return fmt.Errorf("max_threshold (%v) should be between 0 and 1 for rule %s of %s in %s/%s", rule.MaxThreshold, res, component.String(), tac.Namespace, tac.Name)
		}
		if len(rule.ResourceTypes) == 0 {
			return fmt.Errorf("no resources provided for rule %s of %s in %s/%s", res, component.String(), tac.Namespace, tac.Name)
		}
		for _, resType := range rule.ResourceTypes {
			if _, ok := resources[resType]; !ok {
				return fmt.Errorf("unknown resource %s for %s in %s/%s", resType, component.String(), tac.Namespace, tac.Name)
			}
		}
		return nil
	}

	for res, rule := range spec.Rules {
		if err := checkCommon(res, rule); err != nil {
			return err
		}

		switch res {
		case corev1.ResourceCPU:
			if *rule.MinThreshold > 1.0 || *rule.MinThreshold < 0.0 {
				return fmt.Errorf("min_threshold (%v) should be between 0 and 1 for rule %s of %s in %s/%s", *rule.MinThreshold, res, component.String(), tac.Namespace, tac.Name)
			}
			if *rule.MinThreshold > rule.MaxThreshold {
				return fmt.Errorf("min_threshold (%v) > max_threshold (%v) for cpu rule of %s in %s/%s", *rule.MinThreshold, rule.MaxThreshold, component.String(), tac.Namespace, tac.Name)
			}
		}
	}

	return nil
}

func validateTAC(tac *v1alpha1.TidbClusterAutoScaler) error {
	if tac.Spec.TiDB != nil && tac.Spec.TiDB.External == nil && len(tac.Spec.TiDB.Resources) == 0 {
		return fmt.Errorf("no resources provided for tidb in %s/%s", tac.Namespace, tac.Name)
	}

	if tac.Spec.TiKV != nil && tac.Spec.TiKV.External == nil && len(tac.Spec.TiKV.Resources) == 0 {
		return fmt.Errorf("no resources provided for tikv in %s/%s", tac.Namespace, tac.Name)
	}

	if tidb := tac.Spec.TiDB; tidb != nil {
		err := validateBasicAutoScalerSpec(tac, v1alpha1.TiDBMemberType)
		if err != nil {
			return err
		}
	}

	if tikv := tac.Spec.TiKV; tikv != nil {
		err := validateBasicAutoScalerSpec(tac, v1alpha1.TiKVMemberType)
		if err != nil {
			return err
		}
	}

	return nil
}

func genMetricsEndpoint(tac *v1alpha1.TidbClusterAutoScaler) (string, error) {
	if tac.Spec.MetricsUrl == nil && tac.Spec.Monitor == nil {
		return "", fmt.Errorf("tac[%s/%s] metrics url or monitor should be defined explicitly", tac.Namespace, tac.Name)
	}
	if tac.Spec.MetricsUrl != nil {
		return *tac.Spec.MetricsUrl, nil
	}
	return fmt.Sprintf("http://%s-prometheus.%s.svc:9090", tac.Spec.Monitor.Name, tac.Spec.Monitor.Namespace), nil
}

func autoscalerToStrategy(tac *v1alpha1.TidbClusterAutoScaler, component v1alpha1.MemberType) *pdapi.Strategy {
	resources := getSpecResources(tac, component)
	strategy := &pdapi.Strategy{
		Resources: make([]*pdapi.Resource, 0, len(resources)),
	}

	for typ, res := range resources {
		resource := &pdapi.Resource{
			CPU:          res.CPU.AsDec().UnscaledBig().Uint64(),
			Memory:       res.Memory.AsDec().UnscaledBig().Uint64(),
			Storage:      res.Storage.AsDec().UnscaledBig().Uint64(),
			ResourceType: typ,
		}
		if res.Count != nil {
			count := uint64(*res.Count)
			resource.Count = &count
		}
		strategy.Resources = append(strategy.Resources, resource)
	}

	switch component {
	case v1alpha1.TiDBMemberType:
		strategy.Rules = []*pdapi.Rule{autoRulesToStrategyRule(component.String(), tac.Spec.TiDB.Rules)}
	case v1alpha1.TiKVMemberType:
		strategy.Rules = []*pdapi.Rule{autoRulesToStrategyRule(component.String(), tac.Spec.TiKV.Rules)}
	}

	return strategy
}

func autoRulesToStrategyRule(component string, rules map[corev1.ResourceName]v1alpha1.AutoRule) *pdapi.Rule {
	result := &pdapi.Rule{
		Component: component,
	}
	for res, rule := range rules {
		switch res {
		case corev1.ResourceCPU:
			// For CPU rule, users should both specify max_threshold and min_threshold
			// Defaulting and validating make sure that the min_threshold is set
			result.CPURule = &pdapi.CPURule{
				MaxThreshold:  rule.MaxThreshold,
				MinThreshold:  *rule.MinThreshold,
				ResourceTypes: rule.ResourceTypes,
			}
		case corev1.ResourceStorage:
			// For storage rule, users need only set the max_threshold and we convert it to min_threshold for PD
			result.StorageRule = &pdapi.StorageRule{
				MinThreshold:  1.0 - rule.MaxThreshold,
				ResourceTypes: rule.ResourceTypes,
			}
		default:
			klog.Warningf("unknown resource type %v", res.String())
		}
	}
	return result
}

const autoClusterPrefix = "auto-"

func genAutoClusterName(tas *v1alpha1.TidbClusterAutoScaler, component string, labels map[string]string, resource v1alpha1.AutoResource) (string, error) {
	seed := map[string]interface{}{
		"namespace": tas.Namespace,
		"tas":       tas.Name,
		"component": component,
		"cpu":       resource.CPU.AsDec().UnscaledBig().Uint64(),
		"storage":   resource.Storage.AsDec().UnscaledBig().Uint64(),
		"memory":    resource.Memory.AsDec().UnscaledBig().Uint64(),
		"labels":    labels,
	}
	marshaled, err := json.Marshal(seed)
	if err != nil {
		return "", err
	}

	return autoClusterPrefix + v1alpha1.HashContents(marshaled), nil
}
