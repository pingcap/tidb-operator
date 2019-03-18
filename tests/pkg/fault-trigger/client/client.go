package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pingcap/tidb-operator/tests/pkg/fault-trigger/api"
	"github.com/pingcap/tidb-operator/tests/pkg/fault-trigger/manager"
	"github.com/pingcap/tidb-operator/tests/pkg/util"
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

func (c *client) ListVMs() ([]*manager.VM, error) {
	url := util.GenURL(fmt.Sprintf("%s%s/vms", c.cfg.Addr, api.APIPrefix))
	data, err := c.get(url)
	if err != nil {
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
	if len(vmName) == 0 {
		vmName = vm.IP
	}

	url := util.GenURL(fmt.Sprintf("%s/%s/vm/%s/start", c.cfg.Addr, api.APIPrefix, vmName))
	if _, err := c.get(url); err != nil {
		return err
	}

	return nil
}

func (c *client) StopVM(vm *manager.VM) error {
	if err := vm.Verify(); err != nil {
		return err
	}

	vmName := vm.Name
	if len(vmName) == 0 {
		vmName = vm.IP
	}

	url := util.GenURL(fmt.Sprintf("%s/%s/vm/%s/stop", c.cfg.Addr, api.APIPrefix, vmName))
	if _, err := c.get(url); err != nil {
		return err
	}

	return nil
}

func (c *client) StartETCD() error {
	url := util.GenURL(fmt.Sprintf("%s/%s/etcd/start", c.cfg.Addr, api.APIPrefix))
	if _, err := c.get(url); err != nil {
		return err
	}

	return nil
}

func (c *client) StopETCD() error {
	url := util.GenURL(fmt.Sprintf("%s/%s/etcd/stop", c.cfg.Addr, api.APIPrefix))
	if _, err := c.get(url); err != nil {
		return err
	}

	return nil
}

func (c *client) StartKubelet() error {
	url := util.GenURL(fmt.Sprintf("%s/%s/kubelet/start", c.cfg.Addr, api.APIPrefix))
	if _, err := c.get(url); err != nil {
		return err
	}

	return nil
}

func (c *client) StopKubelet() error {
	url := util.GenURL(fmt.Sprintf("%s/%s/kubelet/stop", c.cfg.Addr, api.APIPrefix))
	if _, err := c.get(url); err != nil {
		return err
	}

	return nil
}
