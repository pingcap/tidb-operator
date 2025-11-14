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

//go:generate ${GOBIN}/mockgen -write_command_comment=false -copyright_file ${BOILERPLATE_FILE} -destination mock_generated.go -package=pdapi ${GO_MODULE}/pkg/pdapi/v1 PDClient
package pdapi

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/kvproto/pkg/pdpb"

	"github.com/pingcap/tidb-operator/v2/pkg/compatibility"
	httputil "github.com/pingcap/tidb-operator/v2/pkg/utils/http"
)

const (
	DefaultTimeout       = 5 * time.Second
	evictSchedulerLeader = "evict-leader-scheduler"
	tiKVNotBootstrapped  = `TiKV cluster not bootstrapped, please start TiKV first"`
)

// Namespace is a newtype of a string
type Namespace string

// PDWriter defines write api call of pd
// TODO: move all Get api call to PDClient
type PDWriter interface {
	// SetStoreLabels sets the labels for a store.
	SetStoreLabels(ctx context.Context, storeID uint64, labels map[string]string) (bool, error)
	// DeleteStore deletes a TiKV/TiFlash store from the cluster.
	DeleteStore(ctx context.Context, storeID string) error
	// CancelDeleteStore cancels the deletion of a TiKV/TiFlash store, returning it to online state.
	// If the store is already online, it will return nil.
	CancelDeleteStore(ctx context.Context, storeID string) error
	// DeleteMember deletes a PD member from the cluster.
	DeleteMember(ctx context.Context, name string) error

	// BeginEvictLeader initiates leader eviction for a store.
	BeginEvictLeader(ctx context.Context, storeID string) error
	// EndEvictLeader removes the leader eviction scheduler for a store.
	EndEvictLeader(ctx context.Context, storeID string) error

	// TransferPDLeader transfers PD leader to specified member.
	TransferPDLeader(ctx context.Context, name string) error
}

// PDClient provides PD server's APIs used by TiDB Operator.
type PDClient interface {
	// GetMemberReady returns if a PD member is ready to serve.
	// In order to call this method, the PD member's URL is required.
	GetMemberReady(ctx context.Context, url, version string) (bool, error)
	// GetHealth returns the health of PD's members.
	GetHealth(ctx context.Context) (*HealthInfo, error)
	// GetConfig returns PD's config.
	GetConfig(ctx context.Context) (*PDConfigFromAPI, error)
	// GetCluster returns the cluster information.
	GetCluster(ctx context.Context) (*metapb.Cluster, error)
	// GetMembers returns all PD members of the cluster.
	GetMembers(ctx context.Context) (*MembersInfo, error)
	// GetStores lists all TiKV/TiFlash stores of the cluster.
	GetStores(ctx context.Context) (*StoresInfo, error)
	// GetTombStoneStores lists all tombstone stores of the cluster.
	// GetTombStoneStores() (*StoresInfo, error)
	// GetStore gets a TiKV/TiFlash store for a specific store id of the cluster.
	GetStore(ctx context.Context, storeID string) (*StoreInfo, error)
	// GetEvictLeaderScheduler gets leader eviction schedulers for stores.
	GetEvictLeaderScheduler(ctx context.Context, storeID string) (string, error)

	GetPDEtcdClient() (PDEtcdClient, error)

	GetTSOLeader(ctx context.Context) (string, error)

	// GetTSOMembers returns all PD members service-addr from cluster by specific Micro Service.
	GetTSOMembers(ctx context.Context) ([]ServiceRegistryEntry, error)

	PDWriter

	// NOTE: not used
	//
	// GetEvictLeaderSchedulers gets schedulers of leader eviction.
	// GetEvictLeaderSchedulers(ctx context.Context) ([]string, error)
	//
	// GetPDLeader returns the PD leader.
	// GetPDLeader(ctx context.Context) (*pdpb.Member, error)
	//
	// UpdateReplicationConfig updates the replication config.
	// UpdateReplicationConfig(ctx context.Context, config PDReplicationConfig) error
	//
	// DeleteMemberByID deletes a PD member from the cluster
	// DeleteMemberByID(ctx context.Context, memberID uint64) error
}

