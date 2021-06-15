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

package binlog

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/pingcap/errors"
)

// Client is the client of binlog.
type Client struct {
	tls        *tls.Config
	httpClient *http.Client
	etcdClient *clientv3.Client

	// if setted, will call HookAddr to change the address of pump/drainer
	// before accessing pump/drainer.
	HookAddr func(addr string) (changedAddr string)
}

func (c *Client) hookAddr(addr string) string {
	if c.HookAddr != nil {
		return c.HookAddr(addr)
	}

	return addr
}

// NewBinlogClient create a Client.
func NewBinlogClient(pdEndpoint []string, tlsConfig *tls.Config) (*Client, error) {
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   pdEndpoint,
		DialTimeout: time.Second * 5,
		TLS:         tlsConfig,
	})
	if err != nil {
		return nil, errors.AddStack(err)
	}

	return &Client{
		tls: tlsConfig,
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig:   tlsConfig,
				DisableKeepAlives: true,
			},
		},
		etcdClient: etcdClient,
	}, nil
}

// Close the client.
func (c *Client) Close() error {
	c.httpClient.CloseIdleConnections()
	return c.etcdClient.Close()
}

func (c *Client) getURL(addr string) string {
	scheme := "http"
	if c.tls != nil {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s", scheme, addr)
}

func (c *Client) getOfflineURL(addr string, nodeID string) string {
	return fmt.Sprintf("%s/state/%s/close", c.getURL(addr), nodeID)
}

// StatusResp represents the response of status api.
type StatusResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NodeStatus represents the status saved in etcd.
type NodeStatus struct {
	NodeID string `json:"nodeId"`
	Host   string `json:"host"`
	State  string `json:"state"`

	// NB: Currently we save the whole `NodeStatus` in the status of the CR.
	// However, the following fields will be updated continuously.
	// To avoid CR being updated and re-synced continuously, we exclude these fields.
	// MaxCommitTS int64  `json:"maxCommitTS"`
	// UpdateTS    int64  `json:"updateTS"`
}

// IsPumpTombstone check if drainer is tombstone.
func (c *Client) IsPumpTombstone(ctx context.Context, addr string) (bool, error) {
	nodeID, err := c.nodeID(ctx, addr, "pumps")
	if err != nil {
		return false, err
	}
	return c.isTombstone(ctx, "pumps", nodeID)
}

// IsDrainerTombstone check if drainer is tombstone.
func (c *Client) IsDrainerTombstone(ctx context.Context, addr string) (bool, error) {
	nodeID, err := c.nodeID(ctx, addr, "drainers")
	if err != nil {
		return false, err
	}
	return c.isTombstone(ctx, "drainers", nodeID)
}

func (c *Client) isTombstone(ctx context.Context, ty string, nodeID string) (bool, error) {
	status, err := c.nodeStatus(ctx, ty)
	if err != nil {
		return false, err
	}

	for _, s := range status {
		if s.NodeID == nodeID {
			if s.State == "offline" {
				return true, nil
			}
			return false, nil
		}
	}

	return false, errors.Errorf("node not exist: %s", nodeID)
}

func (c *Client) PumpNodeStatus(ctx context.Context) (status []*NodeStatus, err error) {
	return c.nodeStatus(ctx, "pumps")
}

// nolint (unused)
func (c *Client) drainerNodeStatus(ctx context.Context) (status []*NodeStatus, err error) {
	return c.nodeStatus(ctx, "drainers")
}

func (c *Client) nodeID(ctx context.Context, addr, ty string) (string, error) {
	nodes, err := c.nodeStatus(ctx, ty)
	if err != nil {
		return "", err
	}

	addrs := []string{}
	for _, node := range nodes {
		if addr == node.Host {
			return node.NodeID, nil
		}
		addrs = append(addrs, node.Host)
	}

	return "", errors.Errorf("%s node id for address %s not found, found address: %s", ty, addr, addrs)
}

// UpdateDrainerState update the specify state as the specified state.
func (c *Client) UpdateDrainerState(ctx context.Context, addr string, state string) error {
	nodeID, err := c.nodeID(ctx, addr, "drainers")
	if err != nil {
		return err
	}
	return c.updateStatus(ctx, "drainers", nodeID, state)
}

// UpdatePumpState update the specify state as the specified state.
func (c *Client) UpdatePumpState(ctx context.Context, addr string, state string) error {
	nodeID, err := c.nodeID(ctx, addr, "pumps")
	if err != nil {
		return err
	}
	return c.updateStatus(ctx, "pumps", nodeID, state)
}

// updateStatus update the specify state as the specified state.
func (c *Client) updateStatus(ctx context.Context, ty string, nodeID string, state string) error {
	key := fmt.Sprintf("/tidb-binlog/v1/%s/%s", ty, nodeID)

	resp, err := c.etcdClient.KV.Get(ctx, key)
	if err != nil {
		return errors.AddStack(err)
	}

	if len(resp.Kvs) == 0 {
		return errors.Errorf("no %s with node id: %v", ty, nodeID)
	}

	var nodeStatus NodeStatus
	err = json.Unmarshal(resp.Kvs[0].Value, &nodeStatus)
	if err != nil {
		return errors.AddStack(err)
	}

	if nodeStatus.State == state {
		return nil
	}

	nodeStatus.State = state

	data, err := json.Marshal(&nodeStatus)
	if err != nil {
		return errors.AddStack(err)
	}

	_, err = c.etcdClient.Put(ctx, key, string(data))
	if err != nil {
		return errors.AddStack(err)
	}

	return nil
}

func (c *Client) nodeStatus(ctx context.Context, ty string) (status []*NodeStatus, err error) {
	key := fmt.Sprintf("/tidb-binlog/v1/%s", ty)

	resp, err := c.etcdClient.KV.Get(ctx, key, clientv3.WithPrefix())
	if err != nil {
		return nil, errors.AddStack(err)
	}

	for _, kv := range resp.Kvs {
		var s NodeStatus
		err = json.Unmarshal(kv.Value, &s)
		if err != nil {
			return nil, errors.Annotatef(err, "key: %s, data: %s", string(kv.Key), string(kv.Value))
		}

		status = append(status, &s)
	}

	return
}

func (c *Client) offline(addr string, nodeID string) error {
	url := c.getOfflineURL(c.hookAddr(addr), nodeID)
	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return errors.AddStack(err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.AddStack(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return errors.Errorf("error requesting %s, code: %d",
			resp.Request.URL, resp.StatusCode)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.AddStack(err)
	}

	var status StatusResp
	err = json.Unmarshal(data, &status)
	if err != nil {
		return errors.Annotatef(err, "data: %s", string(data))
	}

	if status.Code != 200 {
		return errors.Errorf("server error: %s", status.Message)
	}

	return nil
}

// OfflinePump offline a pump.
func (c *Client) OfflinePump(ctx context.Context, addr string) error {
	nodeID, err := c.nodeID(ctx, addr, "pumps")
	if err != nil {
		return err
	}
	return c.offline(addr, nodeID)
}

// OfflineDrainer offline a drainer.
func (c *Client) OfflineDrainer(ctx context.Context, addr string) error {
	nodeID, err := c.nodeID(ctx, addr, "drainers")
	if err != nil {
		return err
	}
	return c.offline(addr, nodeID)
}
