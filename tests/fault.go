package tests

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/tests/pkg/fault-trigger/client"
	"github.com/pingcap/tidb-operator/tests/pkg/fault-trigger/manager"
	"k8s.io/client-go/kubernetes"
)

const (
	startAction = "start"
	stopAction  = "stop"
)

type FaultTriggerActions interface {
	StopNode(physicalNode string, node string) error
	StartNode(physicalNode string, node string) error
	StopETCD(nodes ...string) error
	StartETCD(nodes ...string) error
	StopKubelet(node string) error
	StartKubelet(node string) error
	StopKubeAPIServer(node string) error
	StartKubeAPIServer(node string) error
	StopKubeControllerManager(node string) error
	StartKubeControllerManager(node string) error
	StopKubeScheduler(node string) error
	StartKubeScheduler(node string) error
	// TODO: support more faults
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
		Addr: fa.genFaultTriggerAddr(physicalNode),
	})

	err := faultCli.StopVM(&manager.VM{
		IP: node,
	})

	if err != nil {
		glog.Errorf("failed to stop node %s on physical node: %s: %v", node, physicalNode, err)
		return err
	}

	glog.Infof("node %s on physical node %s is stopped", node, physicalNode)

	return nil
}

func (fa *faultTriggerActions) StartNode(physicalNode string, node string) error {
	faultCli := client.NewClient(client.Config{
		Addr: fa.genFaultTriggerAddr(physicalNode),
	})

	err := faultCli.StartVM(&manager.VM{
		IP: node,
	})

	if err != nil {
		glog.Errorf("failed to start node %s on physical node %s: %v", node, physicalNode, err)
		return err
	}

	glog.Infof("node %s on physical node %s is started", physicalNode, node)

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
		if err := fa.serviceAction(node, manager.ETCDService, stopAction); err != nil {
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
		if err := fa.serviceAction(node, manager.ETCDService, startAction); err != nil {
			return err
		}
	}

	return nil
}

// StopKubelet stops the kubelet service.
func (fa *faultTriggerActions) StopKubelet(node string) error {
	return fa.serviceAction(node, manager.KubeletService, stopAction)
}

// StartKubelet starts the kubelet service.
func (fa *faultTriggerActions) StartKubelet(node string) error {
	return fa.serviceAction(node, manager.KubeletService, startAction)
}

// // StopKubeProxy stops the kube-proxy service.
//func (fa *faultTriggerActions) StopKubeProxy(node string) error {
//	return fa.serviceAction(node, manager.KubeProxyService, stopAction)
//}
//
//// StartKubeProxy starts the kube-proxy service.
//func (fa *faultTriggerActions) StartKubeProxy(node string) error {
//	return fa.serviceAction(node, manager.KubeProxyService, startAction)
//}

// StopKubeScheduler stops the kube-scheduler service.
func (fa *faultTriggerActions) StopKubeScheduler(node string) error {
	return fa.serviceAction(node, manager.KubeSchedulerService, stopAction)
}

// StartKubeScheduler starts the kube-scheduler service.
func (fa *faultTriggerActions) StartKubeScheduler(node string) error {
	return fa.serviceAction(node, manager.KubeSchedulerService, startAction)
}

// StopKubeControllerManager stops the kube-controller-manager service.
func (fa *faultTriggerActions) StopKubeControllerManager(node string) error {
	return fa.serviceAction(node, manager.KubeControllerManagerService, stopAction)
}

// StartKubeControllerManager starts the kube-controller-manager service.
func (fa *faultTriggerActions) StartKubeControllerManager(node string) error {
	return fa.serviceAction(node, manager.KubeControllerManagerService, startAction)
}

// StopKubeAPIServer stops the apiserver service.
func (fa *faultTriggerActions) StopKubeAPIServer(node string) error {
	return fa.serviceAction(node, manager.KubeAPIServerService, stopAction)
}

// StartKubeAPIServer starts the apiserver service.
func (fa *faultTriggerActions) StartKubeAPIServer(node string) error {
	return fa.serviceAction(node, manager.KubeAPIServerService, startAction)
}

func (fa *faultTriggerActions) serviceAction(node string, serverName string, action string) error {
	faultCli := client.NewClient(client.Config{
		Addr: fa.genFaultTriggerAddr(node),
	})

	var err error
	switch action {
	case startAction:
		switch serverName {
		case manager.KubeletService:
			err = faultCli.StartKubelet()
		case manager.KubeSchedulerService:
			err = faultCli.StartKubeScheduler()
		case manager.KubeControllerManagerService:
			err = faultCli.StartKubeControllerManager()
		case manager.KubeAPIServerService:
			err = faultCli.StartKubeAPIServer()
		case manager.ETCDService:
			err = faultCli.StartETCD()
		default:
			err = fmt.Errorf("%s %s is not supported", action, serverName)
			return err
		}
	case stopAction:
		switch serverName {
		case manager.KubeletService:
			err = faultCli.StopKubelet()
		case manager.KubeSchedulerService:
			err = faultCli.StopKubeScheduler()
		case manager.KubeControllerManagerService:
			err = faultCli.StopKubeControllerManager()
		case manager.KubeAPIServerService:
			err = faultCli.StopKubeAPIServer()
		case manager.ETCDService:
			err = faultCli.StopETCD()
		default:
			err = fmt.Errorf("%s %s is not supported", action, serverName)
		}
	default:
		err = fmt.Errorf("action %s is not supported", action)
		return err
	}

	if err != nil {
		glog.Errorf("failed to %s %s %s: %v", action, serverName, node, err)
		return err
	}

	glog.Infof("%s %s %s successfully", action, serverName, node)

	return nil
}

func (fa *faultTriggerActions) genFaultTriggerAddr(node string) string {
	return fmt.Sprintf("%s:%d", node, fa.cfg.FaultTriggerPort)
}
