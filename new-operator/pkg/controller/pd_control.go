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

package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/kvproto/pkg/pdpb"
	"github.com/pingcap/pd/pkg/typeutil"
	"github.com/pingcap/pd/server"
	"github.com/pingcap/tidb-operator/new-operator/pkg/apis/pingcap.com/v1"
	"github.com/pingcap/tidb-operator/pkg/util/errors"
)

const (
	timeout = 2 * time.Second
)

// PDControlInterface is an interface that knows how to manage and get tidb cluster's PD client
type PDControlInterface interface {
	// GetPDClient provide PDClient of the tidb cluster.
	GetPDClient(tc *v1.TidbCluster) PDClient
}

// defaultPDControl is the default implementation of PDControlInterface.
type defaultPDControl struct {
	mutex     sync.Mutex
	pdClients map[string]PDClient
}

// NewDefaultPDControl return a defaultPDControl instance
func NewDefaultPDControl() PDControlInterface {
	return &defaultPDControl{pdClients: map[string]PDClient{}}
}

// GetPDClient provide a PDClient of real pd cluster,if the PDClient not existing, it will create new one.
func (pdc *defaultPDControl) GetPDClient(tc *v1.TidbCluster) PDClient {
	pdc.mutex.Lock()
	defer pdc.mutex.Unlock()
	namespace := tc.GetNamespace()
	tcName := tc.GetName()
	key := pdClientKey(namespace, tcName)
	if _, ok := pdc.pdClients[key]; !ok {
		pdc.pdClients[key] = NewPDClient(pdClientURL(namespace, tcName), timeout)
	}
	return pdc.pdClients[key]
}

// pdClientKey return the pd client key
func pdClientKey(namespace, clusterName string) string {
	return fmt.Sprintf("%s.%s", clusterName, namespace)
}

// pdClientUrl build the url of pd client
func pdClientURL(namespace, clusterName string) string {
	return fmt.Sprintf("http://%s-pd.%s:2379", clusterName, namespace)
}

// PDClient provider pd server's api
type PDClient interface {
	// GetConfig returns PD's config
	GetConfig() (*server.Config, error)
	// GetCluster return used when syncing pod labels.
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
	// DeleteStore deletes a TiKV store from cluster
	DeleteStore(storeID uint64) error
	// DeleteMember delete a PD member from cluster
	DeleteMember(name string) error
	// DeleteMemberByID delete a PD member from cluster
	DeleteMemberByID(memberID uint64) error
	// BeginEvictLeader initiates leader eviction for a storeID.
	// This is used when upgrading a pod.
	BeginEvictLeader(storeID uint64) error
	// EndEvictLeader is used at the end of pod upgrade.
	EndEvictLeader(storeID uint64) error
}

var (
	membersPrefix    = "pd/api/v1/members"
	storesPrefix     = "pd/api/v1/stores"
	storePrefix      = "pd/api/v1/store"
	configPrefix     = "pd/api/v1/config"
	clusterIDPrefix  = "pd/api/v1/cluster"
	schedulersPrefix = "pd/api/v1/schedulers"
)

// pdClient is default implementation of PDClient
type pdClient struct {
	url        string
	httpClient *http.Client
}

// NewPDClient return a new PDClient
func NewPDClient(url string, timeout time.Duration) PDClient {
	return &pdClient{
		url:        url,
		httpClient: &http.Client{Timeout: timeout},
	}
}

// following struct definitions are copied from github.com/pingcap/pd/server/api/store
// these are not exported by that package

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

func (pc *pdClient) GetConfig() (*server.Config, error) {
	apiURL := fmt.Sprintf("%s/%s", pc.url, configPrefix)
	body, err := pc.getBodyOK(apiURL)
	if err != nil {
		return nil, errors.Trace(err)
	}
	config := &server.Config{}
	err = json.Unmarshal(body, config)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return config, nil
}

func (pc *pdClient) GetCluster() (*metapb.Cluster, error) {
	apiURL := fmt.Sprintf("%s/%s", pc.url, clusterIDPrefix)
	body, err := pc.getBodyOK(apiURL)
	if err != nil {
		return nil, errors.Trace(err)
	}
	cluster := &metapb.Cluster{}
	err = json.Unmarshal(body, cluster)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return cluster, nil
}

func (pc *pdClient) GetMembers() (*MembersInfo, error) {
	apiURL := fmt.Sprintf("%s/%s", pc.url, membersPrefix)
	body, err := pc.getBodyOK(apiURL)
	if err != nil {
		return nil, errors.Trace(err)
	}
	members := &MembersInfo{}
	err = json.Unmarshal(body, members)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return members, nil
}

func (pc *pdClient) GetStores() (*StoresInfo, error) {
	apiURL := fmt.Sprintf("%s/%s", pc.url, storesPrefix)
	body, err := pc.getBodyOK(apiURL)
	if err != nil {
		return nil, err
	}
	storesInfo := &StoresInfo{}
	err = json.Unmarshal(body, storesInfo)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return storesInfo, nil
}

func (pc *pdClient) GetTombStoneStores() (*StoresInfo, error) {
	apiURL := fmt.Sprintf("%s/%s?state=%d", pc.url, storesPrefix, metapb.StoreState_Tombstone)
	body, err := pc.getBodyOK(apiURL)
	if err != nil {
		return nil, errors.Trace(err)
	}
	storesInfo := &StoresInfo{}
	err = json.Unmarshal(body, storesInfo)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return storesInfo, nil
}

func (pc *pdClient) GetStore(storeID uint64) (*StoreInfo, error) {
	apiURL := fmt.Sprintf("%s/%s/%d", pc.url, storePrefix, storeID)
	body, err := pc.getBodyOK(apiURL)
	if err != nil {
		return nil, errors.Trace(err)
	}
	storeInfo := &StoreInfo{}
	err = json.Unmarshal(body, storeInfo)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return storeInfo, nil
}

