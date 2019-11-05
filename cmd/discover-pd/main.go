package main

import (
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/discovery"
	"github.com/pingcap/tidb-operator/pkg/discovery/server"
	"github.com/pingcap/tidb-operator/pkg/version"
	"k8s.io/component-base/logs"
)

var (
	printVersion bool
	port         int
	scheme       string
	replicas     int
)

func init() {
	flag.BoolVar(&printVersion, "V", false, "Show version and quit")
	flag.BoolVar(&printVersion, "version", false, "Show version and quit")
	flag.StringVar(&scheme, "scheme", "http", "protocol scheme")
	flag.IntVar(&port, "port", 10261, "The port that the tidb discovery's http service runs on (default 10261)")
	flag.IntVar(&replicas, "replicas", 3, "The number of PD replicas to expect")
	flag.Parse()
}

func main() {
	if printVersion {
		version.PrintVersionInfo()
		os.Exit(0)
	}
	version.LogVersionInfo()

	// run pprof
	go func(){
		for true {
			glog.Error(http.ListenAndServe(":6060", nil)) 
			time.Sleep(5 * time.Second)
		}
	}()

	RunDiscoveryService(port)
}

// RunDiscoveryService blocks forever
// Errors in this function are fatal
func RunDiscoveryService(port int) {
	logs.InitLogs()
	defer logs.FlushLogs()

	td := discovery.NewTiDBDiscoveryImmediate(getCluster{})
	glog.Fatal(server.StartServer(td, port))
}

type getCluster struct{}

func (gc getCluster) GetCluster(clusterID string) (discovery.Cluster, error) {
	return discovery.Cluster{
		Scheme:          scheme,
		ResourceVersion: "1",
		Replicas:        int32(replicas),
	}, nil
}
