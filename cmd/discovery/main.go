package main

import (
	"flag"
	_ "net/http/pprof"
	"os"

	"github.com/pingcap/tidb-operator/pkg/discovery/k8s"
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
		k8s.PrintVersionInfo()
		os.Exit(0)
	}
	k8s.LogVersionInfo()
	k8s.RunDiscoveryService(port)
}
