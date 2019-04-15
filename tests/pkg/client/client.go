package client

import (
	"time"

	"github.com/juju/errors"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/typed/pingcap.com/v1alpha1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func NewCliOrDie() (versioned.Interface, kubernetes.Interface) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}

	return buildClientsOrDie(cfg)
}

func NewOutOfClusterCliOrDie(kubeconfig string) (versioned.Interface, kubernetes.Interface) {
	// TODO: support context selection, current context will be used now
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	return buildClientsOrDie(cfg)
}

var (
	masterUrl      string
	kubeconfigPath string
)

type Client interface {
	kubernetes.Interface
	PingcapV1alpha1() v1alpha1.PingcapV1alpha1Interface
}

func Union(kube kubernetes.Interface, tidb versioned.Interface) Client {
	return &client{Interface: kube, pingcap: tidb}
}

func NewOrDie() Client {
	cfg, err := clientcmd.BuildConfigFromFlags(masterUrl, kubeconfigPath)
	if err != nil {
		panic(err)
	}
	return Union(kubernetes.NewForConfigOrDie(cfg), versioned.NewForConfigOrDie(cfg))
}

type client struct {
	kubernetes.Interface
	pingcap versioned.Interface
}

func (cli *client) PingcapV1alpha1() v1alpha1.PingcapV1alpha1Interface {
	return cli.pingcap.PingcapV1alpha1()
}

func SetConfigPath(path string) {
	kubeconfigPath = path
}

func SetMasterURL(url string) {
	masterUrl = url
}

func LoadConfig() (*rest.Config, error) {
	cfg, err := clientcmd.BuildConfigFromFlags(masterUrl, kubeconfigPath)
	return cfg, errors.Trace(err)
}

func buildClientsOrDie(cfg *rest.Config) (versioned.Interface, kubernetes.Interface) {
	cfg.Timeout = 30 * time.Second
	cli, err := versioned.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}

	kubeCli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}

	return cli, kubeCli
}
