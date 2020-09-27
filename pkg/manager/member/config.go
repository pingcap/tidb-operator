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

package member

import (
	perrors "github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/util/toml"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog"
)

func updateConfigMap(old, new *corev1.ConfigMap) error {
	tomlField := []string{"config-file" /*pd,tikv,tidb */, "pump-config", "config_templ.toml" /*tiflash*/, "proxy_templ.toml" /*tiflash*/}

	for _, k := range tomlField {
		oldData, oldOK := old.Data[k]
		newData, newOK := new.Data[k]

		if !oldOK || !newOK {
			continue
		}

		equal, err := toml.Equal([]byte(oldData), []byte(newData))
		if err != nil {
			return perrors.Annotatef(err, "compare %s and %s failed", oldData, newData)
		}

		klog.V(3).Infof("compare %s and %s %v", oldData, newData, equal)

		if equal {
			new.Data[k] = oldData
		}
	}

	return nil
}

// updateConfigMap set the toml field as the old one if they are logically equal.
func updateConfigMapIfNeed(
	cmLister corelisters.ConfigMapLister,
	configUpdateStrategy v1alpha1.ConfigUpdateStrategy,
	inUseName string,
	desired *corev1.ConfigMap,
) error {
	if configUpdateStrategy == v1alpha1.ConfigUpdateStrategyInPlace {
		if inUseName != "" {
			desired.Name = inUseName
		}
		return nil
	}

	if configUpdateStrategy == v1alpha1.ConfigUpdateStrategyRollingUpdate {
		existing, err := cmLister.ConfigMaps(desired.Namespace).Get(inUseName)
		if err != nil {
			if errors.IsNotFound(err) {
				AddConfigMapDigestSuffix(desired)
				return nil
			}

			return perrors.AddStack(err)
		}

		klog.V(3).Infof("get in use configmap: %v", *existing)

		err = updateConfigMap(existing, desired)
		if err != nil {
			return err
		}

		AddConfigMapDigestSuffix(desired)

		klog.V(3).Infof("old: %+v, new: %+v", existing, desired)
		return nil
	}

	return perrors.Errorf("unknown config update strategy: %v", configUpdateStrategy)
}
