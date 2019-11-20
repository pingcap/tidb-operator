package client

import (
	"flag"
	"fmt"
	"os"
	"time"

	exampleagg "github.com/pingcap/tidb-operator/tests/pkg/apiserver/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/tests/slack"

	"github.com/juju/errors"
	asclientset "github.com/pingcap/advanced-statefulset/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/typed/pingcap/v1alpha1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	masterUrl      string
	kubeconfigPath string
)

func init() {
	flag.StringVar(&kubeconfigPath, "kubeconfig", "",
		"path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterUrl, "master", "",
		"address of the Kubernetes API server. Overrides any value in kubeconfig. "+
			"Only required if out-of-cluster.")
}

func NewCliOrDie() (versioned.Interface, kubernetes.Interface, asclientset.Interface) {
	cfg, err := GetConfig()
	if err != nil {
		slack.NotifyAndPanic(err)
	}

	return buildClientsOrDie(cfg)
}

// NewExampleAggCliOrDie create new client of the example.pingcap.com resource group hosted by our test apiserver
func NewExampleAggCliOrDie() *exampleagg.Clientset {

	cfg, err := GetConfig()
	if err != nil {
		slack.NotifyAndPanic(fmt.Errorf("Error get client rest config, %v", err))
	}
	cli, err := exampleagg.NewForConfig(cfg)
	if err != nil {
		slack.NotifyAndPanic(fmt.Errorf("Error create client of example.pingcap.com group, %v", err))
	}
	return cli
}

func GetConfig() (*rest.Config, error) {
	// If kubeconfigPath provided, use that
	if len(kubeconfigPath) > 0 {
		return clientcmd.BuildConfigFromFlags(masterUrl, kubeconfigPath)
	}
	// If an env variable is specified with the config locaiton, use that
	if len(os.Getenv("KUBECONFIG")) > 0 {
		return clientcmd.BuildConfigFromFlags(masterUrl, os.Getenv("KUBECONFIG"))
	}
	// If no explicit location, try the in-cluster config
	if c, err := rest.InClusterConfig(); err == nil {
		return c, nil
	}

	return nil, fmt.Errorf("could not locate a kubeconfig")
}

func GetConfigOrDie() *rest.Config {
	cfg, err := GetConfig()
	if err != nil {
		slack.NotifyAndPanic(fmt.Errorf("Error getting kubernetes client config %v", err))
	}
	return cfg
}

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
		slack.NotifyAndPanic(err)
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

func buildClientsOrDie(cfg *rest.Config) (versioned.Interface, kubernetes.Interface, asclientset.Interface) {
	cfg.Timeout = 30 * time.Second
	cli, err := versioned.NewForConfig(cfg)
	if err != nil {
		slack.NotifyAndPanic(err)
	}

	kubeCli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		slack.NotifyAndPanic(err)
	}

	asCli, err := asclientset.NewForConfig(cfg)
	if err != nil {
		slack.NotifyAndPanic(err)
	}

	return cli, kubeCli, asCli
}
