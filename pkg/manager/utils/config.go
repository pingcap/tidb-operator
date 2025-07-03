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

	// When doing point-in-time-restoring or something, we will put an "overlay" over the tikv configuration.
	// Invariants:
	// `old` -> contains two fields "xxx" and "xxx-overlay", where:
	// "xxx"         => contains the config derived from spec UNIONs the overlay, this config will be used by the component.
	// "xxx-overlay" => contains the overlay, this should be applied to `new`.
	// '             *> this field will be added by other components after the compare finishes.
	// '             *> the "overlay" part won't join the compare.
	// `new` -> contains "xxx" only, which is derived from the spec.
	//
	// when comparing `old` and `new`, as the overlay configuration isn't expected to be compared,
	// it will be applied to `new` becore for comparing, so we have the equality:
	//
	// old == spec.old + overlay == spec.new + overlay, which sounds iff spec.old == spec.new
	//        |                |    |       \  |     +---------\
	//        +----old.data----+    +new.data+ +old.data-overlay+
	applyOverlay := func(old, new *corev1.ConfigMap, key string) ([]byte, bool, error) {
		if overlay, ok := old.Data[fmt.Sprintf("%s-overlay", key)]; ok {
			cfg := v1alpha1.NewTiKVConfig()
			if err := cfg.UnmarshalTOML([]byte(new.Data[key])); err != nil {
				return nil, false, err
			}

			// We should parse and merge.
			// directly call `cfg.UnmarshalTOML` will override the `gc` scope in the origin test.
			cfgOverlay := v1alpha1.NewTiKVConfig()
			if err := cfgOverlay.UnmarshalTOML([]byte(overlay)); err != nil {
				return nil, false, err
			}
			if err := cfg.Merge(cfgOverlay.GenericConfig); err != nil {
				return nil, false, err
			}
			applied, err := cfg.MarshalTOML()
			return applied, true, err
		}
		return []byte(new.Data[key]), false, nil
	}

	// check config
	tomlField := []string{
		"config-file",       // pd,dm,tikv,tidb,ng-monitoring
		"pump-config",       // pump
		"config_templ.toml", // tiflash
		"proxy_templ.toml",  // tiflash
	}
	for _, k := range tomlField {
		oldData, oldOK := old.Data[k]
		_, newOK := new.Data[k]

		if oldOK != newOK {
			dataEqual = false
		}

		if !oldOK || !newOK {
			continue
		}

		appliedNewData, overlayApplied, err := applyOverlay(old, new, k)
		if err != nil {
			return false, err
		}

		equal, err := toml.Equal(appliedNewData, []byte(oldData))
		if err != nil {
			return false, perrors.Annotatef(err, "compare %s/%s %s and %s failed", old.Namespace, old.Name, oldData, string(appliedNewData))
		}

		if equal {
			// If there is an overlay, don't put the applied part to the new data.
			// They should be added by the caller soon.
			if !overlayApplied {
				new.Data[k] = oldData
			}
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
