// Copyright 2023 PingCAP, Inc.
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
	"time"

	constants "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
)

func (vbks *VolumeBackupSchedule) GetBackupCRDName(timestamp time.Time) string {
	return fmt.Sprintf("%s-%s", vbks.GetName(), timestamp.UTC().Format(constants.BackupNameTimeFormat))
}
