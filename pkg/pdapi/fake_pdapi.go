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

package pdapi

import (
	"fmt"

	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/kvproto/pkg/pdpb"
)

type ActionType string

const (
	GetHealthActionType                ActionType = "GetHealth"
	GetConfigActionType                ActionType = "GetConfig"
	GetClusterActionType               ActionType = "GetCluster"
	GetMembersActionType               ActionType = "GetMembers"
	GetStoresActionType                ActionType = "GetStores"
	GetTombStoneStoresActionType       ActionType = "GetTombStoneStores"
	GetStoreActionType                 ActionType = "GetStore"
	DeleteStoreActionType              ActionType = "DeleteStore"
	SetStoreStateActionType            ActionType = "SetStoreState"
	DeleteMemberByIDActionType         ActionType = "DeleteMemberByID"
	DeleteMemberActionType             ActionType = "DeleteMember "
	SetStoreLabelsActionType           ActionType = "SetStoreLabels"
	UpdateReplicationActionType        ActionType = "UpdateReplicationConfig"
	BeginEvictLeaderActionType         ActionType = "BeginEvictLeader"
	EndEvictLeaderActionType           ActionType = "EndEvictLeader"
	GetEvictLeaderSchedulersActionType ActionType = "GetEvictLeaderSchedulers"
	GetPDLeaderActionType              ActionType = "GetPDLeader"
	TransferPDLeaderActionType         ActionType = "TransferPDLeader"
)

type NotFoundReaction struct {
	actionType ActionType
}

func (nfr *NotFoundReaction) Error() string {
	return fmt.Sprintf("not found %s reaction. Please add the reaction", nfr.actionType)
}

type Action struct {
	ID          uint64
	Name        string
	Labels      map[string]string
	Replication PDReplicationConfig
}

type Reaction func(action *Action) (interface{}, error)

// FakePDClient implements a fake version of PDClient.
type FakePDClient struct {
	reactions map[ActionType]Reaction
}

func NewFakePDClient() *FakePDClient {
	return &FakePDClient{reactions: map[ActionType]Reaction{}}
}

func (c *FakePDClient) AddReaction(actionType ActionType, reaction Reaction) {
	c.reactions[actionType] = reaction
}

// fakeAPI is a small helper for fake API calls
func (c *FakePDClient) fakeAPI(actionType ActionType, action *Action) (interface{}, error) {
	if reaction, ok := c.reactions[actionType]; ok {
		result, err := reaction(action)
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return nil, &NotFoundReaction{actionType}
}

func (c *FakePDClient) GetHealth() (*HealthInfo, error) {
	action := &Action{}
	result, err := c.fakeAPI(GetHealthActionType, action)
	if err != nil {
		return nil, err
	}
	return result.(*HealthInfo), nil
}

func (c *FakePDClient) GetConfig() (*PDConfigFromAPI, error) {
	action := &Action{}
	result, err := c.fakeAPI(GetConfigActionType, action)
	if err != nil {
		return nil, err
	}
	return result.(*PDConfigFromAPI), nil
}

func (c *FakePDClient) GetCluster() (*metapb.Cluster, error) {
	action := &Action{}
	result, err := c.fakeAPI(GetClusterActionType, action)
	if err != nil {
		return nil, err
	}
	return result.(*metapb.Cluster), nil
}

func (c *FakePDClient) GetMembers() (*MembersInfo, error) {
	action := &Action{}
	result, err := c.fakeAPI(GetMembersActionType, action)
	if err != nil {
		return nil, err
	}
	return result.(*MembersInfo), nil
}

func (c *FakePDClient) GetStores() (*StoresInfo, error) {
	action := &Action{}
	result, err := c.fakeAPI(GetStoresActionType, action)
	if err != nil {
		return nil, err
	}
	return result.(*StoresInfo), nil
}

func (c *FakePDClient) GetTombStoneStores() (*StoresInfo, error) {
	action := &Action{}
	result, err := c.fakeAPI(GetTombStoneStoresActionType, action)
	if err != nil {
		return nil, err
	}
	return result.(*StoresInfo), nil
}

func (c *FakePDClient) GetStore(id uint64) (*StoreInfo, error) {
	action := &Action{
		ID: id,
	}
	result, err := c.fakeAPI(GetStoreActionType, action)
	if err != nil {
		return nil, err
	}
	return result.(*StoreInfo), nil
}

func (c *FakePDClient) DeleteStore(id uint64) error {
	if reaction, ok := c.reactions[DeleteStoreActionType]; ok {
		action := &Action{ID: id}
		_, err := reaction(action)
		return err
	}
	return nil
}

func (c *FakePDClient) SetStoreState(id uint64, state string) error {
	if reaction, ok := c.reactions[SetStoreStateActionType]; ok {
		action := &Action{ID: id}
		_, err := reaction(action)
		return err
	}
	return nil
}

func (c *FakePDClient) DeleteMemberByID(id uint64) error {
	if reaction, ok := c.reactions[DeleteMemberByIDActionType]; ok {
		action := &Action{ID: id}
		_, err := reaction(action)
		return err
	}
	return nil
}

func (c *FakePDClient) DeleteMember(name string) error {
	if reaction, ok := c.reactions[DeleteMemberActionType]; ok {
		action := &Action{Name: name}
		_, err := reaction(action)
		return err
	}
	return nil
}

// SetStoreLabels sets TiKV labels
func (c *FakePDClient) SetStoreLabels(storeID uint64, labels map[string]string) (bool, error) {
	if reaction, ok := c.reactions[SetStoreLabelsActionType]; ok {
		action := &Action{ID: storeID, Labels: labels}
		result, err := reaction(action)
		return result.(bool), err
	}
	return true, nil
}

// UpdateReplicationConfig updates the replication config
func (c *FakePDClient) UpdateReplicationConfig(config PDReplicationConfig) error {
	if reaction, ok := c.reactions[UpdateReplicationActionType]; ok {
		action := &Action{Replication: config}
		_, err := reaction(action)
		return err
	}
	return nil
}

func (c *FakePDClient) BeginEvictLeader(storeID uint64) error {
	if reaction, ok := c.reactions[BeginEvictLeaderActionType]; ok {
		action := &Action{ID: storeID}
		_, err := reaction(action)
		return err
	}
	return nil
}

func (c *FakePDClient) EndEvictLeader(storeID uint64) error {
	if reaction, ok := c.reactions[EndEvictLeaderActionType]; ok {
		action := &Action{ID: storeID}
		_, err := reaction(action)
		return err
	}
	return nil
}

func (c *FakePDClient) GetEvictLeaderSchedulers() ([]string, error) {
	if reaction, ok := c.reactions[GetEvictLeaderSchedulersActionType]; ok {
		action := &Action{}
		result, err := reaction(action)
		return result.([]string), err
	}
	return nil, nil
}

func (c *FakePDClient) GetPDLeader() (*pdpb.Member, error) {
	if reaction, ok := c.reactions[GetPDLeaderActionType]; ok {
		action := &Action{}
		result, err := reaction(action)
		return result.(*pdpb.Member), err
	}
	return nil, nil
}

func (c *FakePDClient) TransferPDLeader(memberName string) error {
	if reaction, ok := c.reactions[TransferPDLeaderActionType]; ok {
		action := &Action{Name: memberName}
		_, err := reaction(action)
		return err
	}
	return nil
}
