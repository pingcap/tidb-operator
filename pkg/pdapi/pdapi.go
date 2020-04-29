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
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/kvproto/pkg/pdpb"
	"github.com/pingcap/pd/pkg/typeutil"
	"github.com/pingcap/tidb-operator/pkg/httputil"
	"github.com/pingcap/tidb-operator/pkg/util"
	"github.com/pingcap/tidb-operator/pkg/util/crypto"
	types "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

const (
	DefaultTimeout = 5 * time.Second
)

// Namespace is a newtype of a string
type Namespace string

// PDControlInterface is an interface that knows how to manage and get tidb cluster's PD client
type PDControlInterface interface {
	// GetPDClient provides PDClient of the tidb cluster.
	GetPDClient(Namespace, string, bool) PDClient
}

// defaultPDControl is the default implementation of PDControlInterface.
type defaultPDControl struct {
	mutex     sync.Mutex
	kubeCli   kubernetes.Interface
	pdClients map[string]PDClient
}

// NewDefaultPDControl returns a defaultPDControl instance
func NewDefaultPDControl(kubeCli kubernetes.Interface) PDControlInterface {
	return &defaultPDControl{kubeCli: kubeCli, pdClients: map[string]PDClient{}}
}

// GetTLSConfig returns *tls.Config for given TiDB cluster.
// It loads in-cluster root ca if caCert is empty.
func GetTLSConfig(kubeCli kubernetes.Interface, namespace Namespace, tcName string, caCert []byte) (*tls.Config, error) {
	secretName := util.ClusterClientTLSSecretName(tcName)
	secret, err := kubeCli.CoreV1().Secrets(string(namespace)).Get(secretName, types.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to load certificates from secret %s/%s: %v", namespace, secretName, err)
	}

	return crypto.LoadTlsConfigFromSecret(secret, caCert)
}

// GetPDClient provides a PDClient of real pd cluster,if the PDClient not existing, it will create new one.
func (pdc *defaultPDControl) GetPDClient(namespace Namespace, tcName string, tlsEnabled bool) PDClient {
	pdc.mutex.Lock()
	defer pdc.mutex.Unlock()

	var tlsConfig *tls.Config
	var err error
	var scheme = "http"

	if tlsEnabled {
		scheme = "https"
		tlsConfig, err = GetTLSConfig(pdc.kubeCli, namespace, tcName, nil)
		if err != nil {
			klog.Errorf("Unable to get tls config for tidb cluster %q, pd client may not work: %v", tcName, err)
			return &pdClient{url: PdClientURL(namespace, tcName, scheme), httpClient: &http.Client{Timeout: DefaultTimeout}}
		}

		return NewPDClient(PdClientURL(namespace, tcName, scheme), DefaultTimeout, tlsConfig)
	}

	key := pdClientKey(scheme, namespace, tcName)
	if _, ok := pdc.pdClients[key]; !ok {
		pdc.pdClients[key] = NewPDClient(PdClientURL(namespace, tcName, scheme), DefaultTimeout, nil)
	}
	return pdc.pdClients[key]
}

// pdClientKey returns the pd client key
func pdClientKey(scheme string, namespace Namespace, clusterName string) string {
	return fmt.Sprintf("%s.%s.%s", scheme, clusterName, string(namespace))
}

// pdClientUrl builds the url of pd client
func PdClientURL(namespace Namespace, clusterName string, scheme string) string {
	return fmt.Sprintf("%s://%s-pd.%s:2379", scheme, clusterName, string(namespace))
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
	// storeLabelsEqualNodeLabels compares store labels with node labels
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
	// GetPDLeader returns pd leader
	GetPDLeader() (*pdpb.Member, error)
	// TransferPDLeader transfers pd leader to specified member
	TransferPDLeader(name string) error
}

var (
	healthPrefix           = "pd/health"
	membersPrefix          = "pd/api/v1/members"
	storesPrefix           = "pd/api/v1/stores"
	storePrefix            = "pd/api/v1/store"
	configPrefix           = "pd/api/v1/config"
	clusterIDPrefix        = "pd/api/v1/cluster"
	schedulersPrefix       = "pd/api/v1/schedulers"
	pdLeaderPrefix         = "pd/api/v1/leader"
	pdLeaderTransferPrefix = "pd/api/v1/leader/transfer"
	pdReplicationPrefix    = "pd/api/v1/config/replicate"
)

// pdClient is default implementation of PDClient
type pdClient struct {
	url        string
	httpClient *http.Client
}

