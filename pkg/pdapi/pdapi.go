// Copyright 2018 PingCAP, Inc.
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
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/kvproto/pkg/pdpb"
	"github.com/pingcap/tidb-operator/pkg/util/crypto"
	httputil "github.com/pingcap/tidb-operator/pkg/util/http"
	"github.com/tikv/pd/pkg/typeutil"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"
)

const (
	DefaultTimeout       = 5 * time.Second
	evictSchedulerLeader = "evict-leader-scheduler"
	tiKVNotBootstrapped  = `TiKV cluster not bootstrapped, please start TiKV first"`
)

// GetTLSConfig returns *tls.Config for given TiDB cluster.
func GetTLSConfig(secretLister corelisterv1.SecretLister, namespace Namespace, secretName string) (*tls.Config, error) {
	secret, err := secretLister.Secrets(string(namespace)).Get(secretName)
	if err != nil {
		return nil, fmt.Errorf("unable to load certificates from secret %s/%s: %v", namespace, secretName, err)
	}

	return crypto.LoadTlsConfigFromSecret(secret)
}

// PDClient provides pd server's api
type PDClient interface {
	// GetHealth returns the PD's health info
	GetHealth() (*HealthInfo, error)
	// GetConfig returns PD's config
	GetConfig() (*PDConfigFromAPI, error)
	// GetCluster returns used when syncing pod labels.
	GetCluster() (*metapb.Cluster, error)
	// GetMembers returns all PD members from cluster
	GetMembers() (*MembersInfo, error)
	// GetStores lists all TiKV stores from cluster
	GetStores() (*StoresInfo, error)
	// GetTombStoneStores lists all tombstone stores from cluster
	GetTombStoneStores() (*StoresInfo, error)
	// GetStore gets a TiKV store for a specific store id from cluster
	GetStore(storeID uint64) (*StoreInfo, error)
	// SetStoreLabels compares store labels with node labels
	// for historic reasons, PD stores TiKV labels as []*StoreLabel which is a key-value pair slice
	SetStoreLabels(storeID uint64, labels map[string]string) (bool, error)
	// UpdateReplicationConfig updates the replication config
	UpdateReplicationConfig(config PDReplicationConfig) error
	// DeleteStore deletes a TiKV store from cluster
	DeleteStore(storeID uint64) error
	// SetStoreState sets store to specified state.
	SetStoreState(storeID uint64, state string) error
	// DeleteMember deletes a PD member from cluster
	DeleteMember(name string) error
	// DeleteMemberByID deletes a PD member from cluster
	DeleteMemberByID(memberID uint64) error
	// BeginEvictLeader initiates leader eviction for a storeID.
	// This is used when upgrading a pod.
	BeginEvictLeader(storeID uint64) error
	// EndEvictLeader is used at the end of pod upgrade.
	EndEvictLeader(storeID uint64) error
	// GetEvictLeaderSchedulers gets schedulers of evict leader
	GetEvictLeaderSchedulers() ([]string, error)
	// GetEvictLeaderSchedulersForStores gets schedulers of evict leader for given stores
	GetEvictLeaderSchedulersForStores(storeIDs ...uint64) (map[uint64]string, error)
	// GetPDLeader returns pd leader
	GetPDLeader() (*pdpb.Member, error)
	// TransferPDLeader transfers pd leader to specified member
	TransferPDLeader(name string) error
	// GetAutoscalingPlans returns the scaling plan for the cluster
	GetAutoscalingPlans(strategy Strategy) ([]Plan, error)
	// GetRecoveringMark return the pd recovering mark
	GetRecoveringMark() (bool, error)

	// GetReady checks if a specific PD member is ready.
	// NOTE: in order to call this method, a PDClient for a specific PD member (`GetPDClientForMember`) is required.
	GetReady() (bool, error)

	// GetMSMembers returns all PDMS members service-addr from cluster by specific microservice
	GetMSMembers(service string) ([]string, error)
	// GetMSPrimary returns the primary PDMS member service-addr from cluster by specific microservice
	GetMSPrimary(service string) (string, error)
}

