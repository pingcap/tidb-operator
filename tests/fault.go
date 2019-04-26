package tests

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/tests/pkg/fault-trigger/client"
	"github.com/pingcap/tidb-operator/tests/pkg/fault-trigger/manager"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

const (
	startAction = "start"
	stopAction  = "stop"
)

type FaultTriggerActions interface {
	CheckAndRecoverEnv() error
	CheckAndRecoverEnvOrDie()
	StopNode() (string, string, time.Time, error)
	StopNodeOrDie() (string, string, time.Time)
	StartNode(physicalNode string, node string) error
	StartNodeOrDie(physicalNode string, node string)
	StopETCD(nodes ...string) error
	StopETCDOrDie(nodes ...string)
	StartETCD(nodes ...string) error
	StartETCDOrDie(nodes ...string)
	StopKubelet(node string) error
	StartKubelet(node string) error
	StopKubeAPIServer(node string) error
	StopKubeAPIServerOrDie(node string)
	StartKubeAPIServer(node string) error
	StartKubeAPIServerOrDie(node string)
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

func (fa *faultTriggerActions) CheckAndRecoverEnv() error {
	glog.Infof("ensure all nodes are running")
	for _, physicalNode := range fa.cfg.Nodes {
		for _, vNode := range physicalNode.Nodes {
			err := fa.StartNode(physicalNode.PhysicalNode, vNode)
			if err != nil {
				return err
			}
		}
	}
	glog.Infof("ensure all etcds are running")
	err := fa.StartETCD()
	if err != nil {
		return err
	}
	glog.Infof("ensure all kubelets are running")
	for _, physicalNode := range fa.cfg.Nodes {
		for _, vNode := range physicalNode.Nodes {
			err := fa.StartKubelet(vNode)
			if err != nil {
				return err
			}
		}
	}
	glog.Infof("ensure all static pods are running")
	for _, physicalNode := range fa.cfg.APIServers {
		for _, vNode := range physicalNode.Nodes {
			err := fa.StartKubeAPIServer(vNode)
			if err != nil {
				return err
			}
			err = fa.StartKubeControllerManager(vNode)
			if err != nil {
				return err
			}
			err = fa.StartKubeScheduler(vNode)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (fa *faultTriggerActions) CheckAndRecoverEnvOrDie() {
	if err := fa.CheckAndRecoverEnv(); err != nil {
		glog.Fatal(err)
	}
}

func (fa *faultTriggerActions) StopNode() (string, string, time.Time, error) {
	now := time.Now()
	node, err := getFaultNode(fa.kubeCli)
	if err != nil {
		return "", "", now, err
	}
	glog.Infof("selecting %s as the node to failover", node)

	physicalNode := getPhysicalNode(node, fa.cfg)

	if physicalNode == "" {
		return "", "", now, fmt.Errorf("physical node is empty")
	}

	faultCli := client.NewClient(client.Config{
		Addr: fa.genFaultTriggerAddr(physicalNode),
	})

	if err := faultCli.StopVM(&manager.VM{
		IP: node,
	}); err != nil {
		glog.Errorf("failed to stop node %s on physical node: %s: %v", node, physicalNode, err)
		return "", "", now, err
	}

	glog.Infof("node %s on physical node %s is stopped", node, physicalNode)
	return physicalNode, node, now, nil
}

func (fa *faultTriggerActions) StopNodeOrDie() (string, string, time.Time) {
	var pn string
	var n string
	var err error
	var now time.Time
	if pn, n, now, err = fa.StopNode(); err != nil {
		panic(err)
	}
	return pn, n, now
}

func (fa *faultTriggerActions) StartNode(physicalNode string, node string) error {
	faultCli := client.NewClient(client.Config{
		Addr: fa.genFaultTriggerAddr(physicalNode),
	})

	vms, err := faultCli.ListVMs()
	if err != nil {
		return err
	}

	for _, vm := range vms {
		if vm.IP == node && vm.Status == "running" {
			return nil
		}
	}

	if err := faultCli.StartVM(&manager.VM{
		IP: node,
	}); err != nil {
		glog.Errorf("failed to start node %s on physical node %s: %v", node, physicalNode, err)
		return err
	}

	glog.Infof("node %s on physical node %s is started", node, physicalNode)

	return nil
}

func (fa *faultTriggerActions) StartNodeOrDie(physicalNode string, node string) {
	if err := fa.StartNode(physicalNode, node); err != nil {
		panic(err)
	}
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

func (fa *faultTriggerActions) StopETCDOrDie(nodes ...string) {
	if err := fa.StopETCD(nodes...); err != nil {
		panic(err)
	}
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

func (fa *faultTriggerActions) StartETCDOrDie(nodes ...string) {
	if err := fa.StartETCD(nodes...); err != nil {
		panic(err)
	}
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

func (fa *faultTriggerActions) StopKubeAPIServerOrDie(node string) {
	if err := fa.StopKubeAPIServer(node); err != nil {
		panic(err)
	}
}

// StartKubeAPIServer starts the apiserver service.
func (fa *faultTriggerActions) StartKubeAPIServer(node string) error {
	return fa.serviceAction(node, manager.KubeAPIServerService, startAction)
}

func (fa *faultTriggerActions) StartKubeAPIServerOrDie(node string) {
	if err := fa.StartKubeAPIServer(node); err != nil {
		panic(err)
	}
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

	glog.V(4).Infof("%s %s %s successfully", action, serverName, node)

	return nil
}

func (fa *faultTriggerActions) genFaultTriggerAddr(node string) string {
	return fmt.Sprintf("%s:%d", node, fa.cfg.FaultTriggerPort)
}

func getMyNodeName() string {
	return os.Getenv("MY_NODE_NAME")
}

func getFaultNode(kubeCli kubernetes.Interface) (string, error) {
	var err error
	var nodes *v1.NodeList
	err = wait.Poll(2*time.Second, 10*time.Second, func() (bool, error) {
		nodes, err = kubeCli.CoreV1().Nodes().List(metav1.ListOptions{})
		if err != nil {
			glog.Errorf("trigger node stop failed when get all nodes, error: %v", err)
			return false, nil
		}

		return true, nil
	})

	if err != nil {
		glog.Errorf("failed to list nodes: %v", err)
		return "", err
	}

	if len(nodes.Items) <= 1 {
		return "", fmt.Errorf("the number of nodes cannot be less than 1")
	}

	myNode := getMyNodeName()

	index := rand.Intn(len(nodes.Items))
	faultNode := nodes.Items[index].Name
	if faultNode != myNode {
		return faultNode, nil
	}

	if index == 0 {
		faultNode = nodes.Items[index+1].Name
	} else {
		faultNode = nodes.Items[index-1].Name
	}

	if faultNode == myNode {
		err := fmt.Errorf("there are at least two nodes with the name %s", myNode)
		glog.Error(err.Error())
		return "", err
	}

	return faultNode, nil
}

func getPhysicalNode(faultNode string, cfg *Config) string {
	var physicalNode string
	for _, nodes := range cfg.Nodes {
		for _, node := range nodes.Nodes {
			if node == faultNode {
				physicalNode = nodes.PhysicalNode
			}
		}
	}

	return physicalNode
}
