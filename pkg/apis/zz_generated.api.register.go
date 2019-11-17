/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// This file was autogenerated by apiregister-gen. Do not edit it manually!

package apis

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap"
	_ "github.com/pingcap/tidb-operator/pkg/apis/pingcap/install"
	pingcapv1alpha2 "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha2"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/apiserver-builder-alpha/pkg/builders"
)

var (
	localSchemeBuilder = runtime.SchemeBuilder{
		pingcapv1alpha2.AddToScheme,
	}
	AddToScheme = localSchemeBuilder.AddToScheme
)

// GetAllApiBuilders returns all known APIGroupBuilders
// so they can be registered with the apiserver
func GetAllApiBuilders() []*builders.APIGroupBuilder {
	return []*builders.APIGroupBuilder{
		GetPingcapAPIBuilder(),
	}
}

var pingcapApiGroup = builders.NewApiGroupBuilder(
	"pingcap.com",
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap").
	WithUnVersionedApi(pingcap.ApiVersion).
	WithVersionedApis(
		pingcapv1alpha2.ApiVersion,
	).
	WithRootScopedKinds()

func GetPingcapAPIBuilder() *builders.APIGroupBuilder {
	return pingcapApiGroup
}
