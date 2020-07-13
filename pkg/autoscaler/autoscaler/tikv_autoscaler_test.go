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
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	autoscaling "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func TestCheckWhetherAbleToScaleDueToStorage(t *testing.T) {
	g := NewGomegaWithT(t)
	now := time.Now()
	tac1 := newStorageMetricTac()
	tac1.Status = v1alpha1.TidbClusterAutoSclaerStatus{
		TiKV: &v1alpha1.TikvAutoScalerStatus{
			BasicAutoScalerStatus: v1alpha1.BasicAutoScalerStatus{
				MetricsStatusList: []v1alpha1.MetricsStatus{
					{
						Name: string(corev1.ResourceStorage),
						StorageMetricsStatus: v1alpha1.StorageMetricsStatus{
							StoragePressure: pointer.BoolPtr(true),
							StoragePressureStartTime: &metav1.Time{
								Time: now.Add(-2 * time.Minute),
							},
						},
					},
				},
				LastAutoScalingTimestamp: &metav1.Time{
					Time: now.Add(-10 * time.Second),
				},
			},
		},
	}
	tac2 := newStorageMetricTac()
	tac2.Status = v1alpha1.TidbClusterAutoSclaerStatus{
		TiKV: &v1alpha1.TikvAutoScalerStatus{
			BasicAutoScalerStatus: v1alpha1.BasicAutoScalerStatus{
				MetricsStatusList: []v1alpha1.MetricsStatus{
					{
						Name: string(corev1.ResourceStorage),
						StorageMetricsStatus: v1alpha1.StorageMetricsStatus{
							StoragePressure: pointer.BoolPtr(true),
							StoragePressureStartTime: &metav1.Time{
								Time: now.Add(-20 * time.Second),
							},
						},
					},
				},
				LastAutoScalingTimestamp: &metav1.Time{
					Time: now.Add(-10 * time.Second),
				},
			},
		},
	}
	tac3 := newStorageMetricTac()
	tac3.Status = v1alpha1.TidbClusterAutoSclaerStatus{
		TiKV: &v1alpha1.TikvAutoScalerStatus{
			BasicAutoScalerStatus: v1alpha1.BasicAutoScalerStatus{
				MetricsStatusList: []v1alpha1.MetricsStatus{
					{
						Name: string(corev1.ResourceStorage),
						StorageMetricsStatus: v1alpha1.StorageMetricsStatus{
							StoragePressure: pointer.BoolPtr(true),
							StoragePressureStartTime: &metav1.Time{
								Time: now.Add(-2 * time.Minute),
							},
						},
					},
				},
				LastAutoScalingTimestamp: &metav1.Time{
					Time: now.Add(-10 * time.Minute),
				},
			},
		},
	}
	testcases := []struct {
		name           string
		tac            *v1alpha1.TidbClusterAutoScaler
		metric         v1alpha1.CustomMetric
		expectedResult bool
	}{
		{
			name: "experienced disk pressure for enough time",
			tac:  tac1,
			metric: v1alpha1.CustomMetric{
				LeastStoragePressurePeriodSeconds: pointer.Int64Ptr(60),
			},
			expectedResult: true,
		},
		{
			name: "haven't experienced disk pressure for enough time",
			tac:  tac2,
			metric: v1alpha1.CustomMetric{
				LeastStoragePressurePeriodSeconds: pointer.Int64Ptr(60),
			},
			expectedResult: false,
		},
		{
			name: "last syncing time is stale",
			tac:  tac3,
			metric: v1alpha1.CustomMetric{
				LeastStoragePressurePeriodSeconds: pointer.Int64Ptr(60),
			},
			expectedResult: false,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			result, err := checkWhetherAbleToScaleDueToStorage(testcase.tac, testcase.metric, now, 30*time.Second)
			g.Expect(err).Should(BeNil())
			g.Expect(result).Should(Equal(testcase.expectedResult))
		})
	}
}

func newStorageMetricTac() *v1alpha1.TidbClusterAutoScaler {
	tac := newTidbClusterAutoScaler()
	tac.Spec.TiKV.Metrics = []v1alpha1.CustomMetric{
		{
			LeastStoragePressurePeriodSeconds: pointer.Int64Ptr(60),
			MetricSpec: autoscaling.MetricSpec{
				Type: autoscaling.ResourceMetricSourceType,
				Resource: &autoscaling.ResourceMetricSource{
					Name: corev1.ResourceStorage,
				},
			},
		},
	}
	return tac
}
