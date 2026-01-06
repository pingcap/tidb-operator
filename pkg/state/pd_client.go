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

package state

import (
	"github.com/pingcap/tidb-operator/v2/pkg/timanager"
	pdm "github.com/pingcap/tidb-operator/v2/pkg/timanager/pd"
)

type IPDClient interface {
	GetPDClient(m pdm.PDClientManager) (pdm.PDClient, bool)
}

type pdClientState struct {
	c ICluster

	pdClient pdm.PDClient
}

func (s *pdClientState) GetPDClient(m pdm.PDClientManager) (pdm.PDClient, bool) {
	if s.pdClient != nil {
		return s.pdClient, true
	}

	ck := s.c.Cluster()
	c, ok := m.Get(timanager.PrimaryKey(ck.Namespace, ck.Name))
	if !ok {
		return nil, false
	}
	if !c.HasSynced() {
		return nil, false
	}

	s.pdClient = c
	return s.pdClient, true
}

func NewPDClientState(c ICluster) IPDClient {
	return &pdClientState{
		c: c,
	}
}
