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
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"
)

func TestValidateAnnotations(t *testing.T) {
	successCases := []struct {
		name string
		tc   v1alpha1.TidbCluster
	}{
		{
			name: "all-fields-valid",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Annotations: map[string]string{
						label.AnnTiKVDeleteSlots:    "[1,2]",
						label.AnnTiFlashDeleteSlots: "[1]",
					},
				},
				Spec: v1alpha1.TidbClusterSpec{
					Version: "v3.0.8",
					PD: &v1alpha1.PDSpec{
						BaseImage: "pingcap/pd",
						Config:    v1alpha1.NewPDConfig(),
					},
					TiKV: &v1alpha1.TiKVSpec{
						BaseImage: "pingcap/tikv",
						Config:    v1alpha1.NewTiKVConfig(),
					},
					TiDB: &v1alpha1.TiDBSpec{
						BaseImage: "pingcap/tidb",
						Config:    v1alpha1.NewTiDBConfig(),
					},
				},
			},
		},
		{
			name: "no delete slots",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test",
					Annotations: map[string]string{},
				},
				Spec: v1alpha1.TidbClusterSpec{
					Version: "v3.0.8",
					PD: &v1alpha1.PDSpec{
						BaseImage: "pingcap/pd",
						Config:    v1alpha1.NewPDConfig(),
					},
					TiKV: &v1alpha1.TiKVSpec{
						BaseImage: "pingcap/tikv",
						Config:    v1alpha1.NewTiKVConfig(),
					},
					TiDB: &v1alpha1.TiDBSpec{
						BaseImage: "pingcap/tidb",
						Config:    v1alpha1.NewTiDBConfig(),
					},
				},
			},
		},
		// TODO: more cases
	}

	for _, v := range successCases {
		if errs := validateAnnotations(v.tc.ObjectMeta.Annotations, field.NewPath("metadata", "annotations")); len(errs) != 0 {
			t.Errorf("[%s]: unexpected error: %v", v.name, errs)
		}
	}

	errorCases := []struct {
		name string
		tc   v1alpha1.TidbCluster
		errs []field.Error
	}{
		{
			name: "delete slots empty string",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Annotations: map[string]string{
						label.AnnTiKVDeleteSlots:    "",
						label.AnnTiFlashDeleteSlots: "",
					},
				},
				Spec: v1alpha1.TidbClusterSpec{
					Version: "v3.0.8",
					PD: &v1alpha1.PDSpec{
						BaseImage: "pingcap/pd",
						Config:    v1alpha1.NewPDConfig(),
					},
					TiKV: &v1alpha1.TiKVSpec{
						BaseImage: "pingcap/tikv",
						Config:    v1alpha1.NewTiKVConfig(),
					},
					TiDB: &v1alpha1.TiDBSpec{
						BaseImage: "pingcap/tidb",
						Config:    v1alpha1.NewTiDBConfig(),
					},
				},
			},
			errs: []field.Error{
				{
					Type:   field.ErrorTypeInvalid,
					Detail: `value of "tikv.tidb.pingcap.com/delete-slots" annotation must be a JSON list of int32`,
				},
				{
					Type:   field.ErrorTypeInvalid,
					Detail: `value of "tiflash.tidb.pingcap.com/delete-slots" annotation must be a JSON list of int32`,
				},
			},
		},
		{
			name: "delete slots invalid format",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Annotations: map[string]string{
						label.AnnTiDBDeleteSlots: "1,2,3",
					},
				},
				Spec: v1alpha1.TidbClusterSpec{
					Version: "v3.0.8",
					PD: &v1alpha1.PDSpec{
						BaseImage: "pingcap/pd",
						Config:    v1alpha1.NewPDConfig(),
					},
					TiKV: &v1alpha1.TiKVSpec{
						BaseImage: "pingcap/tikv",
						Config:    v1alpha1.NewTiKVConfig(),
					},
					TiDB: &v1alpha1.TiDBSpec{
						BaseImage: "pingcap/tidb",
						Config:    v1alpha1.NewTiDBConfig(),
					},
				},
			},
			errs: []field.Error{
				{
					Type:   field.ErrorTypeInvalid,
					Detail: `value of "tidb.tidb.pingcap.com/delete-slots" annotation must be a JSON list of int32`,
				},
			},
		},
	}

	for _, v := range errorCases {
		errs := validateAnnotations(v.tc.ObjectMeta.Annotations, field.NewPath("metadata", "annotations"))
		if len(errs) != len(v.errs) {
			t.Errorf("[%s]: expected %d failures, got %d failures: %v", v.name, len(v.errs), len(errs), errs)
			continue
		}
		for i := range errs {
			if errs[i].Type != v.errs[i].Type {
				t.Errorf("[%s]: expected error type %q, got %q", v.name, v.errs[i].Type, errs[i].Type)
			}
			if !strings.Contains(errs[i].Detail, v.errs[i].Detail) {
				t.Errorf("[%s]: expected error errs[i].Detail %q, got %q", v.name, v.errs[i].Detail, errs[i].Detail)
			}
			if len(v.errs[i].Field) > 0 {
				if errs[i].Field != v.errs[i].Field {
					t.Errorf("[%s]: expected error field %q, got %q", v.name, v.errs[i].Field, errs[i].Field)
				}
			}
		}
	}
}

