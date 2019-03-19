package tests

import (
	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/tests/pkg/fault-trigger/client"
	"github.com/pingcap/tidb-operator/tests/pkg/fault-trigger/manager"
	"k8s.io/client-go/kubernetes"
)

type FaultTriggerActions interface {
	StopNode(physicalNode string, node string) error
	StartNode(physicalNode string, node string) error
	StopETCD(nodes ...string) error
	StartETCD(nodes ...string) error
	StopKubelet(node string) error
	StartKubelet(node string) error
	// TODO: support more faults
	// StopKubeAPIServer() error
	// StartKubeAPIServer() error
	// StopKubeControllerManager() error
	// StartKubeControllerManager() error
	// StopKubeScheduler() error
	// StartKubeScheduler() error
	// StopKubeProxy(node string) error
	// StartKubeProxy(node string) error
	// DiskCorruption(node string) error
	// NetworkPartition(fromNode, toNode string) error
	// NetworkDelay(fromNode, toNode string) error
	// DockerCrash(nodeName string) error
}

func NewFaultTriggerAction(cli versioned.Interface, kubeCli kubernetes.Interface, cfg *Config) FaultTriggerActions {
	return &faultTriggerActions{
		cli:       cli,
		kubeCli:   kubeCli,
		pdControl: controller.NewDefaultPDControl(),
		cfg:       cfg,
	}
}

type faultTriggerActions struct {
	cli       versioned.Interface
	kubeCli   kubernetes.Interface
	pdControl controller.PDControlInterface
	cfg       *Config
}

func (fa *faultTriggerActions) StopNode(physicalNode string, node string) error {
	faultCli := client.NewClient(client.Config{
		Addr: physicalNode,
	})

	err := faultCli.StopVM(&manager.VM{
		IP: node,
	})

	if err != nil {
		glog.Errorf("failed to stop %s node %s: %v", physicalNode, node, err)
		return err
	}

	return nil
}

func (fa *faultTriggerActions) StartNode(physicalNode string, node string) error {
	faultCli := client.NewClient(client.Config{
		Addr: physicalNode,
	})

	err := faultCli.StartVM(&manager.VM{
		IP: node,
	})

	if err != nil {
		glog.Errorf("failed to start %s node %s: %v", physicalNode, node, err)
		return err
	}

	return nil
}

// StopETCD stops the etcd service.
// If the `nodes` is empty, StopEtcd will stop all etcd service.
func (fa *faultTriggerActions) StopETCD(nodes ...string) error {
	if len(nodes) == 0 {
		for _, ns := range fa.cfg.ETCDs {
			nodes = append(nodes, ns.Nodes...)
		}
	}

	for _, node := range nodes {
		faultCli := client.NewClient(client.Config{
			Addr: node,
		})

		if err := faultCli.StopETCD(); err != nil {
			glog.Errorf("failed to stop etcd %s: %v", node, err)
			return err
		}
	}

	return nil
}

// StartETCD starts the etcd service.
// If the `nodes` is empty, StartETCD will start all etcd service.
func (fa *faultTriggerActions) StartETCD(nodes ...string) error {
	if len(nodes) == 0 {
		for _, ns := range fa.cfg.ETCDs {
			nodes = append(nodes, ns.Nodes...)
		}
	}

	for _, node := range nodes {
		faultCli := client.NewClient(client.Config{
			Addr: node,
		})

		if err := faultCli.StartETCD(); err != nil {
			glog.Errorf("failed to start etcd %s: %v", node, err)
			return err
		}
	}

	return nil
}

// StopKubelet stops the kubelet service.
func (fa *faultTriggerActions) StopKubelet(node string) error {
	faultCli := client.NewClient(client.Config{
		Addr: node,
	})

	if err := faultCli.StopKubelet(); err != nil {
		glog.Errorf("failed to stop kubelet %s: %v", node, err)
		return err
	}

	return nil
}

// StartKubelet starts the kubelet service.
func (fa *faultTriggerActions) StartKubelet(node string) error {
	faultCli := client.NewClient(client.Config{
		Addr: node,
	})

	if err := faultCli.StartKubelet(); err != nil {
		glog.Errorf("failed to start kubelet %s: %v", node, err)
		return err
	}

	return nil
}