const (
	healthPrefix    = "pd/api/v1/health"
	configPrefix    = "pd/api/v1/config"
	clusterIDPrefix = "pd/api/v1/cluster"

	membersPrefix      = "pd/api/v1/members"
	microServicePrefix = "pd/api/v2/ms"

	storesPrefix       = "pd/api/v1/stores"
	storePrefix        = "pd/api/v1/store"
	storeUpStatePrefix = "pd/api/v1/store/%v/state?state=Up"

	pdReplicationPrefix = "pd/api/v1/config/replicate"

	schedulersPrefix                 = "pd/api/v1/schedulers"
	pdLeaderPrefix                   = "pd/api/v1/leader"
	pdLeaderTransferPrefix           = "pd/api/v1/leader/transfer"
	evictLeaderSchedulerConfigPrefix = "pd/api/v1/scheduler-config/evict-leader-scheduler/list"

	pdReadyPrefix = "pd/api/v2/ready"

	// Micro Service
	// leader endpoint
	tsoLeaderPrefix = microServicePrefix + "/primary/tso"
)

// pdClient is the default implementation of PDClient.
type pdClient struct {
	url        string
	httpClient *http.Client
	tlsConfig  *tls.Config

	etcdmutex     sync.Mutex
	pdEtcdClients map[string]PDEtcdClient
}

// NewPDClient returns a new PDClient
func NewPDClient(url string, timeout time.Duration, tlsConfig *tls.Config) PDClient {
	return &pdClient{
		url: url,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: &http.Transport{TLSClientConfig: tlsConfig},
		},
		tlsConfig:     tlsConfig,
		pdEtcdClients: make(map[string]PDEtcdClient),
	}
}

func (c *pdClient) GetPDEtcdClient() (PDEtcdClient, error) {
	c.etcdmutex.Lock()
	defer c.etcdmutex.Unlock()

	if _, ok := c.pdEtcdClients[c.url]; !ok {
		etcdCli, err := NewPdEtcdClient(c.url, DefaultTimeout, c.tlsConfig)
		if err != nil {
			return nil, err
		}
		c.pdEtcdClients[c.url] = &noOpClose{PDEtcdClient: etcdCli}
	}
	return c.pdEtcdClients[c.url], nil
}

