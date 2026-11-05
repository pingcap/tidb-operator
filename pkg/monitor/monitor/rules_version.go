// Copyright 2026 PingCAP, Inc.
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

package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (m *MonitorManager) syncMonitorRulesVersion(tm *v1alpha1.TidbMonitor) error {
	version := getAlertManagerRulesVersion(tm)
	if tm.Spec.AlertManagerRulesVersion == nil && tm.Annotations[label.AnnMonitorRulesVersion] == "" {
		// Read the API directly: an informer cache miss must not reset an existing directory.
		sets, err := m.deps.KubeClientset.AppsV1().StatefulSets(tm.Namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return err
		}
		deployments, err := m.deps.KubeClientset.AppsV1().Deployments(tm.Namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return err
		}
		observed := ""
		observe := func(object metav1.Object, containers []corev1.Container) error {
			if !metav1.IsControlledBy(object, tm) {
				return nil
			}
			value, err := reloaderRulesVersion(containers)
			if err != nil {
				return fmt.Errorf("recover rules version from %s: %w; set spec.alertManagerRulesVersion explicitly", object.GetName(), err)
			}
			if observed != "" && observed != value {
				return fmt.Errorf("monitor workloads use different rules versions; set spec.alertManagerRulesVersion explicitly")
			}
			observed = value
			return nil
		}
		for i := range sets.Items {
			if err := observe(&sets.Items[i], sets.Items[i].Spec.Template.Spec.Containers); err != nil {
				return err
			}
		}
		for i := range deployments.Items {
			if err := observe(&deployments.Items[i], deployments.Items[i].Spec.Template.Spec.Containers); err != nil {
				return err
			}
		}
		if observed != "" {
			version = observed
		} else if tm.Status.StatefulSet != nil || tm.Status.DeploymentStorageStatus != nil {
			return fmt.Errorf("previous monitor workload is missing and its rules version is unknown; set spec.alertManagerRulesVersion explicitly")
		}
	}
	if tm.Annotations[label.AnnMonitorRulesVersion] == version {
		return nil
	}
	patch, err := json.Marshal(map[string]interface{}{"metadata": map[string]interface{}{
		"resourceVersion": tm.ResourceVersion,
		"annotations":     map[string]string{label.AnnMonitorRulesVersion: version},
	}})
	if err != nil {
		return err
	}
	updated, err := m.deps.Clientset.PingcapV1alpha1().TidbMonitors(tm.Namespace).Patch(context.TODO(), tm.Name, types.MergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("record monitor rules version before rollout: %w", err)
	}
	tm.Annotations = updated.Annotations
	tm.ResourceVersion = updated.ResourceVersion
	return nil
}

func reloaderRulesVersion(containers []corev1.Container) (string, error) {
	for _, container := range containers {
		if container.Name != "reloader" {
			continue
		}
		args := append(append([]string{}, container.Command...), container.Args...)
		version := ""
		for i := 0; i < len(args); i++ {
			if args[i] == "--" {
				break
			}
			if strings.HasPrefix(args[i], "--sub-store-path=") {
				version = strings.TrimPrefix(args[i], "--sub-store-path=")
			} else if args[i] == "--sub-store-path" {
				i++
				if i == len(args) || strings.HasPrefix(args[i], "--") {
					return "", fmt.Errorf("invalid --sub-store-path")
				}
				version = args[i]
			}
		}
		if version != "" {
			return version, nil
		}
		return "", fmt.Errorf("cannot determine reloader --sub-store-path")
	}
	return "", fmt.Errorf("reloader container not found")
}
