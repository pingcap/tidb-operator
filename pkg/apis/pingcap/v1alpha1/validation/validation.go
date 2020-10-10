// Copyright 2019 PingCAP, Inc.
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

package validation

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	corev1 "k8s.io/api/core/v1"
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	utilnet "k8s.io/utils/net"
)

// ValidateTidbCluster validates a TidbCluster, it performs basic validation for all TidbClusters despite it is legacy
// or not
func ValidateTidbCluster(tc *v1alpha1.TidbCluster) field.ErrorList {
	allErrs := field.ErrorList{}
	// validate metadata
	fldPath := field.NewPath("metadata")
	// validate metadata/annotations
	allErrs = append(allErrs, validateAnnotations(tc.ObjectMeta.Annotations, fldPath.Child("annotations"))...)
	// validate spec
	allErrs = append(allErrs, validateTiDBClusterSpec(&tc.Spec, field.NewPath("spec"))...)
	return allErrs
}

// ValidateDMCluster validates a DMCluster, it performs basic validation for all DMClusters despite it is legacy
// or not
func ValidateDMCluster(dc *v1alpha1.DMCluster) field.ErrorList {
	allErrs := field.ErrorList{}
	// validate metadata
	fldPath := field.NewPath("metadata")
	// validate metadata/annotations
	allErrs = append(allErrs, validateDMAnnotations(dc.ObjectMeta.Annotations, fldPath.Child("annotations"))...)
	// validate spec
	allErrs = append(allErrs, validateDMClusterSpec(&dc.Spec, field.NewPath("spec"))...)
	return allErrs
}

func ValidateTidbMonitor(monitor *v1alpha1.TidbMonitor) field.ErrorList {
	allErrs := field.ErrorList{}
	// validate monitor service
	if monitor.Spec.Grafana != nil {
		allErrs = append(allErrs, validateService(&monitor.Spec.Grafana.Service, field.NewPath("spec"))...)
	}
	allErrs = append(allErrs, validateService(&monitor.Spec.Prometheus.Service, field.NewPath("spec"))...)
	allErrs = append(allErrs, validateService(&monitor.Spec.Reloader.Service, field.NewPath("spec"))...)
	return allErrs
}

func validateAnnotations(anns map[string]string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, apivalidation.ValidateAnnotations(anns, fldPath)...)
	for _, key := range []string{label.AnnPDDeleteSlots, label.AnnTiDBDeleteSlots, label.AnnTiKVDeleteSlots, label.AnnTiFlashDeleteSlots} {
		allErrs = append(allErrs, validateDeleteSlots(anns, key, fldPath.Child(key))...)
	}
	return allErrs
}

func validateDMAnnotations(anns map[string]string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, apivalidation.ValidateAnnotations(anns, fldPath)...)
	for _, key := range []string{label.AnnDMMasterDeleteSlots, label.AnnDMWorkerDeleteSlots} {
		allErrs = append(allErrs, validateDeleteSlots(anns, key, fldPath.Child(key))...)
	}
	return allErrs
}

func validateTiDBClusterSpec(spec *v1alpha1.TidbClusterSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if spec.PD != nil {
		allErrs = append(allErrs, validatePDSpec(spec.PD, fldPath.Child("pd"))...)
	}
	if spec.TiKV != nil {
		allErrs = append(allErrs, validateTiKVSpec(spec.TiKV, fldPath.Child("tikv"))...)
	}
	if spec.TiDB != nil {
		allErrs = append(allErrs, validateTiDBSpec(spec.TiDB, fldPath.Child("tidb"))...)
	}
	if spec.Pump != nil {
		allErrs = append(allErrs, validatePumpSpec(spec.Pump, fldPath.Child("pump"))...)
	}
	if spec.TiFlash != nil {
		allErrs = append(allErrs, validateTiFlashSpec(spec.TiFlash, fldPath.Child("tiflash"))...)
	}
	if spec.TiCDC != nil {
		allErrs = append(allErrs, validateTiCDCSpec(spec.TiCDC, fldPath.Child("ticdc"))...)
	}
	return allErrs
}

func validatePDSpec(spec *v1alpha1.PDSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validateComponentSpec(&spec.ComponentSpec, fldPath)...)
	allErrs = append(allErrs, validateRequestsStorage(spec.ResourceRequirements.Requests, fldPath)...)
	return allErrs
}