var (
	healthPrefix           = "pd/api/v1/health"
	membersPrefix          = "pd/api/v1/members"
	storesPrefix           = "pd/api/v1/stores"
	storePrefix            = "pd/api/v1/store"
	configPrefix           = "pd/api/v1/config"
	clusterIDPrefix        = "pd/api/v1/cluster"
	schedulersPrefix       = "pd/api/v1/schedulers"
	pdLeaderPrefix         = "pd/api/v1/leader"
	pdLeaderTransferPrefix = "pd/api/v1/leader/transfer"
	pdReplicationPrefix    = "pd/api/v1/config/replicate"
	// evictLeaderSchedulerConfigPrefix is the prefix of evict-leader-scheduler
	// config API, available since PD v3.1.0.
	evictLeaderSchedulerConfigPrefix = "pd/api/v1/scheduler-config/evict-leader-scheduler/list"
	autoscalingPrefix                = "autoscaling"
	recoveringMarkPrefix             = "pd/api/v1/admin/cluster/markers/snapshot-recovering"

	readyPrefix = "pd/api/v2/ready"

	// microservice
	MicroservicePrefix = "pd/api/v2/ms"
)

// pdClient is default implementation of PDClient
type pdClient struct {
	url        string
	httpClient *http.Client
}

// NewPDClient returns a new PDClient
func NewPDClient(url string, timeout time.Duration, tlsConfig *tls.Config) PDClient {
	var disableKeepalive bool
	if tlsConfig != nil {
		disableKeepalive = true
	}
	return &pdClient{
		url: url,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: &http.Transport{TLSClientConfig: tlsConfig, DisableKeepAlives: disableKeepalive},
		},
	}
}

// following struct definitions are copied from github.com/pingcap/pd/server/api/store
// these are not exported by that package

// HealthInfo define PD's healthy info
type HealthInfo struct {
	Healths []MemberHealth
}

// MemberHealth define a pd member's healthy info
type MemberHealth struct {
	Name       string   `json:"name"`
	MemberID   uint64   `json:"member_id"`
	ClientUrls []string `json:"client_urls"`
	Health     bool     `json:"health"`
}

// MetaStore is TiKV store status defined in protobuf
type MetaStore struct {
	*metapb.Store
	StateName string `json:"state_name"`
}

// StoreStatus is TiKV store status returned from PD RESTful interface
type StoreStatus struct {
	Capacity           typeutil.ByteSize `json:"capacity"`
	Available          typeutil.ByteSize `json:"available"`
	LeaderCount        int               `json:"leader_count"`
	RegionCount        int               `json:"region_count"`
	SendingSnapCount   uint32            `json:"sending_snap_count"`
	ReceivingSnapCount uint32            `json:"receiving_snap_count"`
	ApplyingSnapCount  uint32            `json:"applying_snap_count"`
	IsBusy             bool              `json:"is_busy"`

	StartTS         time.Time         `json:"start_ts"`
	LastHeartbeatTS time.Time         `json:"last_heartbeat_ts"`
	Uptime          typeutil.Duration `json:"uptime"`
}

// StoreInfo is a single store info returned from PD RESTful interface
type StoreInfo struct {
	Store  *MetaStore   `json:"store"`
	Status *StoreStatus `json:"status"`
}

// StoresInfo is stores info returned from PD RESTful interface
type StoresInfo struct {
	Count  int          `json:"count"`
	Stores []*StoreInfo `json:"stores"`
}

// MembersInfo is PD members info returned from PD RESTful interface
// type Members map[string][]*pdpb.Member
type MembersInfo struct {
	Header     *pdpb.ResponseHeader `json:"header,omitempty"`
	Members    []*pdpb.Member       `json:"members,omitempty"`
	Leader     *pdpb.Member         `json:"leader,omitempty"`
	EtcdLeader *pdpb.Member         `json:"etcd_leader,omitempty"`
}