func (pc *pdClient) DeleteStore(storeID uint64) error {
	apiURL := fmt.Sprintf("%s/%s/%d", pc.url, storePrefix, storeID)
	req, err := http.NewRequest("DELETE", apiURL, nil)
	if err != nil {
		return errors.Trace(err)
	}
	res, err := pc.httpClient.Do(req)
	if err != nil {
		return errors.Trace(err)
	}
	defer DeferClose(res.Body, &err)

	// Remove an offline store should return http.StatusOK
	if res.StatusCode == http.StatusOK {
		return nil
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return errors.Trace(err)
	}

	// FIXME: We should not rely on error text
	// Remove a tombstone store should return "store has been removed"
	bodyStr := string(body)
	if strings.Contains(bodyStr, "store has been removed") {
		return nil
	}

	err = errors.Errorf("failed to delete store %d: %v", storeID, string(body))
	return err
}

func (pc *pdClient) DeleteMemberByID(memberID uint64) error {
	apiURL := fmt.Sprintf("%s/%s/id/%d", pc.url, membersPrefix, memberID)
	req, err := http.NewRequest("DELETE", apiURL, nil)
	if err != nil {
		return errors.Trace(err)
	}
	res, err := pc.httpClient.Do(req)
	if err != nil {
		return errors.Trace(err)
	}
	defer DeferClose(res.Body, &err)
	if res.StatusCode == http.StatusOK {
		return nil
	}
	err2 := readErrorBody(res.Body)
	err = errors.Errorf("failed %v to delete member %d: %v", res.StatusCode, memberID, err2)
	return err
}

func (pc *pdClient) DeleteMember(name string) error {
	apiURL := fmt.Sprintf("%s/%s/name/%s", pc.url, membersPrefix, name)
	req, err := http.NewRequest("DELETE", apiURL, nil)
	if err != nil {
		return errors.Trace(err)
	}
	res, err := pc.httpClient.Do(req)
	if err != nil {
		return errors.Trace(err)
	}
	defer DeferClose(res.Body, &err)
	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusNotFound {
		return nil
	}
	err2 := readErrorBody(res.Body)
	err = errors.Errorf("failed %v to delete member %s: %v", res.StatusCode, name, err2)
	return err
}

func (pc *pdClient) SetStoreLabels(storeID uint64, labels map[string]string) (bool, error) {
	apiURL := fmt.Sprintf("%s/%s/%d/label", pc.url, storePrefix, storeID)
	data, err := json.Marshal(labels)
	if err != nil {
		return false, err
	}
	res, err := http.Post(apiURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return false, err
	}
	defer DeferClose(res.Body, &err)
	if res.StatusCode == http.StatusOK {
		return true, nil
	}
	err2 := readErrorBody(res.Body)
	err = errors.Errorf("failed %v to set store labels: %v", res.StatusCode, err2)
	return false, err
}

func (pc *pdClient) BeginEvictLeader(storeID uint64) error {
	leaderEvictInfo := getLeaderEvictSchedulerInfo(storeID)
	apiURL := fmt.Sprintf("%s/%s", pc.url, schedulersPrefix)
	data, err := json.Marshal(leaderEvictInfo)
	if err != nil {
		return errors.Trace(err)
	}
	res, err := http.Post(apiURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return errors.Trace(err)
	}
	defer DeferClose(res.Body, &err)
	if res.StatusCode == http.StatusOK {
		return nil
	}
	err2 := readErrorBody(res.Body)
	err = errors.Errorf("failed %v to begin evict leader of store:[%d],error: %v", res.StatusCode, storeID, err2)
	return err
}

func (pc *pdClient) EndEvictLeader(storeID uint64) error {
	apiURL := fmt.Sprintf("%s/%s/%s", pc.url, schedulersPrefix, getLeaderEvictSchedulerStr(storeID))
	req, err := http.NewRequest("DELETE", apiURL, nil)
	if err != nil {
		return errors.Trace(err)
	}
	res, err := pc.httpClient.Do(req)
	if err != nil {
		return errors.Trace(err)
	}
	defer DeferClose(res.Body, &err)
	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusNotFound {
		return nil
	}
	err2 := readErrorBody(res.Body)
	err = errors.Errorf("failed %v to end leader evict scheduler of store [%d],error:%v", res.StatusCode, storeID, err2)
	return err
}

func (pc *pdClient) getBodyOK(apiURL string) ([]byte, error) {
	res, err := pc.httpClient.Get(apiURL)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if res.StatusCode >= 400 {
		errMsg := fmt.Errorf(fmt.Sprintf("Error response %v", res.StatusCode))
		return nil, errors.Trace(errMsg)
	}

	defer DeferClose(res.Body, &err)
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return body, err
}

func getLeaderEvictSchedulerInfo(storeID uint64) *schedulerInfo {
	return &schedulerInfo{"evict-leader-scheduler", storeID}
}

func getLeaderEvictSchedulerStr(storeID uint64) string {
	return fmt.Sprintf("%s-%d", "evict-leader-scheduler", storeID)
}

// DeferClose captures the error returned from closing (if an error occurs).
// This is designed to be used in a defer statement.
func DeferClose(c io.Closer, err *error) {
	if cerr := c.Close(); cerr != nil && *err == nil {
		*err = errors.Trace(cerr, 2)
	}
}

// In the error case ready the body message.
// But return it as an error (or return an error from reading the body).
func readErrorBody(body io.Reader) (err error) {
	bodyBytes, err := ioutil.ReadAll(body)
	if err != nil {
		err = errors.Trace(err, 2)
	} else {
		err = errors.Errorf(string(bodyBytes))
	}
	return err
}
