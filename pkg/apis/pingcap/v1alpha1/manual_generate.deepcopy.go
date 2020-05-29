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

package v1alpha1

import (
	"fmt"
	"reflect"

	"gopkg.in/yaml.v2"
)

func (in *WaitForLockTimeout) DeepCopyInto(out *WaitForLockTimeout) {
	if in == nil {
		return
	}
	if in.Values == nil {
		out.Values = nil
		return
	}
	switch in.Values.(type) {
	case string:
		out.Values = in.Values.(string)
	case int:
		out.Values = in.Values.(int)
	default:
		fmt.Println(reflect.TypeOf(in.Values))
	}
	return
}

func (in *WakeUpDelayDuration) DeepCopyInto(out *WakeUpDelayDuration) {
	if in == nil {
		return
	}
	b, err := yaml.Marshal(in.Values)
	if err != nil {
		return
	}
	var values interface{}
	err = yaml.Unmarshal(b, &values)
	if err != nil {
		return
	}
	out.Values = values
}