func validateTiKVSpec(spec *v1alpha1.TiKVSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validateComponentSpec(&spec.ComponentSpec, fldPath)...)
	allErrs = append(allErrs, validateRequestsStorage(spec.ResourceRequirements.Requests, fldPath)...)
	if len(spec.DataSubDir) > 0 {
		allErrs = append(allErrs, validateLocalDescendingPath(spec.DataSubDir, fldPath.Child("dataSubDir"))...)
	}
	allErrs = append(allErrs, validateTimeDurationStr(spec.EvictLeaderTimeout, fldPath.Child("evictLeaderTimeout"))...)
	return allErrs
}

func validateTiFlashSpec(spec *v1alpha1.TiFlashSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validateComponentSpec(&spec.ComponentSpec, fldPath)...)
	allErrs = append(allErrs, validateTiFlashConfig(spec.Config, fldPath)...)
	if len(spec.StorageClaims) < 1 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("spec.StorageClaims"),
			spec.StorageClaims, "storageClaims should be configured at least one item."))
	}
	return allErrs
}

func validateTiCDCSpec(spec *v1alpha1.TiCDCSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validateComponentSpec(&spec.ComponentSpec, fldPath)...)
	return allErrs
}

func validateTiFlashConfig(config *v1alpha1.TiFlashConfig, path *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if config == nil {
		return allErrs
	}

	if config.CommonConfig != nil {
		if config.CommonConfig.Flash != nil {
			if config.CommonConfig.Flash.OverlapThreshold != nil {
				if *config.CommonConfig.Flash.OverlapThreshold < 0 || *config.CommonConfig.Flash.OverlapThreshold > 1 {
					allErrs = append(allErrs, field.Invalid(path.Child("config.config.flash.overlap_threshold"),
						config.CommonConfig.Flash.OverlapThreshold,
						"overlap_threshold must be in the range of [0,1]."))
				}
			}
			if config.CommonConfig.Flash.FlashCluster != nil {
				if config.CommonConfig.Flash.FlashCluster.ClusterLog != nil {
					splitPath := strings.Split(*config.CommonConfig.Flash.FlashCluster.ClusterLog, string(os.PathSeparator))
					// The log path should be at least /dir/base.log
					if len(splitPath) < 3 {
						allErrs = append(allErrs, field.Invalid(path.Child("config.config.flash.flash_cluster.log"),
							config.CommonConfig.Flash.FlashCluster.ClusterLog,
							"log path should include at least one level dir."))
					}
				}
			}
			if config.CommonConfig.Flash.FlashProxy != nil {
				if config.CommonConfig.Flash.FlashProxy.LogFile != nil {
					splitPath := strings.Split(*config.CommonConfig.Flash.FlashProxy.LogFile, string(os.PathSeparator))
					// The log path should be at least /dir/base.log
					if len(splitPath) < 3 {
						allErrs = append(allErrs, field.Invalid(path.Child("config.config.flash.flash_proxy.log-file"),
							config.CommonConfig.Flash.FlashProxy.LogFile,
							"log path should include at least one level dir."))
					}
				}
			}
		}
		if config.CommonConfig.FlashLogger != nil {
			if config.CommonConfig.FlashLogger.ServerLog != nil {
				splitPath := strings.Split(*config.CommonConfig.FlashLogger.ServerLog, string(os.PathSeparator))
				// The log path should be at least /dir/base.log
				if len(splitPath) < 3 {
					allErrs = append(allErrs, field.Invalid(path.Child("config.config.logger.log"),
						config.CommonConfig.FlashLogger.ServerLog,
						"log path should include at least one level dir."))
				}
			}
			if config.CommonConfig.FlashLogger.ErrorLog != nil {
				splitPath := strings.Split(*config.CommonConfig.FlashLogger.ErrorLog, string(os.PathSeparator))
				// The log path should be at least /dir/base.log
				if len(splitPath) < 3 {
					allErrs = append(allErrs, field.Invalid(path.Child("config.config.logger.errorlog"),
						config.CommonConfig.FlashLogger.ErrorLog,
						"log path should include at least one level dir."))
				}
			}
		}
	}
	return allErrs
}

func validateTiDBSpec(spec *v1alpha1.TiDBSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validateComponentSpec(&spec.ComponentSpec, fldPath)...)
	if spec.Service != nil {
		allErrs = append(allErrs, validateService(&spec.Service.ServiceSpec, fldPath)...)
	}
	return allErrs
}

func validatePumpSpec(spec *v1alpha1.PumpSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validateComponentSpec(&spec.ComponentSpec, fldPath)...)
	return allErrs
}