func TestValidateDMAnnotations(t *testing.T) {
	successCases := []struct {
		name string
		dc   v1alpha1.DMCluster
	}{
		{
			name: "all-fields-valid",
			dc: v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Annotations: map[string]string{
						label.AnnDMMasterDeleteSlots: "[1,2]",
						label.AnnDMWorkerDeleteSlots: "[1]",
					},
				},
				Spec: v1alpha1.DMClusterSpec{
					Version: "v2.0.0-rc.1",
					Master: v1alpha1.MasterSpec{
						BaseImage: "pingcap/dm",
						Config:    &v1alpha1.MasterConfig{},
					},
					Worker: &v1alpha1.WorkerSpec{
						BaseImage: "pingcap/dm",
						Config:    &v1alpha1.WorkerConfig{},
					},
				},
			},
		},
		{
			name: "no delete slots",
			dc: v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test",
					Annotations: map[string]string{},
				},
				Spec: v1alpha1.DMClusterSpec{
					Version: "v2.0.0-rc.1",
					Master: v1alpha1.MasterSpec{
						BaseImage: "pingcap/dm",
						Config:    &v1alpha1.MasterConfig{},
					},
					Worker: &v1alpha1.WorkerSpec{
						BaseImage: "pingcap/dm",
						Config:    &v1alpha1.WorkerConfig{},
					},
				},
			},
		},
		// TODO: more cases
	}

	for _, v := range successCases {
		if errs := validateAnnotations(v.dc.ObjectMeta.Annotations, field.NewPath("metadata", "annotations")); len(errs) != 0 {
			t.Errorf("[%s]: unexpected error: %v", v.name, errs)
		}
	}

	errorCases := []struct {
		name string
		dc   v1alpha1.DMCluster
		errs []field.Error
	}{
		{
			name: "delete slots empty string",
			dc: v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Annotations: map[string]string{
						label.AnnDMMasterDeleteSlots: "",
						label.AnnDMWorkerDeleteSlots: "",
					},
				},
				Spec: v1alpha1.DMClusterSpec{
					Version: "v2.0.0-rc.1",
					Master: v1alpha1.MasterSpec{
						BaseImage: "pingcap/dm",
						Config:    &v1alpha1.MasterConfig{},
					},
					Worker: &v1alpha1.WorkerSpec{
						BaseImage: "pingcap/dm",
						Config:    &v1alpha1.WorkerConfig{},
					},
				},
			},
			errs: []field.Error{
				{
					Type:   field.ErrorTypeInvalid,
					Detail: `value of "dm-master.tidb.pingcap.com/delete-slots" annotation must be a JSON list of int32`,
				},
				{
					Type:   field.ErrorTypeInvalid,
					Detail: `value of "dm-worker.tidb.pingcap.com/delete-slots" annotation must be a JSON list of int32`,
				},
			},
		},
		{
			name: "delete slots invalid format",
			dc: v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Annotations: map[string]string{
						label.AnnDMWorkerDeleteSlots: "1,2,3",
					},
				},
				Spec: v1alpha1.DMClusterSpec{
					Version: "v2.0.0-rc.1",
					Master: v1alpha1.MasterSpec{
						BaseImage: "pingcap/dm",
						Config:    &v1alpha1.MasterConfig{},
					},
					Worker: &v1alpha1.WorkerSpec{
						BaseImage: "pingcap/dm",
						Config:    &v1alpha1.WorkerConfig{},
					},
				},
			},
			errs: []field.Error{
				{
					Type:   field.ErrorTypeInvalid,
					Detail: `value of "dm-worker.tidb.pingcap.com/delete-slots" annotation must be a JSON list of int32`,
				},
			},
		},
	}

	for _, v := range errorCases {
		errs := validateDMAnnotations(v.dc.ObjectMeta.Annotations, field.NewPath("metadata", "annotations"))
		if len(errs) != len(v.errs) {
			t.Errorf("[%s]: expected %d failures, got %d failures: %v", v.name, len(v.errs), len(errs), errs)
			continue
		}
		for i := range errs {
			if errs[i].Type != v.errs[i].Type {
				t.Errorf("[%s]: expected error type %q, got %q", v.name, v.errs[i].Type, errs[i].Type)
			}
			if !strings.Contains(errs[i].Detail, v.errs[i].Detail) {
				t.Errorf("[%s]: expected error errs[i].Detail %q, got %q", v.name, v.errs[i].Detail, errs[i].Detail)
			}
			if len(v.errs[i].Field) > 0 {
				if errs[i].Field != v.errs[i].Field {
					t.Errorf("[%s]: expected error field %q, got %q", v.name, v.errs[i].Field, errs[i].Field)
				}
			}
		}
	}
}

