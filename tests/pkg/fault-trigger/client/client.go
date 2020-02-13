// Copyright 2019 PingCAP, Inc.
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

package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pingcap/tidb-operator/tests/pkg/fault-trigger/api"
	"github.com/pingcap/tidb-operator/tests/pkg/fault-trigger/manager"
	"github.com/pingcap/tidb-operator/tests/pkg/util"
	"k8s.io/klog"
)

// Client is a fault-trigger client
type Client interface {
	// ListVMs lists all virtual machines
	ListVMs() ([]*manager.VM, error)
	// StartVM start a specified virtual machine
	StartVM(vm *manager.VM) error
	// StopVM stops a specified virtual machine
	StopVM(vm *manager.VM) error
	// StartETCD starts the etcd service
	StartETCD() error
	// StopETCD stops the etcd service
	StopETCD() error
	// StartKubelet starts the kubelet service
	StartKubelet() error
	// StopKubelet stops the kubelet service
	StopKubelet() error
	// StartKubeAPIServer starts the apiserver service
	StartKubeAPIServer() error
	// StopKubeAPIServer stops the apiserver service
	StopKubeAPIServer() error
	// StartKubeScheduler starts the kube-scheduler service
	StartKubeScheduler() error
	// StopKubeScheduler stops the kube-scheduler service
	StopKubeScheduler() error
	// StartKubeControllerManager starts the kube-controller-manager service
	StartKubeControllerManager() error
	// StopKubeControllerManager stops the kube-controller-manager service
	StopKubeControllerManager() error
}

// client is used to communicate with the fault-trigger
type client struct {
	cfg     Config
	httpCli *http.Client
}

// Config defines for fault-trigger client
type Config struct {
	Addr string
}

// NewClient creates a new fault-trigger client from a given address
func NewClient(cfg Config) Client {
	return &client{
		cfg:     cfg,
		httpCli: http.DefaultClient,
	}
}

type clientError struct {
	code int
	msg  string
}

func (e *clientError) Error() string {
	return fmt.Sprintf("%s (code: %d)", e.msg, e.code)
}

func (c client) do(req *http.Request) (*http.Response, []byte, error) {
	resp, err := c.httpCli.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	code := resp.StatusCode

	if code != http.StatusOK {
		return resp, nil, &clientError{
			code: code,
			msg:  "fail to request to http service",
		}
	}

	bodyByte, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, &clientError{
			code: code,
			msg:  fmt.Sprintf("failed to read data from resp body, error: %v", err),
		}
	}

	data, err := api.ExtractResponse(bodyByte)
	if err != nil {
		return resp, nil, &clientError{
			code: code,
			msg:  err.Error(),
		}
	}

	return resp, data, err
}

func (c client) get(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	_, body, err := c.do(req)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (c *client) post(url string, data []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	_, body, err := c.do(req)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (c *client) ListVMs() ([]*manager.VM, error) {
	url := util.GenURL(fmt.Sprintf("%s%s/vms", c.cfg.Addr, api.APIPrefix))
	data, err := c.get(url)
	if err != nil {
		klog.Errorf("failed to get %s: %v", url, err)
		return nil, err
	}

	var vms []*manager.VM
	if err = json.Unmarshal(data, &vms); err != nil {
		return nil, err
	}

	return vms, nil
}

func (c *client) StartVM(vm *manager.VM) error {
	if err := vm.Verify(); err != nil {
		return err
	}

	vmName := vm.Name

	url := util.GenURL(fmt.Sprintf("%s%s/vm/%s/start", c.cfg.Addr, api.APIPrefix, vmName))
	if _, err := c.post(url, nil); err != nil {
		klog.Errorf("faled to post %s: %v", url, err)
		return err
	}

	return nil
}

func (c *client) StopVM(vm *manager.VM) error {
	if err := vm.Verify(); err != nil {
		return err
	}

	vmName := vm.Name

	url := util.GenURL(fmt.Sprintf("%s%s/vm/%s/stop", c.cfg.Addr, api.APIPrefix, vmName))
	if _, err := c.post(url, nil); err != nil {
		klog.Errorf("faled to post %s: %v", url, err)
		return err
	}

	return nil
}

func (c *client) StartETCD() error {
	return c.startService(manager.ETCDService)
}

func (c *client) StopETCD() error {
	return c.stopService(manager.ETCDService)
}

func (c *client) StartKubelet() error {
	return c.startService(manager.KubeletService)
}

func (c *client) StopKubelet() error {
	return c.stopService(manager.KubeletService)
}

func (c *client) StartKubeAPIServer() error {
	return c.startService(manager.KubeAPIServerService)
}

func (c *client) StopKubeAPIServer() error {
	return c.stopService(manager.KubeAPIServerService)
}

func (c *client) StartKubeScheduler() error {
	return c.startService(manager.KubeSchedulerService)
}

func (c *client) StopKubeScheduler() error {
	return c.stopService(manager.KubeSchedulerService)
}

func (c *client) StartKubeControllerManager() error {
	return c.startService(manager.KubeControllerManagerService)
}

func (c *client) StopKubeControllerManager() error {
	return c.stopService(manager.KubeControllerManagerService)
}

func (c *client) startService(serviceName string) error {
	url := util.GenURL(fmt.Sprintf("%s%s/%s/start", c.cfg.Addr, api.APIPrefix, serviceName))
	if _, err := c.post(url, nil); err != nil {
		klog.Errorf("failed to post %s: %v", url, err)
		return err
	}

	return nil
}

func (c *client) stopService(serviceName string) error {
	url := util.GenURL(fmt.Sprintf("%s%s/%s/stop", c.cfg.Addr, api.APIPrefix, serviceName))
	if _, err := c.post(url, nil); err != nil {
		klog.Errorf("failed to post %s: %v", url, err)
		return err
	}

	return nil
}