func validateDMClusterSpec(spec *v1alpha1.DMClusterSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if spec.Version != "" {
		clusterVersionLT2, _ := clusterVersionLessThan2(spec.Version)
		if clusterVersionLT2 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("version"), spec.Version, "dm cluster version can't set to v1.x.y"))
		}
	}
	if spec.Discovery.Address == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("discovery.address"), "discovery.address must not be empty"))
	}
	allErrs = append(allErrs, validateMasterSpec(&spec.Master, fldPath.Child("master"))...)
	if spec.Worker != nil {
		allErrs = append(allErrs, validateWorkerSpec(spec.Worker, fldPath.Child("worker"))...)
	}
	return allErrs
}

func validateMasterSpec(spec *v1alpha1.MasterSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validateComponentSpec(&spec.ComponentSpec, fldPath)...)
	// make sure that storageSize for dm-master is assigned
	if spec.Replicas > 0 && spec.StorageSize == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("storageSize"), "storageSize must not be empty"))
	}
	return allErrs
}

func validateWorkerSpec(spec *v1alpha1.WorkerSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validateComponentSpec(&spec.ComponentSpec, fldPath)...)
	return allErrs
}

func validateComponentSpec(spec *v1alpha1.ComponentSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	// TODO validate other fields
	allErrs = append(allErrs, validateEnv(spec.Env, fldPath.Child("env"))...)
	return allErrs
}

//validateRequestsStorage validates resources requests storage
func validateRequestsStorage(requests corev1.ResourceList, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if _, ok := requests[corev1.ResourceStorage]; !ok {
		allErrs = append(allErrs, field.Required(fldPath.Child("requests.storage").Key((string(corev1.ResourceStorage))), "storage request must not be empty"))
	}
	return allErrs
}

// validateEnv validates env vars
func validateEnv(vars []corev1.EnvVar, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	for i, ev := range vars {
		idxPath := fldPath.Index(i)
		if len(ev.Name) == 0 {
			allErrs = append(allErrs, field.Required(idxPath.Child("name"), ""))
		} else {
			for _, msg := range validation.IsEnvVarName(ev.Name) {
				allErrs = append(allErrs, field.Invalid(idxPath.Child("name"), ev.Name, msg))
			}
		}
		allErrs = append(allErrs, validateEnvVarValueFrom(ev, idxPath.Child("valueFrom"))...)
	}
	return allErrs
}

func validateEnvVarValueFrom(ev corev1.EnvVar, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if ev.ValueFrom == nil {
		return allErrs
	}

	numSources := 0

	if ev.ValueFrom.FieldRef != nil {
		numSources++
		allErrs = append(allErrs, field.Invalid(fldPath.Child("fieldRef"), "", "fieldRef is not supported"))
	}
	if ev.ValueFrom.ResourceFieldRef != nil {
		numSources++
		allErrs = append(allErrs, field.Invalid(fldPath.Child("resourceFieldRef"), "", "resourceFieldRef is not supported"))
	}
	if ev.ValueFrom.ConfigMapKeyRef != nil {
		numSources++
		allErrs = append(allErrs, validateConfigMapKeySelector(ev.ValueFrom.ConfigMapKeyRef, fldPath.Child("configMapKeyRef"))...)
	}
	if ev.ValueFrom.SecretKeyRef != nil {
		numSources++
		allErrs = append(allErrs, validateSecretKeySelector(ev.ValueFrom.SecretKeyRef, fldPath.Child("secretKeyRef"))...)
	}

	if numSources == 0 {
		allErrs = append(allErrs, field.Invalid(fldPath, "", "must specify one of: `configMapKeyRef` or `secretKeyRef`"))
	} else if len(ev.Value) != 0 {
		if numSources != 0 {
			allErrs = append(allErrs, field.Invalid(fldPath, "", "may not be specified when `value` is not empty"))
		}
	} else if numSources > 1 {
		allErrs = append(allErrs, field.Invalid(fldPath, "", "may not have more than one field specified at a time"))
	}

	return allErrs
}

func validateConfigMapKeySelector(s *corev1.ConfigMapKeySelector, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	for _, msg := range apivalidation.NameIsDNSSubdomain(s.Name, false) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("name"), s.Name, msg))
	}
	if len(s.Key) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("key"), ""))
	} else {
		for _, msg := range validation.IsConfigMapKey(s.Key) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("key"), s.Key, msg))
		}
	}

	return allErrs
}

func validateSecretKeySelector(s *corev1.SecretKeySelector, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	for _, msg := range apivalidation.NameIsDNSSubdomain(s.Name, false) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("name"), s.Name, msg))
	}
	if len(s.Key) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("key"), ""))
	} else {
		for _, msg := range validation.IsConfigMapKey(s.Key) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("key"), s.Key, msg))
		}
	}

	return allErrs
}