func TestValidateRequestsStorage(t *testing.T) {
	g := NewGomegaWithT(t)
	tests := []struct {
		name                 string
		haveRequest          bool
		resourceRequirements corev1.ResourceRequirements
		expectedErrors       int
	}{
		{
			name:        "has request storage",
			haveRequest: true,
			resourceRequirements: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("10G"),
				},
			},
			expectedErrors: 0,
		},
		{
			name:        "Empty request storage",
			haveRequest: false,
			resourceRequirements: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{},
			},
			expectedErrors: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := newTidbCluster()
			if tt.haveRequest {
				tc.Spec.PD.ResourceRequirements = tt.resourceRequirements
				tc.Spec.TiKV.ResourceRequirements = tt.resourceRequirements
			}
			err := ValidateTidbCluster(tc)
			r := len(err)
			g.Expect(r).Should(Equal(tt.expectedErrors))
		})
	}
}

func TestValidateService(t *testing.T) {
	g := NewGomegaWithT(t)
	tests := []struct {
		name                     string
		loadBalancerSourceRanges []string
		expectedErrors           int
	}{
		{
			name:                     "correct LoadBalancerSourceRanges",
			loadBalancerSourceRanges: strings.Split("192.168.0.1/32", ","),
			expectedErrors:           0,
		},
		{
			name:                     "incorrect LoadBalancerSourceRanges",
			loadBalancerSourceRanges: strings.Split("192.168.0.1", ","),
			expectedErrors:           1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newService()
			svc.LoadBalancerSourceRanges = tt.loadBalancerSourceRanges
			err := validateService(svc, field.NewPath("spec"))
			r := len(err)
			g.Expect(r).Should(Equal(tt.expectedErrors))
			if r > 0 {
				for _, e := range err {
					g.Expect(e.Detail).To(ContainSubstring("service.Spec.LoadBalancerSourceRanges is not valid. Expecting a list of IP ranges. For example, 10.0.0.0/24."))
				}
			}
		})
	}
}

func TestValidateTidbMonitor(t *testing.T) {
	g := NewGomegaWithT(t)
	tests := []struct {
		name                     string
		loadBalancerSourceRanges []string
		expectedErrors           int
	}{
		{
			name:                     "correct LoadBalancerSourceRanges",
			loadBalancerSourceRanges: strings.Split("192.168.0.1/24,192.168.1.1/24", ","),
			expectedErrors:           0,
		},
		{
			name:                     "incorrect LoadBalancerSourceRanges",
			loadBalancerSourceRanges: strings.Split("192.168.0.1,192.168.1.1", ","),
			expectedErrors:           3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor := newTidbMonitor()
			monitor.Spec.Prometheus.Service.LoadBalancerSourceRanges = tt.loadBalancerSourceRanges
			monitor.Spec.Grafana.Service.LoadBalancerSourceRanges = tt.loadBalancerSourceRanges
			monitor.Spec.Reloader.Service.LoadBalancerSourceRanges = tt.loadBalancerSourceRanges
			err := ValidateTidbMonitor(monitor)
			r := len(err)
			g.Expect(r).Should(Equal(tt.expectedErrors))
			if r > 0 {
				for _, e := range err {
					g.Expect(e.Detail).To(ContainSubstring("service.Spec.LoadBalancerSourceRanges is not valid. Expecting a list of IP ranges. For example, 10.0.0.0/24."))
				}
			}
		})
	}
}

