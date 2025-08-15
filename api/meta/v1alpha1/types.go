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

package v1alpha1

const (
	// Finalizer is the finalizer used by all resources managed by TiDB Operator.
	Finalizer = "pingcap.com/finalizer"
)

const (
	// NamePrefix for "names" in k8s resources
	// Users may overlay some fields in managed resource such as pods. Names with this
	// prefix is preserved to avoid conflicts with fields defined by users.
	NamePrefix = "ti-"

	// VolNamePrefix is defined for custom persistent volume which may have name conflicts
	// with the volumes managed by tidb operator
	VolNamePrefix = NamePrefix + "vol-"
)

type Component string

const (
	// component name
	ComponentPD                Component = "pd"
	ComponentTiDB              Component = "tidb"
	ComponentTiKV              Component = "tikv"
	ComponentTiFlash           Component = "tiflash"
	ComponentTiCDC             Component = "ticdc"
	ComponentTSO               Component = "tso"
	ComponentScheduler         Component = "scheduler"
	ComponentTiProxy           Component = "tiproxy"
	ComponentReplicationWorker Component = "repl-worker"
)