// ServiceRegistryEntry is the registry entry of PD microservice
type ServiceRegistryEntry struct {
	ServiceAddr    string `json:"service-addr"`
	Version        string `json:"version"`
	GitHash        string `json:"git-hash"`
	DeployPath     string `json:"deploy-path"`
	StartTimestamp int64  `json:"start-timestamp"`
}

// below copied from github.com/tikv/pd/pkg/autoscaling

// Strategy within an HTTP request provides rules and resources to help make decision for auto scaling.
type Strategy struct {
	Rules     []*Rule     `json:"rules"`
	Resources []*Resource `json:"resources"`
}

// Rule is a set of constraints for a kind of component.
type Rule struct {
	Component   string       `json:"component"`
	CPURule     *CPURule     `json:"cpu_rule,omitempty"`
	StorageRule *StorageRule `json:"storage_rule,omitempty"`
}

// CPURule is the constraints about CPU.
type CPURule struct {
	MaxThreshold  float64  `json:"max_threshold"`
	MinThreshold  float64  `json:"min_threshold"`
	ResourceTypes []string `json:"resource_types"`
}

// StorageRule is the constraints about storage.
type StorageRule struct {
	MinThreshold  float64  `json:"min_threshold"`
	ResourceTypes []string `json:"resource_types"`
}

// Resource represents a kind of resource set including CPU, memory, storage.
type Resource struct {
	ResourceType string `json:"resource_type"`
	// The basic unit of CPU is milli-core.
	CPU uint64 `json:"cpu"`
	// The basic unit of memory is byte.
	Memory uint64 `json:"memory"`
	// The basic unit of storage is byte.
	Storage uint64 `json:"storage"`
	// If count is not set, it indicates no limit.
	Count *uint64 `json:"count,omitempty"`
}

// Plan is the final result of auto scaling, which indicates how to scale in or scale out.
type Plan struct {
	Component    string            `json:"component"`
	Count        uint64            `json:"count"`
	ResourceType string            `json:"resource_type"`
	Labels       map[string]string `json:"labels"`
}

type schedulerInfo struct {
	Name    string `json:"name"`
	StoreID uint64 `json:"store_id"`
}

type RecoveringMark struct {
	Mark bool `json:"marked"`
}

func (c *pdClient) GetHealth() (*HealthInfo, error) {
	apiURL := fmt.Sprintf("%s/%s", c.url, healthPrefix)
	body, err := httputil.GetBodyOK(c.httpClient, apiURL)
	if err != nil {
		return nil, err
	}
	var healths []MemberHealth
	err = json.Unmarshal(body, &healths)
	if err != nil {
		return nil, err
	}
	return &HealthInfo{
		healths,
	}, nil
}

