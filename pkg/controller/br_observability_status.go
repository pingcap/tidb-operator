// Copyright 2026 PingCAP, Inc.
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

package controller

import (
	"reflect"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
)

const maxBROperations = 10

func updateBROperations(existing []v1alpha1.BROperation, observed *v1alpha1.BROperation) ([]v1alpha1.BROperation, bool) {
	if observed == nil || observed.OperationID == "" {
		return existing, false
	}

	next := make([]v1alpha1.BROperation, 0, len(existing)+1)
	next = append(next, *observed.DeepCopy())
	for _, item := range existing {
		if item.OperationID == observed.OperationID {
			continue
		}
		next = append(next, item)
	}
	if len(next) > maxBROperations {
		next = next[:maxBROperations]
	}

	return next, !reflect.DeepEqual(existing, next)
}
