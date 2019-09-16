// Copyright 2019. PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/pkg/version"
	"k8s.io/apiserver/pkg/util/logs"
)

const (
	timeout = 5 * time.Second
)

var (
	printVersion          bool
	waitForInitialization bool
	waitForLeader         bool
)

func init() {
	flag.BoolVar(&printVersion, "V", false, "Show version and quit")
	flag.BoolVar(&printVersion, "version", false, "Show version and quit")
	flag.BoolVar(&waitForInitialization, pdapi.WaitForInitializationFlag, false, "Wait for initialization of the cluster. This means all replicas have come online")
	flag.BoolVar(&waitForLeader, pdapi.WaitForLeaderFlag, false, "Wait for just the presence of a PD leader.")
	flag.Parse()
}

func pdHasLeader(pdClient pdapi.PDClient) (bool, error) {
	memberInfo, err := pdClient.GetPDLeader()
	if err != nil {
		return false, err
	}
	return memberInfo != nil, nil
}

// On older versions this will return the empty semver version 0.0.0
// Most tidb-operator users will be using version 3.0+ which has the version field.
func pdVersion(pdClient pdapi.PDClient) (string, error) {
	conf, err := pdClient.GetConfig()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s", conf.ClusterVersion), nil
}

// wait-for-pd waits for 1 PD to be running
func main() {
	if printVersion {
		version.PrintVersionInfo()
		os.Exit(0)
	}
	version.LogVersionInfo()

	logs.InitLogs()
	defer logs.FlushLogs()

	oneSecond, err := time.ParseDuration("1s")
	if err != nil {
		glog.Fatalf("Error parsing time %v", err)
	}

	tcName := os.Getenv("CLUSTER_NAME")
	if tcName == "" {
		glog.Fatalf("Expected CLUSTER_NAME env variable to be set")
	}
	namespace := os.Getenv("NAMESPACE")
	if namespace == "" {
		glog.Fatalf("Expected NAMESPACE env variable to be set")
	}

	// This is just a check for intialization at startup, it doesn't seem valuable to try to hide it with https
	// Just being able to observe that there is network traffic going on with this program pretty much tells you everything you would want to know here.
	pdClient := pdapi.NewPDClient(pdapi.PdClientURL(pdapi.Namespace(namespace), tcName, "http"), timeout, false)

	var waitFunction func() bool

	waitForLeaderFunc := func() bool {
		hasLeader, err := pdHasLeader(pdClient)
		if err != nil {
			glog.Infof("Error using pdClient to get members %v", err)
		} else if hasLeader {
			glog.Infof("Found a PD member. Exiting now.")
			return true
		} else {
			glog.Infof("PD Leader not found")
		}
		return false
	}

	waitForInitializationFunc := func() bool {
		isInit, err := pdClient.GetClusterInitialized()
		if err != nil {
			glog.Infof("Error using pdClient to get cluster status %v", err)
		} else if isInit == nil {
			version, verr := pdVersion(pdClient)
			if verr != nil || version == "" {
				glog.Errorf("Error using pdClient to get cluster version %v", verr)
			}
			glog.Warningf("For this PD version %s the cluster status API does not support is_initialized. Will now wait for just a PD leader", version)
			waitFunction = waitForLeaderFunc
		} else if *isInit {
			glog.Infof("Cluster is inititialized. Exiting now.")
			return true
		} else {
			glog.Infof("PD is not initialized")
		}
		return false
	}

	if waitForInitialization {
		waitFunction = waitForInitializationFunc
	} else if waitForLeader {
		waitFunction = waitForLeaderFunc
	} else {
		glog.Fatalf("Expected either the flag --initialization or --leader")
	}

	for {
		if waitFunction() {
			break
		}
		time.Sleep(oneSecond)
	}
}
