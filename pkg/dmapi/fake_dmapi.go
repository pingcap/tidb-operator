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
	GetMastersActionType  ActionType = "GetMasters"
	GetWorkersActionType  ActionType = "GetWorkers"
	GetLeaderActionType   ActionType = "GetLeader"
	EvictLeaderActionType ActionType = "EvictLeader"
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

func (pc *FakeMasterClient) AddReaction(actionType ActionType, reaction Reaction) {
	pc.reactions[actionType] = reaction
}

// fakeAPI is a small helper for fake API calls
func (pc *FakeMasterClient) fakeAPI(actionType ActionType, action *Action) (interface{}, error) {
	if reaction, ok := pc.reactions[actionType]; ok {
		result, err := reaction(action)
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return nil, &NotFoundReaction{actionType}
}

func (pc *FakeMasterClient) GetMasters() ([]*MastersInfo, error) {
	action := &Action{}
	result, err := pc.fakeAPI(GetMastersActionType, action)
	if err != nil {
		return nil, err
	}
	return result.([]*MastersInfo), nil
}

func (pc *FakeMasterClient) GetWorkers() ([]*WorkersInfo, error) {
	action := &Action{}
	result, err := pc.fakeAPI(GetWorkersActionType, action)
	if err != nil {
		return nil, err
	}
	return result.([]*WorkersInfo), nil
}

func (pc *FakeMasterClient) GetLeader() (MembersLeader, error) {
	action := &Action{}
	result, err := pc.fakeAPI(GetLeaderActionType, action)
	if err != nil {
		return MembersLeader{}, err
	}
	return result.(MembersLeader), nil
}

func (pc *FakeMasterClient) EvictLeader() error {
	action := &Action{}
	_, err := pc.fakeAPI(EvictLeaderActionType, action)
	if err != nil {
		return err
	}
	return err
}