// ValidateCreateTidbCLuster validates a newly created TidbCluster
func ValidateCreateTidbCluster(tc *v1alpha1.TidbCluster) field.ErrorList {
	allErrs := field.ErrorList{}
	// basic validation
	allErrs = append(allErrs, ValidateTidbCluster(tc)...)
	allErrs = append(allErrs, validateNewTidbClusterSpec(&tc.Spec, field.NewPath("spec"))...)
	return allErrs
}

// ValidateUpdateTidbCluster validates a new TidbCluster against an existing TidbCluster to be updated
func ValidateUpdateTidbCluster(old, tc *v1alpha1.TidbCluster) field.ErrorList {

	allErrs := field.ErrorList{}
	// basic validation
	allErrs = append(allErrs, ValidateTidbCluster(tc)...)
	if old.GetInstanceName() != tc.GetInstanceName() {
		allErrs = append(allErrs, field.Invalid(field.NewPath("labels"), tc.Labels,
			"The instance must not be mutate or set value other than the cluster name"))
	}
	allErrs = append(allErrs, validateUpdatePDConfig(old.Spec.PD.Config, tc.Spec.PD.Config, field.NewPath("spec.pd.config"))...)
	allErrs = append(allErrs, disallowUsingLegacyAPIInNewCluster(old, tc)...)

	return allErrs
}

// For now we limit some validations only in Create phase to keep backward compatibility
// TODO(aylei): call this in ValidateTidbCluster after we deprecated the old versions of helm chart officially
func validateNewTidbClusterSpec(spec *v1alpha1.TidbClusterSpec, path *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if spec.Version == "" {
		allErrs = append(allErrs, field.Invalid(path.Child("version"), spec.Version, "version must not be empty"))
	}
	if spec.TiDB.BaseImage == "" {
		allErrs = append(allErrs, field.Invalid(path.Child("tidb.baseImage"), spec.TiDB.BaseImage, "baseImage of TiDB must not be empty"))
	}
	if spec.PD.BaseImage == "" {
		allErrs = append(allErrs, field.Invalid(path.Child("pd.baseImage"), spec.PD.BaseImage, "baseImage of PD must not be empty"))
	}
	if spec.TiKV.BaseImage == "" {
		allErrs = append(allErrs, field.Invalid(path.Child("tikv.baseImage"), spec.TiKV.BaseImage, "baseImage of TiKV must not be empty"))
	}
	if spec.TiDB.Image != "" {
		allErrs = append(allErrs, field.Invalid(path.Child("tidb.image"), spec.TiDB.Image, "image has been deprecated, use baseImage instead"))
	}
	if spec.TiKV.Image != "" {
		allErrs = append(allErrs, field.Invalid(path.Child("tikv.image"), spec.TiKV.Image, "image has been deprecated, use baseImage instead"))
	}
	if spec.PD.Image != "" {
		allErrs = append(allErrs, field.Invalid(path.Child("pd.image"), spec.PD.Image, "image has been deprecated, use baseImage instead"))
	}
	return allErrs
}

// disallowUsingLegacyAPIInNewCluster checks if user use the legacy API in newly create cluster during update
// TODO(aylei): this could be removed after we enable validateTidbCluster() in update, which is more strict
func disallowUsingLegacyAPIInNewCluster(old, tc *v1alpha1.TidbCluster) field.ErrorList {
	allErrs := field.ErrorList{}
	path := field.NewPath("spec")
	if old.Spec.Version != "" && tc.Spec.Version == "" {
		allErrs = append(allErrs, field.Invalid(path.Child("version"), tc.Spec.Version, "version must not be empty"))
	}
	if old.Spec.TiDB.BaseImage != "" && tc.Spec.TiDB.BaseImage == "" {
		allErrs = append(allErrs, field.Invalid(path.Child("tidb.baseImage"), tc.Spec.TiDB.BaseImage, "baseImage of TiDB must not be empty"))
	}
	if old.Spec.PD.BaseImage != "" && tc.Spec.PD.BaseImage == "" {
		allErrs = append(allErrs, field.Invalid(path.Child("pd.baseImage"), tc.Spec.PD.BaseImage, "baseImage of PD must not be empty"))
	}
	if old.Spec.TiKV.BaseImage != "" && tc.Spec.TiKV.BaseImage == "" {
		allErrs = append(allErrs, field.Invalid(path.Child("tikv.baseImage"), tc.Spec.TiKV.BaseImage, "baseImage of TiKV must not be empty"))
	}
	if old.Spec.TiDB.Config != nil && tc.Spec.TiDB.Config == nil {
		allErrs = append(allErrs, field.Invalid(path.Child("tidb.config"), tc.Spec.TiDB.Config, "tidb.config must not be nil"))
	}
	if old.Spec.TiKV.Config != nil && tc.Spec.TiKV.Config == nil {
		allErrs = append(allErrs, field.Invalid(path.Child("tikv.config"), tc.Spec.TiKV.Config, "TiKV.config must not be nil"))
	}
	if old.Spec.PD.Config != nil && tc.Spec.PD.Config == nil {
		allErrs = append(allErrs, field.Invalid(path.Child("pd.config"), tc.Spec.PD.Config, "PD.config must not be nil"))
	}
	return allErrs
}

