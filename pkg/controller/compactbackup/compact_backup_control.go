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
// See the License for the specific language governing permissions and
// limitations under the License.

package compact

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
)

// ControlInterface implements the control logic for updating Compact Backup
// It is implemented as an interface to allow for extensions that provide different semantics.
// Currently, there is only one implementation.
type ControlInterface interface {
	// UpdateStatus updates the status for a Compact Backup, include condition and status info
	UpdateStatus(backup *v1alpha1.Backup, condition *v1alpha1.BackupCondition, newStatus *controller.BackupUpdateStatus) error
}

type defaultCompactControl struct {
	cli           versioned.Interface
	backupManager backup.BackupManager
}

// NewDefaultCompactControl returns a new instance of the default implementation ControlInterface for Compact Backup
func NewDefaultCompactControl(
	cli versioned.Interface,
	deps *controller.Dependencies) ControlInterface {
	
}