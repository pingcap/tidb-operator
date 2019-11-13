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

package manager

import (
	"fmt"
)

// Manager to manager fault trigger
type Manager struct {
	VMManager
}

// NewManager returns a manager instance
func NewManager(vmManagerName string) *Manager {
	var vmManager VMManager
	if vmManagerName == "qm" {
		vmManager = &QMVMManager{}
	} else if vmManagerName == "virsh" {
		vmManager = &VirshVMManager{}
	} else {
		panic(fmt.Errorf("stability test have not supported the vm manager:[%s],please choose [qm] or [virsh]", vmManagerName))
	}
	return &Manager{
		vmManager,
	}
}