func validateUpdatePDConfig(old, conf *v1alpha1.PDConfig, path *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	// for newly created cluster, both old and new are non-nil, guaranteed by validation
	if old == nil || conf == nil {
		return allErrs
	}

	if conf.Security != nil && len(conf.Security.CertAllowedCN) > 1 {
		allErrs = append(allErrs, field.Invalid(path.Child("security.cert-allowed-cn"), conf.Security.CertAllowedCN,
			"Only one CN is currently supported"))
	}

	if !reflect.DeepEqual(old.Schedule, conf.Schedule) {
		allErrs = append(allErrs, field.Invalid(path.Child("schedule"), conf.Schedule,
			"PD Schedule Config is immutable through CRD, please modify with pd-ctl instead."))
	}
	if !reflect.DeepEqual(old.Replication, conf.Replication) {
		allErrs = append(allErrs, field.Invalid(path.Child("replication"), conf.Replication,
			"PD Replication Config is immutable through CRD, please modify with pd-ctl instead."))
	}
	return allErrs
}

func validateDeleteSlots(annotations map[string]string, key string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if annotations != nil {
		if value, ok := annotations[key]; ok {
			var slice []int32
			err := json.Unmarshal([]byte(value), &slice)
			if err != nil {
				msg := fmt.Sprintf("value of %q annotation must be a JSON list of int32", key)
				allErrs = append(allErrs, field.Invalid(fldPath, value, msg))
			}
		}
	}
	return allErrs
}

func validateService(spec *v1alpha1.ServiceSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	//validate LoadBalancerSourceRanges field from service
	if len(spec.LoadBalancerSourceRanges) > 0 {
		ip := spec.LoadBalancerSourceRanges
		_, err := utilnet.ParseIPNets(ip...)
		if err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("spec.LoadBalancerSourceRanges"), spec.LoadBalancerSourceRanges, "service.Spec.LoadBalancerSourceRanges is not valid. Expecting a list of IP ranges. For example, 10.0.0.0/24."))
		}
	}
	return allErrs
}

// This validate will make sure targetPath:
// 1. is not abs path
// 2. does not have any element which is ".."
func validateLocalDescendingPath(targetPath string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if path.IsAbs(targetPath) {
		allErrs = append(allErrs, field.Invalid(fldPath, targetPath, "must be a relative path"))
	}

	allErrs = append(allErrs, validatePathNoBacksteps(targetPath, fldPath)...)

	return allErrs
}

// validatePathNoBacksteps makes sure the targetPath does not have any `..` path elements when split
//
// This assumes the OS of the apiserver and the nodes are the same. The same check should be done
// on the node to ensure there are no backsteps.
func validatePathNoBacksteps(targetPath string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	parts := strings.Split(filepath.ToSlash(targetPath), "/")
	for _, item := range parts {
		if item == ".." {
			allErrs = append(allErrs, field.Invalid(fldPath, targetPath, "must not contain '..'"))
			break // even for `../../..`, one error is sufficient to make the point
		}
	}
	return allErrs
}

func validateTimeDurationStr(timeStr *string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if timeStr != nil {
		d, err := time.ParseDuration(*timeStr)
		if err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath, timeStr, "mush be a valid Go time duration string, e.g. 3m"))
		} else if d <= 0 {
			allErrs = append(allErrs, field.Invalid(fldPath, timeStr, "must be a positive Go time duration"))
		}
	}
	return allErrs
}

// clusterVersionLessThan2 makes sure that deployed dm cluster version not to be v1.0.x
func clusterVersionLessThan2(version string) (bool, error) {
	v, err := semver.NewVersion(version)
	if err != nil {
		return false, err
	}

	return v.Major() < 2, nil
}
