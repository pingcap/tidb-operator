// Copyright 2020 PingCAP, Inc.
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

package dmapi

import (
	"fmt"
)

type ActionType string

const (
	GetMastersActionType   ActionType = "GetMasters"
	GetWorkersActionType   ActionType = "GetWorkers"
	GetLeaderActionType    ActionType = "GetLeader"
	EvictLeaderActionType  ActionType = "EvictLeader"
	DeleteMasterActionType ActionType = "DeleteMaster"
	DeleteWorkerActionType ActionType = "DeleteWorker"
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

// FakeMasterClient implements a fake version of MasterClient.
type FakeMasterClient struct {
	reactions map[ActionType]Reaction
}

func NewFakeMasterClient() *FakeMasterClient {
	return &FakeMasterClient{reactions: map[ActionType]Reaction{}}
}

func (c *FakeMasterClient) AddReaction(actionType ActionType, reaction Reaction) {
	c.reactions[actionType] = reaction
}

// fakeAPI is a small helper for fake API calls
func (c *FakeMasterClient) fakeAPI(actionType ActionType, action *Action) (interface{}, error) {
	if reaction, ok := c.reactions[actionType]; ok {
		result, err := reaction(action)
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return nil, &NotFoundReaction{actionType}
}

func (c *FakeMasterClient) GetMasters() ([]*MastersInfo, error) {
	action := &Action{}
	result, err := c.fakeAPI(GetMastersActionType, action)
	if err != nil {
		return nil, err
	}
	return result.([]*MastersInfo), nil
}

func (c *FakeMasterClient) GetWorkers() ([]*WorkersInfo, error) {
	action := &Action{}
	result, err := c.fakeAPI(GetWorkersActionType, action)
	if err != nil {
		return nil, err
	}
	return result.([]*WorkersInfo), nil
}

func (c *FakeMasterClient) GetLeader() (MembersLeader, error) {
	action := &Action{}
	result, err := c.fakeAPI(GetLeaderActionType, action)
	if err != nil {
		return MembersLeader{}, err
	}
	return result.(MembersLeader), nil
}

func (c *FakeMasterClient) EvictLeader() error {
	action := &Action{}
	_, err := c.fakeAPI(EvictLeaderActionType, action)
	return err
}

func (c *FakeMasterClient) DeleteMaster(_ string) error {
	action := &Action{}
	_, err := c.fakeAPI(DeleteMasterActionType, action)
	return err
}

func (c *FakeMasterClient) DeleteWorker(_ string) error {
	action := &Action{}
	_, err := c.fakeAPI(DeleteWorkerActionType, action)
	return err
}
