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

package v1beta1

import (
	"github.com/pingcap/tidb-operator/tests/pkg/apiserver/apis/example"
	"k8s.io/apimachinery/pkg/conversion"
)

// Convert_v1beta1_PodSpec_To_example_PodSpec override the generated conversion func
func Convert_v1beta1_PodSpec_To_example_PodSpec(in *PodSpec, out *example.PodSpec, s conversion.Scope) error {
	// call default
	if err := autoConvert_v1beta1_PodSpec_To_example_PodSpec(in, out, s); err != nil {
		return err
	}
	out.HasTolerations = true
	return nil
}

// Convert_example_PodSpec_To_v1beta1_PodSpec override the generated conversion func
func Convert_example_PodSpec_To_v1beta1_PodSpec(in *example.PodSpec, out *PodSpec, s conversion.Scope) error {
	// use default
	return autoConvert_example_PodSpec_To_v1beta1_PodSpec(in, out, s)
}
