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
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/kvproto/pkg/pdpb"
	pd "github.com/tikv/pd/client/http"
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
	GetAutoscalingPlansActionType      ActionType = "GetAutoscalingPlans"
	GetRecoveringMarkActionType        ActionType = "GetRecoveringMark"
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
	Replication pd.ReplicationConfig
}

type Reaction func(action *Action) (interface{}, error)

// FakePDClient implements a fake version of PDClient.
type FakePDClient struct {
	reactions  map[ActionType]Reaction
	schedulers []string
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

func (c *FakePDClient) WithCallerID(s string) pd.Client {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) WithRespHandler(f func(resp *http.Response, res interface{}) error) pd.Client {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) Close() {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) GetStoresByState(ctx context.Context, state metapb.StoreState) (*pd.StoresInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) GetScheduleConfig(ctx context.Context) (*pd.ScheduleConfig, error) {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) SetScheduleConfig(ctx context.Context, config *pd.ScheduleConfig) error {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) GetRecoveringMark(ctx context.Context) (bool, error) {
	action := &Action{}
	_, err := c.fakeAPI(GetRecoveringMarkActionType, action)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (c *FakePDClient) GetConfig(ctx context.Context) (*pd.ServerConfig, error) {
	action := &Action{}
	result, err := c.fakeAPI(GetConfigActionType, action)
	if err != nil {
		return nil, err
	}
	return result.(*pd.ServerConfig), nil
}

func (c *FakePDClient) GetStore(ctx context.Context, id uint64) (*pd.StoreInfo, error) {
	action := &Action{
		ID: id,
	}
	result, err := c.fakeAPI(GetStoreActionType, action)
	if err != nil {
		return nil, err
	}
	return result.(*pd.StoreInfo), nil
}

func (c *FakePDClient) SetStoreLabels(ctx context.Context, storeID uint64, labels map[string]string) (bool, error) {
	if reaction, ok := c.reactions[SetStoreLabelsActionType]; ok {
		action := &Action{ID: storeID, Labels: labels}
		result, err := reaction(action)
		return result.(bool), err
	}
	return true, nil
}

func (c *FakePDClient) DeleteStore(ctx context.Context, id uint64) error {
	if reaction, ok := c.reactions[DeleteStoreActionType]; ok {
		action := &Action{ID: id}
		_, err := reaction(action)
		return err
	}
	return nil
}

func (c *FakePDClient) SetStoreState(ctx context.Context, id uint64, s string) error {
	if reaction, ok := c.reactions[SetStoreStateActionType]; ok {
		action := &Action{ID: id}
		_, err := reaction(action)
		return err
	}
	return nil
}

func (c *FakePDClient) GetEvictLeaderSchedulerConfig(ctx context.Context) (*pd.EvictLeaderSchedulerConfig, error) {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) GetRegionByID(ctx context.Context, u uint64) (*pd.RegionInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) GetRegionByKey(ctx context.Context, bytes []byte) (*pd.RegionInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) GetRegions(ctx context.Context) (*pd.RegionsInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) GetRegionsByKeyRange(ctx context.Context, keyRange *pd.KeyRange, i int) (*pd.RegionsInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) GetRegionsByStoreID(ctx context.Context, u uint64) (*pd.RegionsInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) GetRegionsReplicatedStateByKeyRange(ctx context.Context, keyRange *pd.KeyRange) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) GetHotReadRegions(ctx context.Context) (*pd.StoreHotPeersInfos, error) {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) GetHotWriteRegions(ctx context.Context) (*pd.StoreHotPeersInfos, error) {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) GetHistoryHotRegions(ctx context.Context, request *pd.HistoryHotRegionsRequest) (*pd.HistoryHotRegions, error) {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) GetRegionStatusByKeyRange(ctx context.Context, keyRange *pd.KeyRange, b bool) (*pd.RegionStats, error) {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) GetStores(ctx context.Context) (*pd.StoresInfo, error) {
	action := &Action{}
	result, err := c.fakeAPI(GetStoresActionType, action)
	if err != nil {
		return nil, err
	}
	return result.(*pd.StoresInfo), nil
}

func (c *FakePDClient) GetAllPlacementRuleBundles(ctx context.Context) ([]*pd.GroupBundle, error) {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) GetPlacementRuleBundleByGroup(ctx context.Context, s string) (*pd.GroupBundle, error) {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) GetPlacementRulesByGroup(ctx context.Context, s string) ([]*pd.Rule, error) {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) SetPlacementRule(ctx context.Context, rule *pd.Rule) error {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) SetPlacementRuleInBatch(ctx context.Context, ops []*pd.RuleOp) error {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) SetPlacementRuleBundles(ctx context.Context, bundles []*pd.GroupBundle, b bool) error {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) DeletePlacementRule(ctx context.Context, s string, s2 string) error {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) GetAllPlacementRuleGroups(ctx context.Context) ([]*pd.RuleGroup, error) {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) GetPlacementRuleGroupByID(ctx context.Context, s string) (*pd.RuleGroup, error) {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) SetPlacementRuleGroup(ctx context.Context, group *pd.RuleGroup) error {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) DeletePlacementRuleGroupByID(ctx context.Context, s string) error {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) GetAllRegionLabelRules(ctx context.Context) ([]*pd.LabelRule, error) {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) GetRegionLabelRulesByIDs(ctx context.Context, strings []string) ([]*pd.LabelRule, error) {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) SetRegionLabelRule(ctx context.Context, rule *pd.LabelRule) error {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) PatchRegionLabelRules(ctx context.Context, patch *pd.LabelRulePatch) error {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) AccelerateSchedule(ctx context.Context, keyRange *pd.KeyRange) error {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) AccelerateScheduleInBatch(ctx context.Context, ranges []*pd.KeyRange) error {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) GetMinResolvedTSByStoresIDs(ctx context.Context, uint64s []uint64) (uint64, map[uint64]uint64, error) {
	//TODO implement me
	panic("implement me")
}

func (c *FakePDClient) GetHealth(ctx context.Context) (*pd.HealthInfo, error) {
	action := &Action{}
	result, err := c.fakeAPI(GetHealthActionType, action)
	if err != nil {
		return nil, err
	}
	return result.(*pd.HealthInfo), nil
}

func (c *FakePDClient) GetMembers(ctx context.Context) (*pd.MembersInfo, error) {
	action := &Action{}
	result, err := c.fakeAPI(GetMembersActionType, action)
	if err != nil {
		return nil, err
	}
	return result.(*pd.MembersInfo), nil
}

func (c *FakePDClient) GetCluster(ctx context.Context) (*metapb.Cluster, error) {
	action := &Action{}
	result, err := c.fakeAPI(GetClusterActionType, action)
	if err != nil {
		return nil, err
	}
	return result.(*metapb.Cluster), nil
}

func (c *FakePDClient) GetPDLeader(ctx context.Context) (*pdpb.Member, error) {
	if reaction, ok := c.reactions[GetPDLeaderActionType]; ok {
		action := &Action{}
		result, err := reaction(action)
		return result.(*pdpb.Member), err
	}
	return nil, nil
}

func (c *FakePDClient) GetAutoscalingPlans(ctx context.Context, strategy pd.Strategy) ([]pd.Plan, error) {
	if reaction, ok := c.reactions[GetAutoscalingPlansActionType]; ok {
		action := &Action{}
		result, err := reaction(action)
		return result.([]pd.Plan), err
	}
	return nil, nil
}

func (c *FakePDClient) UpdateReplicationConfig(ctx context.Context, config pd.ReplicationConfig) error {
	if reaction, ok := c.reactions[UpdateReplicationActionType]; ok {
		action := &Action{Replication: config}
		_, err := reaction(action)
		return err
	}
	return nil
}

func (c *FakePDClient) TransferPDLeader(ctx context.Context, memberName string) error {
	if reaction, ok := c.reactions[TransferPDLeaderActionType]; ok {
		action := &Action{Name: memberName}
		_, err := reaction(action)
		return err
	}
	return nil
}

func (c *FakePDClient) DeleteMemberByID(ctx context.Context, id uint64) error {
	if reaction, ok := c.reactions[DeleteMemberByIDActionType]; ok {
		action := &Action{ID: id}
		_, err := reaction(action)
		return err
	}
	return nil
}

func (c *FakePDClient) DeleteMember(ctx context.Context, name string) error {
	if reaction, ok := c.reactions[DeleteMemberActionType]; ok {
		action := &Action{Name: name}
		_, err := reaction(action)
		return err
	}
	return nil
}

func (c *FakePDClient) CreateScheduler(ctx context.Context, name string, storeID uint64) error {
	name = fmt.Sprintf("%s-%v", name, storeID)
	for _, scheduler := range c.schedulers {
		if scheduler == name {
			return nil
		}
	}
	c.schedulers = append(c.schedulers, name)
	return nil
}

func (c *FakePDClient) GetSchedulers(ctx context.Context) ([]string, error) {
	return c.schedulers, nil
}

func (c *FakePDClient) DeleteScheduler(ctx context.Context, name string, storeID uint64) error {
	name = fmt.Sprintf("%s-%v", name, storeID)
	for i := 0; i < len(c.schedulers); i++ {
		if c.schedulers[i] == name {
			c.schedulers = append(c.schedulers[:i], c.schedulers[i+1:]...)
			return nil
		}
	}
	return errors.New("scheduler not found")
}

func (c *FakePDClient) GetEvictLeaderSchedulers() ([]string, error) {
	if reaction, ok := c.reactions[GetEvictLeaderSchedulersActionType]; ok {
		action := &Action{}
		result, err := reaction(action)
		return result.([]string), err
	}
	return nil, nil
}

func (c *FakePDClient) GetEvictLeaderSchedulersForStores(storeIDs ...uint64) (map[uint64]string, error) {
	if reaction, ok := c.reactions[GetEvictLeaderSchedulersActionType]; ok {
		action := &Action{}
		result, err := reaction(action)
		return result.(map[uint64]string), err
	}
	return nil, nil
}