func (c *pdClient) GetConfig() (*PDConfigFromAPI, error) {
	apiURL := fmt.Sprintf("%s/%s", c.url, configPrefix)
	body, err := httputil.GetBodyOK(c.httpClient, apiURL)
	if err != nil {
		return nil, err
	}
	config := &PDConfigFromAPI{}
	err = json.Unmarshal(body, config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func (c *pdClient) GetCluster() (*metapb.Cluster, error) {
	apiURL := fmt.Sprintf("%s/%s", c.url, clusterIDPrefix)
	body, err := httputil.GetBodyOK(c.httpClient, apiURL)
	if err != nil {
		return nil, err
	}
	cluster := &metapb.Cluster{}
	err = json.Unmarshal(body, cluster)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

func (c *pdClient) GetMembers() (*MembersInfo, error) {
	apiURL := fmt.Sprintf("%s/%s", c.url, membersPrefix)
	body, err := httputil.GetBodyOK(c.httpClient, apiURL)
	if err != nil {
		return nil, err
	}
	members := &MembersInfo{}
	err = json.Unmarshal(body, members)
	if err != nil {
		return nil, err
	}
	return members, nil
}

func (c *pdClient) GetMSMembers(service string) ([]string, error) {
	apiURL := fmt.Sprintf("%s/%s/members/%s", c.url, MicroservicePrefix, service)
	body, err := httputil.GetBodyOK(c.httpClient, apiURL)
	if err != nil {
		return nil, err
	}
	var members []ServiceRegistryEntry
	err = json.Unmarshal(body, &members)
	if err != nil {
		return nil, err
	}
	var addrs []string
	for _, member := range members {
		addrs = append(addrs, member.ServiceAddr)
	}
	return addrs, nil
}

func (c *pdClient) GetMSPrimary(service string) (string, error) {
	apiURL := fmt.Sprintf("%s/%s/primary/%s", c.url, MicroservicePrefix, service)
	body, err := httputil.GetBodyOK(c.httpClient, apiURL)
	if err != nil {
		return "", err
	}
	var primary string
	err = json.Unmarshal(body, &primary)
	if err != nil {
		return "", err
	}

	return primary, nil
}

func (c *pdClient) getStores(apiURL string) (*StoresInfo, error) {
	body, err := httputil.GetBodyOK(c.httpClient, apiURL)
	if err != nil {
		return nil, err
	}
	storesInfo := &StoresInfo{}
	err = json.Unmarshal(body, storesInfo)
	if err != nil {
		return nil, err
	}
	return storesInfo, nil
}

func (c *pdClient) GetStores() (*StoresInfo, error) {
	storesInfo, err := c.getStores(fmt.Sprintf("%s/%s", c.url, storesPrefix))
	if err != nil {
		if strings.HasSuffix(err.Error(), tiKVNotBootstrapped+"\n") {
			err = TiKVNotBootstrappedErrorf(err.Error())
		}
		return nil, err
	}
	return storesInfo, nil
}

func (c *pdClient) GetTombStoneStores() (*StoresInfo, error) {
	return c.getStores(fmt.Sprintf("%s/%s?state=%d", c.url, storesPrefix, metapb.StoreState_Tombstone))
}

func (c *pdClient) GetStore(storeID uint64) (*StoreInfo, error) {
	apiURL := fmt.Sprintf("%s/%s/%d", c.url, storePrefix, storeID)
	body, err := httputil.GetBodyOK(c.httpClient, apiURL)
	if err != nil {
		return nil, err
	}
	storeInfo := &StoreInfo{}
	err = json.Unmarshal(body, storeInfo)
	if err != nil {
		return nil, err
	}
	return storeInfo, nil
}

func (c *pdClient) DeleteStore(storeID uint64) error {
	var exist bool
	stores, err := c.GetStores()
	if err != nil {
		return err
	}
	for _, store := range stores.Stores {
		if store.Store.GetId() == storeID {
			exist = true
			break
		}
	}
	if !exist {
		return nil
	}
	apiURL := fmt.Sprintf("%s/%s/%d", c.url, storePrefix, storeID)
	req, err := http.NewRequest("DELETE", apiURL, nil)
	if err != nil {
		return err
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer httputil.DeferClose(res.Body)

	// Remove an offline store should return http.StatusOK
	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusNotFound {
		return nil
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	return fmt.Errorf("failed to delete store %d: %v", storeID, string(body))
}

// SetStoreState sets store to specified state.
func (c *pdClient) SetStoreState(storeID uint64, state string) error {
	apiURL := fmt.Sprintf("%s/%s/%d/state?state=%s", c.url, storePrefix, storeID, state)
	req, err := http.NewRequest("POST", apiURL, nil)
	if err != nil {
		return err
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer httputil.DeferClose(res.Body)

	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusNotFound {
		return nil
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	return fmt.Errorf("failed to delete store %d: %v", storeID, string(body))
}

func (c *pdClient) DeleteMemberByID(memberID uint64) error {
	var exist bool
	members, err := c.GetMembers()
	if err != nil {
		return err
	}
	for _, member := range members.Members {
		if member.MemberId == memberID {
			exist = true
			break
		}
	}
	if !exist {
		return nil
	}
	apiURL := fmt.Sprintf("%s/%s/id/%d", c.url, membersPrefix, memberID)
	req, err := http.NewRequest("DELETE", apiURL, nil)
	if err != nil {
		return err
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer httputil.DeferClose(res.Body)
	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusNotFound {
		return nil
	}
	err2 := httputil.ReadErrorBody(res.Body)
	return fmt.Errorf("failed %v to delete member %d: %v", res.StatusCode, memberID, err2)
}

func (c *pdClient) DeleteMember(name string) error {
	var exist bool
	members, err := c.GetMembers()
	if err != nil {
		return err
	}
	for _, member := range members.Members {
		if member.Name == name {
			exist = true
			break
		}
	}
	if !exist {
		return nil
	}
	apiURL := fmt.Sprintf("%s/%s/name/%s", c.url, membersPrefix, name)
	req, err := http.NewRequest("DELETE", apiURL, nil)
	if err != nil {
		return err
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer httputil.DeferClose(res.Body)
	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusNotFound {
		return nil
	}
	err2 := httputil.ReadErrorBody(res.Body)
	return fmt.Errorf("failed %v to delete member %s: %v", res.StatusCode, name, err2)
}

func (c *pdClient) SetStoreLabels(storeID uint64, labels map[string]string) (bool, error) {
	apiURL := fmt.Sprintf("%s/%s/%d/label", c.url, storePrefix, storeID)
	data, err := json.Marshal(labels)
	if err != nil {
		return false, err
	}
	res, err := c.httpClient.Post(apiURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return false, err
	}
	defer httputil.DeferClose(res.Body)
	if res.StatusCode == http.StatusOK {
		return true, nil
	}
	err2 := httputil.ReadErrorBody(res.Body)
	return false, fmt.Errorf("failed %v to set store labels: %v", res.StatusCode, err2)
}

func (c *pdClient) UpdateReplicationConfig(config PDReplicationConfig) error {
	apiURL := fmt.Sprintf("%s/%s", c.url, pdReplicationPrefix)
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	res, err := c.httpClient.Post(apiURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer httputil.DeferClose(res.Body)
	if res.StatusCode == http.StatusOK {
		return nil
	}
	err = httputil.ReadErrorBody(res.Body)
	return fmt.Errorf("failed %v to update replication: %v", res.StatusCode, err)
}

func (c *pdClient) BeginEvictLeader(storeID uint64) error {
	leaderEvictInfo := getLeaderEvictSchedulerInfo(storeID)
	apiURL := fmt.Sprintf("%s/%s", c.url, schedulersPrefix)
	data, err := json.Marshal(leaderEvictInfo)
	if err != nil {
		return err
	}
	res, err := c.httpClient.Post(apiURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer httputil.DeferClose(res.Body)
	if res.StatusCode == http.StatusOK {
		return nil
	}

	// pd will return an error with the body contains "scheduler existed" if the scheduler already exists
	// this is not the standard response.
	// so these lines are just a workaround for now:
	//   - make a new request to get all schedulers
	//   - return nil if the scheduler already exists
	//
	// when PD returns standard json response, we should get rid of this verbose code.
	evictLeaderSchedulers, err := c.GetEvictLeaderSchedulers()
	if err != nil {
		return err
	}
	for _, s := range evictLeaderSchedulers {
		if s == getLeaderEvictSchedulerStr(storeID) {
			return nil
		}
	}

	err2 := httputil.ReadErrorBody(res.Body)
	return fmt.Errorf("failed %v to begin evict leader of store:[%d],error: %v", res.StatusCode, storeID, err2)
}

func (c *pdClient) EndEvictLeader(storeID uint64) error {
	sName := getLeaderEvictSchedulerStr(storeID)
	apiURL := fmt.Sprintf("%s/%s/%s", c.url, schedulersPrefix, sName)
	req, err := http.NewRequest("DELETE", apiURL, nil)
	if err != nil {
		return err
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer httputil.DeferClose(res.Body)
	if res.StatusCode == http.StatusNotFound {
		return nil
	}
	if res.StatusCode == http.StatusOK {
		klog.Infof("call DELETE method: %s success", apiURL)
	} else {
		err2 := httputil.ReadErrorBody(res.Body)
		klog.Errorf("call DELETE method: %s failed,statusCode: %v,error: %v", apiURL, res.StatusCode, err2)
	}

	// pd will return an error with the body contains "scheduler not found" if the scheduler is not found
	// this is not the standard response.
	// so these lines are just a workaround for now:
	//   - make a new request to get all schedulers
	//   - return nil if the scheduler is not found
	//
	// when PD returns standard json response, we should get rid of this verbose code.
	evictLeaderSchedulers, err := c.GetEvictLeaderSchedulers()
	if err != nil {
		return err
	}
	for _, s := range evictLeaderSchedulers {
		if s == sName {
			return fmt.Errorf("end leader evict scheduler failed,the store:[%d]'s leader evict scheduler is still exist", storeID)
		}
	}

	return nil
}

func (c *pdClient) GetEvictLeaderSchedulers() ([]string, error) {
	apiURL := fmt.Sprintf("%s/%s", c.url, schedulersPrefix)
	body, err := httputil.GetBodyOK(c.httpClient, apiURL)
	if err != nil {
		return nil, err
	}
	var schedulers []string
	err = json.Unmarshal(body, &schedulers)
	if err != nil {
		return nil, err
	}
	var evicts []string
	for _, scheduler := range schedulers {
		if strings.HasPrefix(scheduler, evictSchedulerLeader) {
			evicts = append(evicts, scheduler)
		}
	}
	evictSchedulers, err := c.filterLeaderEvictScheduler(evicts)
	if err != nil {
		return nil, err
	}
	return evictSchedulers, nil
}

func (c *pdClient) GetEvictLeaderSchedulersForStores(storeIDs ...uint64) (map[uint64]string, error) {
	schedulers, err := c.GetEvictLeaderSchedulers()
	if err != nil {
		return nil, err
	}

	find := func(id uint64) string {
		for _, scheduler := range schedulers {
			sName := getLeaderEvictSchedulerStr(id)
			if scheduler == sName {
				return scheduler
			}
		}
		return ""
	}

	result := make(map[uint64]string)
	for _, id := range storeIDs {
		if scheduler := find(id); scheduler != "" {
			result[id] = scheduler
		}
	}

	return result, nil
}

// getEvictLeaderSchedulerConfig gets the config of PD scheduler "evict-leader-scheduler"
// It's available since PD 3.1.0.
// In the previous versions, PD API returns 404 and this function will return an error.
func (c *pdClient) getEvictLeaderSchedulerConfig() (*evictLeaderSchedulerConfig, error) {
	apiURL := fmt.Sprintf("%s/%s", c.url, evictLeaderSchedulerConfigPrefix)
	body, err := httputil.GetBodyOK(c.httpClient, apiURL)
	if err != nil {
		return nil, err
	}
	config := &evictLeaderSchedulerConfig{}
	err = json.Unmarshal(body, config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

// This method is to make compatible between old pdapi version and versions after 3.1/4.0.
// To get more detail, see:
// - https://github.com/pingcap/tidb-operator/pull/1831
// - https://github.com/pingcap/pd/issues/2550
func (c *pdClient) filterLeaderEvictScheduler(evictLeaderSchedulers []string) ([]string, error) {
	var schedulerIds []string
	if len(evictLeaderSchedulers) == 1 && evictLeaderSchedulers[0] == evictSchedulerLeader {
		// If there is only one evcit scehduler entry without store ID postfix.
		// We should get the store IDs via scheduler config API and append them
		// to provide consistent results.
		c, err := c.getEvictLeaderSchedulerConfig()
		if err != nil {
			return nil, err
		}
		for k := range c.StoreIDWithRanges {
			schedulerIds = append(schedulerIds, fmt.Sprintf("%s-%v", evictSchedulerLeader, k))
		}
	} else {
		schedulerIds = append(schedulerIds, evictLeaderSchedulers...)
	}
	return schedulerIds, nil
}

func (c *pdClient) GetRecoveringMark() (bool, error) {
	apiURL := fmt.Sprintf("%s/%s", c.url, recoveringMarkPrefix)
	body, err := httputil.GetBodyOK(c.httpClient, apiURL)
	if err != nil {
		return false, err
	}
	recoveringMark := &RecoveringMark{}
	err = json.Unmarshal(body, recoveringMark)
	if err != nil {
		return false, err
	}
	return recoveringMark.Mark, nil
}

func (c *pdClient) GetPDLeader() (*pdpb.Member, error) {
	apiURL := fmt.Sprintf("%s/%s", c.url, pdLeaderPrefix)
	body, err := httputil.GetBodyOK(c.httpClient, apiURL)
	if err != nil {
		return nil, err
	}
	leader := &pdpb.Member{}
	err = json.Unmarshal(body, leader)
	if err != nil {
		return nil, err
	}
	return leader, nil
}

func (c *pdClient) TransferPDLeader(memberName string) error {
	apiURL := fmt.Sprintf("%s/%s/%s", c.url, pdLeaderTransferPrefix, memberName)
	req, err := http.NewRequest("POST", apiURL, nil)
	if err != nil {
		return err
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer httputil.DeferClose(res.Body)
	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusNotFound {
		return nil
	}
	err2 := httputil.ReadErrorBody(res.Body)
	return fmt.Errorf("failed %v to transfer pd leader to %s,error: %v", res.StatusCode, memberName, err2)
}

func (c *pdClient) GetAutoscalingPlans(strategy Strategy) ([]Plan, error) {
	apiURL := fmt.Sprintf("%s/%s", c.url, autoscalingPrefix)
	data, err := json.Marshal(strategy)
	if err != nil {
		return nil, err
	}
	body, err := httputil.PostBodyOK(c.httpClient, apiURL, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	var plans []Plan
	err = json.Unmarshal(body, &plans)
	if err != nil {
		return nil, err
	}
	return plans, nil
}

func (c *pdClient) GetReady() (bool, error) {
	apiURL := fmt.Sprintf("%s/%s", c.url, readyPrefix)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return false, err
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer httputil.DeferClose(res.Body)

	if res.StatusCode == http.StatusNotFound {
		// this ready API is added from v8.5.2, so we return true if the status code is 404 here
		klog.Info("ready API is not found, assuming PD is ready")
		return true, nil
	}

	if res.StatusCode != http.StatusOK {
		err2 := httputil.ReadErrorBody(res.Body)
		return false, fmt.Errorf("failed %v to get ready status: %v", res.StatusCode, err2)
	}
	return true, nil
}

func getLeaderEvictSchedulerInfo(storeID uint64) *schedulerInfo {
	return &schedulerInfo{"evict-leader-scheduler", storeID}
}

func getLeaderEvictSchedulerStr(storeID uint64) string {
	return fmt.Sprintf("%s-%d", "evict-leader-scheduler", storeID)
}

// TiKVNotBootstrappedError represents that TiKV cluster is not bootstrapped yet
type TiKVNotBootstrappedError struct {
	s string
}

func (e *TiKVNotBootstrappedError) Error() string {
	return e.s
}

// TiKVNotBootstrappedErrorf returns a TiKVNotBootstrappedError
func TiKVNotBootstrappedErrorf(format string, a ...interface{}) error {
	return &TiKVNotBootstrappedError{fmt.Sprintf(format, a...)}
}

// IsTiKVNotBootstrappedError returns whether err is a TiKVNotBootstrappedError
func IsTiKVNotBootstrappedError(err error) bool {
	_, ok := err.(*TiKVNotBootstrappedError)
	return ok
}
