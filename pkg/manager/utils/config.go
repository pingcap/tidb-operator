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

package utils

import (
	"fmt"

	perrors "github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apis/util/toml"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	corelisters "k8s.io/client-go/listers/core/v1"
)

func updateConfigMap(old, new *corev1.ConfigMap) (bool, error) {
	dataEqual := true

	// check config
	tomlField := []string{
		"config-file",          // pd,tikv,tidb
		"pump-config",          // pump
		"config_templ.toml",    // tiflash
		"proxy_templ.toml",     // tiflash
		"ng-monitoring-config", // ng-monitoring
	}
	for _, k := range tomlField {
		oldData, oldOK := old.Data[k]
		newData, newOK := new.Data[k]

		if oldOK != newOK {
			dataEqual = false
		}

		if !oldOK || !newOK {
			continue
		}

		equal, err := toml.Equal([]byte(oldData), []byte(newData))
		if err != nil {
			return false, perrors.Annotatef(err, "compare %s/%s %s and %s failed", old.Namespace, old.Name, oldData, newData)
		}

		if equal {
			new.Data[k] = oldData
		} else {
			dataEqual = false
		}
	}

	// check startup script
	field := "startup-script"
	oldScript, oldExist := old.Data[field]
	newScript, newExist := new.Data[field]
	if oldExist != newExist {
		dataEqual = false
	} else if oldExist && newExist {
		if oldScript != newScript {
			dataEqual = false
		}
	}

	return dataEqual, nil
}

// UpdateConfigMapIfNeed set the toml field as the old one if they are logically equal.
func UpdateConfigMapIfNeed(
	cmLister corelisters.ConfigMapLister,
	configUpdateStrategy v1alpha1.ConfigUpdateStrategy,
	inUseName string,
	desired *corev1.ConfigMap,
) error {

	switch configUpdateStrategy {
	case v1alpha1.ConfigUpdateStrategyInPlace:
		if inUseName != "" {
			desired.Name = inUseName
		}
		return nil
	case v1alpha1.ConfigUpdateStrategyRollingUpdate:
		existing, err := cmLister.ConfigMaps(desired.Namespace).Get(inUseName)
		if err != nil {
			if errors.IsNotFound(err) {
				AddConfigMapDigestSuffix(desired)
				return nil
			}

			return perrors.AddStack(err)
		}

		dataEqual, err := updateConfigMap(existing, desired)
		if err != nil {
			return err
		}

		AddConfigMapDigestSuffix(desired)

		confirmNameByData(existing, desired, dataEqual)

		return nil
	default:
		return perrors.Errorf("unknown config update strategy: %v", configUpdateStrategy)

	}
}

// confirmNameByData is used to fix the problem that
// when configUpdateStrategy is changed from InPlace to RollingUpdate for the first time,
// the name of desired configmap maybe different from the existing one while
// the data of them are the same, which will cause rolling update.
func confirmNameByData(existing, desired *corev1.ConfigMap, dataEqual bool) {
	if dataEqual && existing.Name != desired.Name {
		desired.Name = existing.Name
	}
	if !dataEqual && existing.Name == desired.Name {
		desired.Name = fmt.Sprintf("%s-new", desired.Name)
	}
}
