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

package compatibility

// All these are features supported by different TiDB versions
// NOTE: Do not add constraints with prerelease version
// NOTE: MUST add key PRs in comments
var (
	// PDMS is supported
	// This feature is enabled after the tso primary transferring is supported
	// See https://github.com/tikv/pd/pull/8157
	PDMS = MustNewConstraints(">= v8.3.0")

	// PD ready api
	// See https://github.com/tikv/pd/pull/8749
	PDReadyAPI = MustNewConstraints(">= v9.0.0 || ^v8.5.2")

	// TiKV ready api
	// See https://github.com/tikv/tikv/pull/18237
	TiKVReadyAPI = MustNewConstraints(">= v9.0.0")
)
