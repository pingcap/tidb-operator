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

package discovery

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	discoveryfake "k8s.io/client-go/discovery/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestIsAPIGroupVersionSupported(t *testing.T) {
	tests := []struct {
		name         string
		groupVersion string
		wantOK       bool
	}{
		{
			name:         "found",
			groupVersion: "apiextensions.k8s.io/v1beta1",
			wantOK:       true,
		},
		{
			name:         "not found",
			groupVersion: "apiextensions.k8s.io/v1",
			wantOK:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &k8stesting.Fake{
				Resources: []*metav1.APIResourceList{
					{
						GroupVersion: "apiextensions.k8s.io/v1beta1",
						APIResources: []metav1.APIResource{
							{
								Name:    "customresourcedefinitions",
								Group:   "apiextensions.k8s.io",
								Version: "v1beta1",
							},
						},
					},
				},
			}
			discoveryClient := &discoveryfake.FakeDiscovery{
				Fake: fake,
			}
			ok, _ := IsAPIGroupVersionSupported(discoveryClient, tt.groupVersion)
			if ok != tt.wantOK {
				t.Errorf("got %v, want %v", ok, tt.wantOK)
			}
		})
	}
}

func TestIsAPIGroupSupported(t *testing.T) {
	tests := []struct {
		name   string
		group  string
		wantOK bool
	}{
		{
			name:   "found",
			group:  "apiextensions.k8s.io",
			wantOK: true,
		},
		{
			name:   "not found",
			group:  "apps.pingcap.com",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &k8stesting.Fake{
				Resources: []*metav1.APIResourceList{
					{
						GroupVersion: "apiextensions.k8s.io/v1beta1",
						APIResources: []metav1.APIResource{
							{
								Name:    "customresourcedefinitions",
								Group:   "apiextensions.k8s.io",
								Version: "v1beta1",
							},
						},
					},
				},
			}
			discoveryClient := &discoveryfake.FakeDiscovery{
				Fake: fake,
			}
			ok, _ := IsAPIGroupSupported(discoveryClient, tt.group)
			if ok != tt.wantOK {
				t.Errorf("got %v, want %v", ok, tt.wantOK)
			}
		})
	}
}