func (c *pdClient) GetHealth(ctx context.Context) (*HealthInfo, error) {
	apiURL := fmt.Sprintf("%s/%s", c.url, healthPrefix)
	body, err := httputil.GetBodyOK(ctx, c.httpClient, apiURL)
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

func (c *pdClient) GetConfig(ctx context.Context) (*PDConfigFromAPI, error) {
	apiURL := fmt.Sprintf("%s/%s", c.url, configPrefix)
	body, err := httputil.GetBodyOK(ctx, c.httpClient, apiURL)
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

func (c *pdClient) GetCluster(ctx context.Context) (*metapb.Cluster, error) {
	apiURL := fmt.Sprintf("%s/%s", c.url, clusterIDPrefix)
	body, err := httputil.GetBodyOK(ctx, c.httpClient, apiURL)
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

func (c *pdClient) GetMembers(ctx context.Context) (*MembersInfo, error) {
	apiURL := fmt.Sprintf("%s/%s", c.url, membersPrefix)
	body, err := httputil.GetBodyOK(ctx, c.httpClient, apiURL)
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

func (c *pdClient) getMSMembers(ctx context.Context, service string) ([]ServiceRegistryEntry, error) {
	apiURL := fmt.Sprintf("%s/%s/members/%s", c.url, microServicePrefix, service)
	body, err := httputil.GetBodyOK(ctx, c.httpClient, apiURL)
	if err != nil {
		return nil, err
	}
	var members []ServiceRegistryEntry
	if err := json.Unmarshal(body, &members); err != nil {
		return nil, err
	}

	return members, nil
}

func (c *pdClient) GetTSOMembers(ctx context.Context) ([]ServiceRegistryEntry, error) {
	return c.getMSMembers(ctx, "tso")
}

func (c *pdClient) GetStores(ctx context.Context) (*StoresInfo, error) {
	storesInfo, err := c.getStores(
		ctx,
		fmt.Sprintf("%s/%s", c.url, storesPrefix),
		metapb.StoreState_Up,
		metapb.StoreState_Offline,
		metapb.StoreState_Tombstone,
	)
	if err != nil {
		if strings.HasSuffix(err.Error(), tiKVNotBootstrapped+"\n") {
			return nil, fmt.Errorf("%w: %w", ErrTiKVNotBootstrapped, err)
		}
		return nil, err
	}
	return storesInfo, nil
}

func (c *pdClient) GetTombStoneStores(ctx context.Context) (*StoresInfo, error) {
	return c.getStores(ctx, fmt.Sprintf("%s/%s", c.url, storesPrefix), metapb.StoreState_Tombstone)
}

func (c *pdClient) GetStore(ctx context.Context, storeID string) (*StoreInfo, error) {
	apiURL := fmt.Sprintf("%s/%s/%s", c.url, storePrefix, storeID)
	body, err := httputil.GetBodyOK(ctx, c.httpClient, apiURL)
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

func (c *pdClient) getStores(ctx context.Context, apiURL string, states ...metapb.StoreState) (*StoresInfo, error) {
	if len(states) != 0 {
		var q []string
		for _, state := range states {
			q = append(q, "state="+strconv.Itoa(int(state)))
		}
		apiURL += "?" + strings.Join(q, "&")
	}
	body, err := httputil.GetBodyOK(ctx, c.httpClient, apiURL)
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

func (c *pdClient) SetStoreLabels(ctx context.Context, storeID uint64, labels map[string]string) (bool, error) {
	apiURL := fmt.Sprintf("%s/%s/%d/label", c.url, storePrefix, storeID)
	data, err := json.Marshal(labels)
	if err != nil {
		return false, err
	}
	if _, err := httputil.PostBodyOK(ctx, c.httpClient, apiURL, bytes.NewBuffer(data)); err != nil {
		return false, fmt.Errorf("failed to set store labels: %w", err)
	}
	return true, nil
}

func (c *pdClient) DeleteStore(ctx context.Context, storeID string) error {
	apiURL := fmt.Sprintf("%s/%s/%s", c.url, storePrefix, storeID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", apiURL, http.NoBody)
	if err != nil {
		return err
	}

	//nolint:bodyclose // has been handled
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

	return fmt.Errorf("failed to delete store %s: %v", storeID, string(body))
}

// CancelDeleteStore cancels the deletion of a TiKV/TiFlash store, returning it to online state.
// Refer: https://github.com/tikv/pd/blob/7a5b221cf66ec469727f8c174493f9132c0b9d8f/tools/pd-ctl/pdctl/command/store_command.go#L388
func (c *pdClient) CancelDeleteStore(ctx context.Context, storeID string) error {
	apiURL := fmt.Sprintf("%s/%s", c.url, fmt.Sprintf(storeUpStatePrefix, storeID))
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, http.NoBody)
	if err != nil {
		return err
	}

	//nolint:bodyclose // has been handled
	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer httputil.DeferClose(res.Body)

	// Cancel store deletion should return http.StatusOK
	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusNotFound {
		return nil
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	return fmt.Errorf("failed to cancel store deletion for %s: %v", storeID, string(body))
}

func (c *pdClient) DeleteMember(ctx context.Context, name string) error {
	var exist bool
	members, err := c.GetMembers(ctx)
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
	req, err := http.NewRequest("DELETE", apiURL, http.NoBody)
	if err != nil {
		return err
	}
	//nolint:bodyclose // has been handled
	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer httputil.DeferClose(res.Body)
	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusNotFound {
		return nil
	}
	err2 := httputil.ReadErrorBody(res.Body)
	return fmt.Errorf("failed %v to delete member %s: %w", res.StatusCode, name, err2)
}

func (c *pdClient) DeleteMemberByID(ctx context.Context, memberID uint64) error {
	var exist bool
	members, err := c.GetMembers(ctx)
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
	req, err := http.NewRequestWithContext(ctx, "DELETE", apiURL, http.NoBody)
	if err != nil {
		return err
	}
	//nolint:bodyclose // has been handled
	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer httputil.DeferClose(res.Body)
	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusNotFound {
		return nil
	}
	err2 := httputil.ReadErrorBody(res.Body)
	return fmt.Errorf("failed %v to delete member %d: %w", res.StatusCode, memberID, err2)
}

func (c *pdClient) UpdateReplicationConfig(ctx context.Context, config PDReplicationConfig) error {
	apiURL := fmt.Sprintf("%s/%s", c.url, pdReplicationPrefix)
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	if _, err := httputil.PostBodyOK(ctx, c.httpClient, apiURL, bytes.NewBuffer(data)); err != nil {
		return fmt.Errorf("failed to update replication: %w", err)
	}
	return nil
}

func (c *pdClient) BeginEvictLeader(ctx context.Context, storeID string) error {
	leaderEvictInfo, err := getLeaderEvictSchedulerInfo(storeID)
	if err != nil {
		return err
	}
	apiURL := fmt.Sprintf("%s/%s", c.url, schedulersPrefix)
	data, err := json.Marshal(leaderEvictInfo)
	if err != nil {
		return err
	}
	if _, err = httputil.PostBodyOK(ctx, c.httpClient, apiURL, bytes.NewBuffer(data)); err == nil {
		return nil
	}

	// pd will return an error with the body contains "scheduler existed" if the scheduler already exists
	// this is not the standard response.
	// so these lines are just a workaround for now:
	//   - make a new request to get all schedulers
	//   - return nil if the scheduler already exists
	//
	// when PD returns standard json response, we should get rid of this verbose code.
	evictLeaderSchedulers, err2 := c.GetEvictLeaderSchedulers(ctx)
	if err2 != nil {
		return err2
	}

	storeIDStr := getLeaderEvictSchedulerStr(storeID)

	if slices.Contains(evictLeaderSchedulers, storeIDStr) {
		return nil
	}

	return fmt.Errorf("failed to begin evict leader of store:[%s], error: %w", storeID, err)
}

func (c *pdClient) EndEvictLeader(ctx context.Context, storeID string) error {
	sName := getLeaderEvictSchedulerStr(storeID)
	apiURL := fmt.Sprintf("%s/%s/%s", c.url, schedulersPrefix, sName)
	req, err := http.NewRequestWithContext(ctx, "DELETE", apiURL, http.NoBody)
	if err != nil {
		return err
	}
	//nolint:bodyclose // has been handled
	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer httputil.DeferClose(res.Body)
	if res.StatusCode == http.StatusNotFound {
		return nil
	} else if res.StatusCode != http.StatusOK {
		err2 := httputil.ReadErrorBody(res.Body)
		return fmt.Errorf("failed %v to end leader evict scheduler of store:[%s], error: %w", res.StatusCode, storeID, err2)
	}

	// pd will return an error with the body contains "scheduler not found" if the scheduler is not found
	// this is not the standard response.
	// so these lines are just a workaround for now:
	//   - make a new request to get all schedulers
	//   - return nil if the scheduler is not found
	//
	// when PD returns standard json response, we should get rid of this verbose code.
	evictLeaderSchedulers, err := c.GetEvictLeaderSchedulers(ctx)
	if err != nil {
		return err
	}
	if slices.Contains(evictLeaderSchedulers, sName) {
		return fmt.Errorf("end leader evict scheduler failed, the store:[%s]'s leader evict scheduler is still exist", storeID)
	}

	return nil
}

func (c *pdClient) GetEvictLeaderSchedulers(ctx context.Context) ([]string, error) {
	apiURL := fmt.Sprintf("%s/%s", c.url, schedulersPrefix)
	body, err := httputil.GetBodyOK(ctx, c.httpClient, apiURL)
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
	evictSchedulers, err := c.filterLeaderEvictScheduler(ctx, evicts)
	if err != nil {
		return nil, err
	}
	return evictSchedulers, nil
}

func (c *pdClient) GetEvictLeaderScheduler(ctx context.Context, storeID string) (string, error) {
	schedulers, err := c.GetEvictLeaderSchedulers(ctx)
	if err != nil {
		return "", err
	}

	for _, scheduler := range schedulers {
		sName := getLeaderEvictSchedulerStr(storeID)
		if scheduler == sName {
			return scheduler, nil
		}
	}

	return "", nil
}

// This method is to make compatible between old pdapi version and versions after 3.1/4.0.
// To get more detail, see:
// - https://github.com/pingcap/tidb-operator/pull/1831
// - https://github.com/pingcap/pd/issues/2550
func (c *pdClient) filterLeaderEvictScheduler(ctx context.Context, evictLeaderSchedulers []string) ([]string, error) {
	if len(evictLeaderSchedulers) == 1 && evictLeaderSchedulers[0] == evictSchedulerLeader {
		var schedulerIDs []string
		// If there is only one evcit scehduler entry without store ID postfix.
		// We should get the store IDs via scheduler config API and append them
		// to provide consistent results.
		c, err := c.getEvictLeaderSchedulerConfig(ctx)
		if err != nil {
			return nil, err
		}
		for k := range c.StoreIDWithRanges {
			schedulerIDs = append(schedulerIDs, fmt.Sprintf("%s-%v", evictSchedulerLeader, k))
		}

		return schedulerIDs, nil
	}

	return evictLeaderSchedulers, nil
}

// getEvictLeaderSchedulerConfig gets the config of PD scheduler "evict-leader-scheduler"
// It's available since PD 3.1.0.
// In the previous versions, PD API returns 404 and this function will return an error.
func (c *pdClient) getEvictLeaderSchedulerConfig(ctx context.Context) (*EvictLeaderSchedulerConfig, error) {
	apiURL := fmt.Sprintf("%s/%s", c.url, evictLeaderSchedulerConfigPrefix)
	body, err := httputil.GetBodyOK(ctx, c.httpClient, apiURL)
	if err != nil {
		return nil, err
	}
	config := &EvictLeaderSchedulerConfig{}
	err = json.Unmarshal(body, config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func getLeaderEvictSchedulerInfo(storeID string) (*SchedulerInfo, error) {
	id, err := strconv.ParseUint(storeID, 10, 64)
	if err != nil {
		return nil, err
	}
	return &SchedulerInfo{"evict-leader-scheduler", id}, nil
}

func getLeaderEvictSchedulerStr(storeID string) string {
	return fmt.Sprintf("%s-%s", "evict-leader-scheduler", storeID)
}

func (c *pdClient) GetPDLeader(ctx context.Context) (*pdpb.Member, error) {
	apiURL := fmt.Sprintf("%s/%s", c.url, pdLeaderPrefix)
	body, err := httputil.GetBodyOK(ctx, c.httpClient, apiURL)
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

func (c *pdClient) TransferPDLeader(ctx context.Context, memberName string) error {
	apiURL := fmt.Sprintf("%s/%s/%s", c.url, pdLeaderTransferPrefix, memberName)
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, http.NoBody)
	if err != nil {
		return err
	}
	//nolint:bodyclose // has been handled
	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer httputil.DeferClose(res.Body)
	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusNotFound {
		return nil
	}
	err2 := httputil.ReadErrorBody(res.Body)
	return fmt.Errorf("failed %v to transfer pd leader to %s, error: %w", res.StatusCode, memberName, err2)
}

func (c *pdClient) GetTSOLeader(ctx context.Context) (string, error) {
	apiURL := fmt.Sprintf("%s/%s", c.url, tsoLeaderPrefix)
	body, err := httputil.GetBodyOK(ctx, c.httpClient, apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to get tso leader, error: %w", err)
	}
	var primary string
	if err := json.Unmarshal(body, &primary); err != nil {
		return "", fmt.Errorf("failed to get tso leader, cannot unmarshal response: %w", err)
	}

	return primary, nil
}

func (c *pdClient) GetMemberReady(ctx context.Context, url, version string) (bool, error) {
	apiURL := fmt.Sprintf("%s/%s", url, pdReadyPrefix)
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, http.NoBody)
	if err != nil {
		return false, fmt.Errorf("failed to new a request: %w", err)
	}
	//nolint:bodyclose // has been handled
	res, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to send a http request: %w", err)
	}
	defer httputil.DeferClose(res.Body)

	switch res.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		v, err := semver.NewVersion(version)
		if err != nil {
			return false, fmt.Errorf("failed to parse version %s: %w", version, err)
		}
		if !compatibility.Check(v, compatibility.PDReadyAPI) {
			// If the version is lower than v8.5.2, we assume PD is ready
			return true, nil
		}
		return false, nil
	case http.StatusInternalServerError:
		// If the status code is 500, it means regions are not loaded yet,
		// according to https://github.com/tikv/pd/pull/8749.
		return false, nil
	default:
		return false, fmt.Errorf("failed to get ready status: %w, status code: %d", httputil.ReadErrorBody(res.Body), res.StatusCode)
	}
}
