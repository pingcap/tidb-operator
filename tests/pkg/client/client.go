package client

import (
	"time"

	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func NewCliOrDie() (versioned.Interface, kubernetes.Interface) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}

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