func TestValidateDMCluster(t *testing.T) {
	g := NewGomegaWithT(t)
	tests := []struct {
		name              string
		version           string
		discoveryAddr     string
		masterReplicas    int32
		masterStorageSize string
		expectedError     string
	}{
		{
			name:          "invalid version",
			version:       "v1.0.6",
			discoveryAddr: "http://basic-discovery.demo:10261",
			expectedError: "dm cluster version can't set to v1.x.y",
		},
		{
			name:          "empty discovery address",
			expectedError: "discovery.address must not be empty",
		},
		{
			name:           "dm-master storageSize not given",
			version:        "v2.0.0-rc.2",
			discoveryAddr:  "http://basic-discovery.demo:10261",
			masterReplicas: 3,
			expectedError:  "storageSize must not be empty",
		},
		{
			name:              "correct configuration",
			version:           "nightly",
			discoveryAddr:     "http://basic-discovery.demo:10261",
			masterReplicas:    3,
			masterStorageSize: "10Gi",
			expectedError:     "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dc := newDMCluster()
			dc.Spec.Version = tt.version
			dc.Spec.Discovery.Address = tt.discoveryAddr
			dc.Spec.Master.Replicas = tt.masterReplicas
			dc.Spec.Master.StorageSize = tt.masterStorageSize
			err := ValidateDMCluster(dc)
			if tt.expectedError != "" {
				g.Expect(len(err)).Should(Equal(1))
				g.Expect(err[0].Detail).To(ContainSubstring(tt.expectedError))
			}
		})
	}
}

func newTidbCluster() *v1alpha1.TidbCluster {
	tc := &v1alpha1.TidbCluster{
		Spec: v1alpha1.TidbClusterSpec{
			PD:   &v1alpha1.PDSpec{},
			TiKV: &v1alpha1.TiKVSpec{},
			TiDB: &v1alpha1.TiDBSpec{},
		},
	}
	tc.Name = "test-validate-requests-storage"
	tc.Namespace = "default"
	return tc
}

func newService() *v1alpha1.ServiceSpec {
	svc := &v1alpha1.ServiceSpec{}
	return svc
}

func newTidbMonitor() *v1alpha1.TidbMonitor {
	monitor := &v1alpha1.TidbMonitor{
		Spec: v1alpha1.TidbMonitorSpec{
			Grafana:    &v1alpha1.GrafanaSpec{},
			Prometheus: v1alpha1.PrometheusSpec{},
			Reloader:   v1alpha1.ReloaderSpec{},
		},
	}
	return monitor
}

func newDMCluster() *v1alpha1.DMCluster {
	dc := &v1alpha1.DMCluster{
		Spec: v1alpha1.DMClusterSpec{
			Discovery: v1alpha1.DMDiscoverySpec{},
			Master:    v1alpha1.MasterSpec{},
			Worker:    &v1alpha1.WorkerSpec{},
		},
	}
	dc.Name = "test-validate-dm-cluster"
	dc.Namespace = "default"
	return dc
}

func TestValidateLocalDescendingPath(t *testing.T) {
	successCases := []string{
		"data",
		"foo/data",
	}

	for _, c := range successCases {
		errs := validateLocalDescendingPath(c, field.NewPath("dataSubDir"))
		if len(errs) > 0 {
			t.Errorf("expected success: %v", errs)
		}
	}

	errorCases := []string{
		"/data",
		"../data",
		"../foo/data",
	}

	for _, c := range errorCases {
		errs := validateLocalDescendingPath(c, field.NewPath("dataSubDir"))
		if len(errs) == 0 {
			t.Errorf("expected failure for %s", c)
		}
	}
}

func TestValidateEvictLeaderTimeout(t *testing.T) {
	successCases := []*string{
		nil,
		pointer.StringPtr("3h"),
		pointer.StringPtr("5m30s"),
	}

	for _, c := range successCases {
		errs := validateTimeDurationStr(c, field.NewPath("evictLeaderTimeout"))
		if len(errs) > 0 {
			t.Errorf("expected success: %v", errs)
		}
	}

	errorCases := []*string{
		pointer.StringPtr("evict"),
		pointer.StringPtr("-5m30s"),
	}

	for _, c := range errorCases {
		errs := validateTimeDurationStr(c, field.NewPath("evictLeaderTimeout"))
		if len(errs) == 0 {
			t.Errorf("expected failure for %s", *c)
		}
	}
}

func TestValidatePDAddresses(t *testing.T) {
	successCases := [][]string{
		{
			"http://1.2.3.4:2379",
			"http://test-pd-0.test-pd-peer.default.svc:2380",
			"http://test:2379",
		},
	}

	for _, c := range successCases {
		errs := validatePDAddresses(c, field.NewPath("pdAddresses"))
		if len(errs) > 0 {
			t.Errorf("expected success: %v", errs)
		}
	}

	errorCases := [][]string{
		{
			"https://1.2.3.4:2379",
		},
		{
			"http://1.2.3.4:2380",
			"https://1.2.3.4:2379",
		},
		{
			"test-pd-0.test-pd-peer.default.svc:2380",
		},
	}

	for _, c := range errorCases {
		errs := validatePDAddresses(c, field.NewPath("pdAddresses"))
		if len(errs) == 0 {
			t.Errorf("expected failure for %s", c)
		}
	}
}
