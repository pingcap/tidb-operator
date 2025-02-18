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

package runtime

import "github.com/pingcap/tidb-operator/pkg/client"

type Group interface {
	Object

	SetReplicas(replicas int32)
	Replicas() int32

	SetVersion(versions string)
	Version() string

	SetStatusVersion(version string)
	StatusVersion() string
	SetStatusReplicas(replicas, ready, update, current int32)
	StatusReplicas() (replicas, ready, update, current int32)
	SetStatusRevision(update, current string, collisionCount *int32)
	StatusRevision() (update, current string, collisionCount *int32)
}

type GroupT[T GroupSet] interface {
	Group

	*T
}

type GroupSet interface {
	PDGroup | TiDBGroup | TiKVGroup | TiFlashGroup | TiCDCGroup
}

type GroupTuple[PT client.Object, PU Group] interface {
	Tuple[PT, PU]
}