// NewPDClient returns a new PDClient
func NewPDClient(url string, timeout time.Duration, tlsConfig *tls.Config) PDClient {
	return &pdClient{
		url: url,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: &http.Transport{TLSClientConfig: tlsConfig},
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
//type Members map[string][]*pdpb.Member
type MembersInfo struct {
	Header     *pdpb.ResponseHeader `json:"header,omitempty"`
	Members    []*pdpb.Member       `json:"members,omitempty"`
	Leader     *pdpb.Member         `json:"leader,omitempty"`
	EtcdLeader *pdpb.Member         `json:"etcd_leader,omitempty"`
}

type schedulerInfo struct {
	Name    string `json:"name"`
	StoreID uint64 `json:"store_id"`
}

func (pc *pdClient) GetHealth() (*HealthInfo, error) {
	apiURL := fmt.Sprintf("%s/%s", pc.url, healthPrefix)
	body, err := httputil.GetBodyOK(pc.httpClient, apiURL)
	if err != nil {
		return nil, err
	}
	healths := []MemberHealth{}
	err = json.Unmarshal(body, &healths)
	if err != nil {
		return nil, err
	}
	return &HealthInfo{
		healths,
	}, nil
}

func (pc *pdClient) GetConfig() (*PDConfigFromAPI, error) {
	apiURL := fmt.Sprintf("%s/%s", pc.url, configPrefix)
	body, err := httputil.GetBodyOK(pc.httpClient, apiURL)
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

func (pc *pdClient) GetCluster() (*metapb.Cluster, error) {
	apiURL := fmt.Sprintf("%s/%s", pc.url, clusterIDPrefix)
	body, err := httputil.GetBodyOK(pc.httpClient, apiURL)
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

func (pc *pdClient) GetMembers() (*MembersInfo, error) {
	apiURL := fmt.Sprintf("%s/%s", pc.url, membersPrefix)
	body, err := httputil.GetBodyOK(pc.httpClient, apiURL)
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

func (pc *pdClient) GetStores() (*StoresInfo, error) {
	apiURL := fmt.Sprintf("%s/%s", pc.url, storesPrefix)
	body, err := httputil.GetBodyOK(pc.httpClient, apiURL)
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

func (pc *pdClient) GetTombStoneStores() (*StoresInfo, error) {
	apiURL := fmt.Sprintf("%s/%s?state=%d", pc.url, storesPrefix, metapb.StoreState_Tombstone)
	body, err := httputil.GetBodyOK(pc.httpClient, apiURL)
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

func (pc *pdClient) GetStore(storeID uint64) (*StoreInfo, error) {
	apiURL := fmt.Sprintf("%s/%s/%d", pc.url, storePrefix, storeID)
	body, err := httputil.GetBodyOK(pc.httpClient, apiURL)
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

func (pc *pdClient) DeleteStore(storeID uint64) error {
	var exist bool
	stores, err := pc.GetStores()
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
	apiURL := fmt.Sprintf("%s/%s/%d", pc.url, storePrefix, storeID)
	req, err := http.NewRequest("DELETE", apiURL, nil)
	if err != nil {
		return err
	}
	res, err := pc.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer httputil.DeferClose(res.Body)

	// Remove an offline store should returns http.StatusOK
	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusNotFound {
		return nil
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	return fmt.Errorf("failed to delete store %d: %v", storeID, string(body))
}

// SetStoreState sets store to specified state.
func (pc *pdClient) SetStoreState(storeID uint64, state string) error {
	apiURL := fmt.Sprintf("%s/%s/%d/state?state=%s", pc.url, storePrefix, storeID, state)
	req, err := http.NewRequest("POST", apiURL, nil)
	if err != nil {
		return err
	}
	res, err := pc.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer httputil.DeferClose(res.Body)

	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusNotFound {
		return nil
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	return fmt.Errorf("failed to delete store %d: %v", storeID, string(body))
}

func (pc *pdClient) DeleteMemberByID(memberID uint64) error {
	var exist bool
	members, err := pc.GetMembers()
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
	apiURL := fmt.Sprintf("%s/%s/id/%d", pc.url, membersPrefix, memberID)
	req, err := http.NewRequest("DELETE", apiURL, nil)
	if err != nil {
		return err
	}
	res, err := pc.httpClient.Do(req)
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

func (pc *pdClient) DeleteMember(name string) error {
	var exist bool
	members, err := pc.GetMembers()
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
	apiURL := fmt.Sprintf("%s/%s/name/%s", pc.url, membersPrefix, name)
	req, err := http.NewRequest("DELETE", apiURL, nil)
	if err != nil {
		return err
	}
	res, err := pc.httpClient.Do(req)
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

func (pc *pdClient) SetStoreLabels(storeID uint64, labels map[string]string) (bool, error) {
	apiURL := fmt.Sprintf("%s/%s/%d/label", pc.url, storePrefix, storeID)
	data, err := json.Marshal(labels)
	if err != nil {
		return false, err
	}
	res, err := pc.httpClient.Post(apiURL, "application/json", bytes.NewBuffer(data))
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

func (pc *pdClient) UpdateReplicationConfig(config PDReplicationConfig) error {
	apiURL := fmt.Sprintf("%s/%s", pc.url, pdReplicationPrefix)
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	res, err := pc.httpClient.Post(apiURL, "application/json", bytes.NewBuffer(data))
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

func (pc *pdClient) BeginEvictLeader(storeID uint64) error {
	leaderEvictInfo := getLeaderEvictSchedulerInfo(storeID)
	apiURL := fmt.Sprintf("%s/%s", pc.url, schedulersPrefix)
	data, err := json.Marshal(leaderEvictInfo)
	if err != nil {
		return err
	}
	res, err := pc.httpClient.Post(apiURL, "application/json", bytes.NewBuffer(data))
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
	evictLeaderSchedulers, err := pc.GetEvictLeaderSchedulers()
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

func (pc *pdClient) EndEvictLeader(storeID uint64) error {
	sName := getLeaderEvictSchedulerStr(storeID)
	apiURL := fmt.Sprintf("%s/%s/%s", pc.url, schedulersPrefix, sName)
	req, err := http.NewRequest("DELETE", apiURL, nil)
	if err != nil {
		return err
	}
	res, err := pc.httpClient.Do(req)
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
	evictLeaderSchedulers, err := pc.GetEvictLeaderSchedulers()
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

func (pc *pdClient) GetEvictLeaderSchedulers() ([]string, error) {
	apiURL := fmt.Sprintf("%s/%s", pc.url, schedulersPrefix)
	body, err := httputil.GetBodyOK(pc.httpClient, apiURL)
	if err != nil {
		return nil, err
	}
	schedulers := []string{}
	err = json.Unmarshal(body, &schedulers)
	if err != nil {
		return nil, err
	}
	evicts := []string{}
	for _, scheduler := range schedulers {
		if strings.HasPrefix(scheduler, "evict-leader-scheduler") {
			evicts = append(evicts, scheduler)
		}
	}
	return evicts, nil
}

func (pc *pdClient) GetPDLeader() (*pdpb.Member, error) {
	apiURL := fmt.Sprintf("%s/%s", pc.url, pdLeaderPrefix)
	body, err := httputil.GetBodyOK(pc.httpClient, apiURL)
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

func (pc *pdClient) TransferPDLeader(memberName string) error {
	apiURL := fmt.Sprintf("%s/%s/%s", pc.url, pdLeaderTransferPrefix, memberName)
	req, err := http.NewRequest("POST", apiURL, nil)
	if err != nil {
		return err
	}
	res, err := pc.httpClient.Do(req)
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

func (pc *pdClient) getBodyOK(apiURL string) ([]byte, error) {
	res, err := pc.httpClient.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer httputil.DeferClose(res.Body)
	if res.StatusCode >= 400 {
		errMsg := fmt.Errorf(fmt.Sprintf("Error response %v URL %s", res.StatusCode, apiURL))
		return nil, errMsg
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return body, err
}

func getLeaderEvictSchedulerInfo(storeID uint64) *schedulerInfo {
	return &schedulerInfo{"evict-leader-scheduler", storeID}
}

func getLeaderEvictSchedulerStr(storeID uint64) string {
	return fmt.Sprintf("%s-%d", "evict-leader-scheduler", storeID)
}

type FakePDControl struct {
	defaultPDControl
}

func NewFakePDControl(kubeCli kubernetes.Interface) *FakePDControl {
	return &FakePDControl{
		defaultPDControl{kubeCli: kubeCli, pdClients: map[string]PDClient{}},
	}
}

func (fpc *FakePDControl) SetPDClient(namespace Namespace, tcName string, pdclient PDClient) {
	fpc.defaultPDControl.pdClients[pdClientKey("http", namespace, tcName)] = pdclient
}

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

type FakePDClient struct {
	reactions map[ActionType]Reaction
}

func NewFakePDClient() *FakePDClient {
	return &FakePDClient{reactions: map[ActionType]Reaction{}}
}

func (pc *FakePDClient) AddReaction(actionType ActionType, reaction Reaction) {
	pc.reactions[actionType] = reaction
}

// fakeAPI is a small helper for fake API calls
func (pc *FakePDClient) fakeAPI(actionType ActionType, action *Action) (interface{}, error) {
	if reaction, ok := pc.reactions[actionType]; ok {
		result, err := reaction(action)
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return nil, &NotFoundReaction{actionType}
}

func (pc *FakePDClient) GetHealth() (*HealthInfo, error) {
	action := &Action{}
	result, err := pc.fakeAPI(GetHealthActionType, action)
	if err != nil {
		return nil, err
	}
	return result.(*HealthInfo), nil
}

func (pc *FakePDClient) GetConfig() (*PDConfigFromAPI, error) {
	action := &Action{}
	result, err := pc.fakeAPI(GetConfigActionType, action)
	if err != nil {
		return nil, err
	}
	return result.(*PDConfigFromAPI), nil
}

func (pc *FakePDClient) GetCluster() (*metapb.Cluster, error) {
	action := &Action{}
	result, err := pc.fakeAPI(GetClusterActionType, action)
	if err != nil {
		return nil, err
	}
	return result.(*metapb.Cluster), nil
}

func (pc *FakePDClient) GetMembers() (*MembersInfo, error) {
	action := &Action{}
	result, err := pc.fakeAPI(GetMembersActionType, action)
	if err != nil {
		return nil, err
	}
	return result.(*MembersInfo), nil
}

func (pc *FakePDClient) GetStores() (*StoresInfo, error) {
	action := &Action{}
	result, err := pc.fakeAPI(GetStoresActionType, action)
	if err != nil {
		return nil, err
	}
	return result.(*StoresInfo), nil
}

func (pc *FakePDClient) GetTombStoneStores() (*StoresInfo, error) {
	action := &Action{}
	result, err := pc.fakeAPI(GetTombStoneStoresActionType, action)
	if err != nil {
		return nil, err
	}
	return result.(*StoresInfo), nil
}

func (pc *FakePDClient) GetStore(id uint64) (*StoreInfo, error) {
	action := &Action{
		ID: id,
	}
	result, err := pc.fakeAPI(GetStoreActionType, action)
	if err != nil {
		return nil, err
	}
	return result.(*StoreInfo), nil
}

func (pc *FakePDClient) DeleteStore(id uint64) error {
	if reaction, ok := pc.reactions[DeleteStoreActionType]; ok {
		action := &Action{ID: id}
		_, err := reaction(action)
		return err
	}
	return nil
}

func (pc *FakePDClient) SetStoreState(id uint64, state string) error {
	if reaction, ok := pc.reactions[SetStoreStateActionType]; ok {
		action := &Action{ID: id}
		_, err := reaction(action)
		return err
	}
	return nil
}

func (pc *FakePDClient) DeleteMemberByID(id uint64) error {
	if reaction, ok := pc.reactions[DeleteMemberByIDActionType]; ok {
		action := &Action{ID: id}
		_, err := reaction(action)
		return err
	}
	return nil
}

func (pc *FakePDClient) DeleteMember(name string) error {
	if reaction, ok := pc.reactions[DeleteMemberActionType]; ok {
		action := &Action{Name: name}
		_, err := reaction(action)
		return err
	}
	return nil
}

// SetStoreLabels sets TiKV labels
func (pc *FakePDClient) SetStoreLabels(storeID uint64, labels map[string]string) (bool, error) {
	if reaction, ok := pc.reactions[SetStoreLabelsActionType]; ok {
		action := &Action{ID: storeID, Labels: labels}
		result, err := reaction(action)
		return result.(bool), err
	}
	return true, nil
}

// UpdateReplicationConfig updates the replication config
func (pc *FakePDClient) UpdateReplicationConfig(config PDReplicationConfig) error {
	if reaction, ok := pc.reactions[UpdateReplicationActionType]; ok {
		action := &Action{Replication: config}
		_, err := reaction(action)
		return err
	}
	return nil
}

func (pc *FakePDClient) BeginEvictLeader(storeID uint64) error {
	if reaction, ok := pc.reactions[BeginEvictLeaderActionType]; ok {
		action := &Action{ID: storeID}
		_, err := reaction(action)
		return err
	}
	return nil
}

func (pc *FakePDClient) EndEvictLeader(storeID uint64) error {
	if reaction, ok := pc.reactions[EndEvictLeaderActionType]; ok {
		action := &Action{ID: storeID}
		_, err := reaction(action)
		return err
	}
	return nil
}

func (pc *FakePDClient) GetEvictLeaderSchedulers() ([]string, error) {
	if reaction, ok := pc.reactions[GetEvictLeaderSchedulersActionType]; ok {
		action := &Action{}
		result, err := reaction(action)
		return result.([]string), err
	}
	return nil, nil
}

func (pc *FakePDClient) GetPDLeader() (*pdpb.Member, error) {
	if reaction, ok := pc.reactions[GetPDLeaderActionType]; ok {
		action := &Action{}
		result, err := reaction(action)
		return result.(*pdpb.Member), err
	}
	return nil, nil
}

func (pc *FakePDClient) TransferPDLeader(memberName string) error {
	if reaction, ok := pc.reactions[TransferPDLeaderActionType]; ok {
		action := &Action{Name: memberName}
		_, err := reaction(action)
		return err
	}
	return nil
}
