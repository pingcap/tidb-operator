package main

import (
	"flag"
	"github.com/pingcap/tidb-operator/pkg/util"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/discovery/server"
	"github.com/pingcap/tidb-operator/version"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/util/logs"
)

var (
	printVersion bool
	port         int
	kubeConfig   string
	masterURL    string
)

func init() {
	flag.BoolVar(&printVersion, "V", false, "Show version and quit")
	flag.BoolVar(&printVersion, "version", false, "Show version and quit")
	flag.IntVar(&port, "port", 10261, "The port that the tidb discovery's http service runs on (default 10261)")
	flag.StringVar(&kubeConfig, "kubeconfig", "", "The path of kubeconfig file, only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", `The url of the Kubernetes API server,will overrides 
		any value in kubeconfig, only required if out-of-cluster.`)
	flag.Parse()
}

func main() {
	if printVersion {
		version.PrintVersionInfo()
		os.Exit(0)
	}
	version.LogVersionInfo()

	logs.InitLogs()
	defer logs.FlushLogs()

	cfg := util.NewClusterConfig(masterURL, kubeConfig)
	cli, err := versioned.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("failed to create Clientset: %v", err)
	}

	go wait.Forever(func() {
		server.StartServer(cli, port)
	}, 5*time.Second)
	glog.Fatal(http.ListenAndServe(":6060", nil))
}
