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
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/controllers/common"
	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

type Config struct {
	BackupManagerImage string
}
type ReconcileContext struct {
	PDClientManager pdm.PDClientManager
	State

	// mark pdgroup is bootstrapped if cache of pd is synced
	IsBootstrapped bool

	Config Config
}

func TaskContextRestore(state RestoreStateInitializer, c client.Client) task.Task {
	w := state.RestoreInitializer()
	return common.TaskContextResource("Restore", w, c, false)
}
