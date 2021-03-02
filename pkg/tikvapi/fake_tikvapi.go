// Copyright 2021 PingCAP, Inc.
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

package tikvapi

import (
	"fmt"
)

type ActionType string

const (
	GetLeaderCountActionType ActionType = "GetLeaderCount"
)

type NotFoundReaction struct {
	actionType ActionType
}

func (nfr *NotFoundReaction) Error() string {
	return fmt.Sprintf("not found %s reaction. Please add the reaction", nfr.actionType)
}

type Action struct {
	ID     uint64
	Name   string
	Labels map[string]string
}

type Reaction func(action *Action) (interface{}, error)

// FakeTiKVClient implements a fake version of TiKVClient.
type FakeTiKVClient struct {
	reactions map[ActionType]Reaction
}

func NewFakeTiKVClient() *FakeTiKVClient {
	return &FakeTiKVClient{reactions: map[ActionType]Reaction{}}
}

func (c *FakeTiKVClient) AddReaction(actionType ActionType, reaction Reaction) {
	c.reactions[actionType] = reaction
}

// fakeAPI is a small helper for fake API calls
func (c *FakeTiKVClient) fakeAPI(actionType ActionType, action *Action) (interface{}, error) {
	if reaction, ok := c.reactions[actionType]; ok {
		result, err := reaction(action)
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return nil, &NotFoundReaction{actionType}
}

func (c *FakeTiKVClient) GetLeaderCount() (int, error) {
	action := &Action{}
	result, err := c.fakeAPI(GetLeaderCountActionType, action)
	if err != nil {
		return 0, err
	}
	return result.(int), nil
}
