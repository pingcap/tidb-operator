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

package calculate

import (
	"fmt"
	"strconv"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	promClient "github.com/prometheus/client_golang/api"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func CalculateWhetherStoragePressure(tac *v1alpha1.TidbClusterAutoScaler, capacitySq, availableSq *SingleQuery,
	client promClient.Client, metric v1alpha1.CustomMetric) (bool, error) {
	if metric.Resource == nil ||
		metric.Resource.Name != corev1.ResourceStorage ||
		metric.LeastRemainAvailableStoragePercent == nil {
		return false, fmt.Errorf("tac[%s/%s] didn't set storage metric correctly", tac.Namespace, tac.Name)
	}

	// query total available storage size
	resp := &Response{}
	err := queryMetricsFromPrometheus(tac, client, availableSq, resp)
	if err != nil {
		return false, err
	}
	var availableSize uint64
	for _, r := range resp.Data.Result {
		if r.Metric.Cluster == tac.Spec.Cluster.Name {
			availableSize, err = strconv.ParseUint(r.Value[1].(string), 10, 64)
			if err != nil {
				return false, err
			}
		}
	}

	// query total capacity storage size
	resp = &Response{}
	err = queryMetricsFromPrometheus(tac, client, capacitySq, resp)
	if err != nil {
		return false, err
	}
	var capacitySize uint64
	for _, r := range resp.Data.Result {
		if r.Metric.Cluster == tac.Spec.Cluster.Name {
			capacitySize, err = strconv.ParseUint(r.Value[1].(string), 10, 64)
			if err != nil {
				return false, err
			}
		}
	}
	v := *metric.LeastRemainAvailableStoragePercent
	baselineAvailableSize := (capacitySize / 100) * uint64(v)
	storagePressure := false
	if availableSize < baselineAvailableSize {
		storagePressure = true
	}

	var newStatus, oldStatus *v1alpha1.MetricsStatus
	for _, m := range tac.Status.TiKV.MetricsStatusList {
		if m.Name == string(corev1.ResourceStorage) {
			oldStatus = &m
			break
		}
	}
	storageMetrics := v1alpha1.StorageMetricsStatus{
		StoragePressure:          pointer.BoolPtr(storagePressure),
		AvailableStorage:         pointer.StringPtr(byteCountDecimal(availableSize)),
		CapacityStorage:          pointer.StringPtr(byteCountDecimal(capacitySize)),
		BaselineAvailableStorage: pointer.StringPtr(byteCountDecimal(baselineAvailableSize)),
	}
	if oldStatus != nil {
		oldStatus.StoragePressure = storageMetrics.StoragePressure
		oldStatus.AvailableStorage = storageMetrics.AvailableStorage
		oldStatus.CapacityStorage = storageMetrics.CapacityStorage
		oldStatus.BaselineAvailableStorage = storageMetrics.BaselineAvailableStorage
		newStatus = oldStatus
	} else {
		newStatus = &v1alpha1.MetricsStatus{
			Name:                 string(corev1.ResourceStorage),
			StorageMetricsStatus: storageMetrics,
		}
	}

	if storagePressure {
		if !isStoragePressureStartTimeRecordAlready(tac.Status) {
			newStatus.StorageMetricsStatus.StoragePressureStartTime = &metav1.Time{Time: time.Now()}
		}
	} else {
		newStatus.StoragePressureStartTime = nil
	}
	addMetricsStatusIntoMetricsStatusList(*newStatus, &tac.Status.TiKV.BasicAutoScalerStatus)
	return storagePressure, nil
}

// TODO: add unit test
func isStoragePressureStartTimeRecordAlready(tacStatus v1alpha1.TidbClusterAutoSclaerStatus) bool {
	if tacStatus.TiKV == nil {
		return false
	}
	if len(tacStatus.TiKV.MetricsStatusList) < 1 {
		return false
	}
	for _, metricsStatus := range tacStatus.TiKV.MetricsStatusList {
		if metricsStatus.Name == "storage" {
			if metricsStatus.StoragePressureStartTime != nil {
				return true
			}
		}
	}
	return false
}

func byteCountDecimal(b uint64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "kMGTPE"[exp])
}
