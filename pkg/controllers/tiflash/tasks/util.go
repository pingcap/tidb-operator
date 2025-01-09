// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tasks

import (
	"fmt"
)

func ConfigMapName(tiflashName string) string {
	return tiflashName
}

func PersistentVolumeClaimName(tiflashName string, volIndex int) string {
	// ref: https://github.com/pingcap/tidb-operator/blob/486cc85c8380efc4f36b3125a1abba9e3146a2c8/pkg/apis/pingcap/v1alpha1/helpers.go#L105
	return fmt.Sprintf("data%d-%s", volIndex, tiflashName)
}
