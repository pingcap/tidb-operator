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

package tiflashapi

import (
	"fmt"
)

type ActionType string

const (
	GetStoreStatusActionType ActionType = "GetStoreStatus"
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

var _ TiFlashClient = &FakeTiFlashClient{}

// FakeTiFlashClient implements a fake version of TiFlashClient.
type FakeTiFlashClient struct {
	reactions map[ActionType]Reaction
}

func NewFakeTiFlashClient() *FakeTiFlashClient {
	return &FakeTiFlashClient{reactions: map[ActionType]Reaction{}}
}

func (c *FakeTiFlashClient) AddReaction(actionType ActionType, reaction Reaction) {
	c.reactions[actionType] = reaction
}

// fakeAPI is a small helper for fake API calls
func (c *FakeTiFlashClient) fakeAPI(actionType ActionType, action *Action) (interface{}, error) {
	if reaction, ok := c.reactions[actionType]; ok {
		result, err := reaction(action)
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return nil, &NotFoundReaction{actionType}
}

func (c *FakeTiFlashClient) GetStoreStatus() (Status, error) {
	action := &Action{}
	result, err := c.fakeAPI(GetStoreStatusActionType, action)
	if err != nil {
		return Status(""), err
	}
	return result.(Status), nil
}
