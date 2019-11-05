package main

import (
	"flag"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/pingcap/tidb-operator/pkg/version"
	"github.com/pingcap/tidb-operator/pkg/discovery/k8s"
	glog "k8s.io/klog"
)

var (
	printVersion bool
	port         int
)

func init() {
	flag.BoolVar(&printVersion, "V", false, "Show version and quit")
	flag.BoolVar(&printVersion, "version", false, "Show version and quit")
	flag.IntVar(&port, "port", 10261, "The port that the tidb discovery's http service runs on (default 10261)")
	flag.Parse()
}

func main() {
	if printVersion {
		version.PrintVersionInfo()
		os.Exit(0)
	}

	// run pprof
	go func(){
		for true {
			glog.Error(http.ListenAndServe(":6060", nil)) 
			time.Sleep(5 * time.Second)
		}
	}()

	version.LogVersionInfo()
	k8s.RunDiscoveryService(port)
}
