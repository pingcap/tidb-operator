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

package tidbapi

// ServerInfo is the information of a TiDB server.
// ref https://github.com/pingcap/tidb/blob/v8.1.0/pkg/server/handler/tikvhandler/tikv_handler.go#L1696
type ServerInfo struct {
	IsOwner bool `json:"is_owner"`
}
