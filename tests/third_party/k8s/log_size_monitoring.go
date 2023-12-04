/*
Copyright 2015 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// this file is copied from k8s.io/kubernetes/test/e2e/framework/log_size_monitoring.go @v1.23.17

package k8s

import (
	"sync"
	"time"

	clientset "k8s.io/client-go/kubernetes"
)

// TimestampedSize contains a size together with a time of measurement.
type TimestampedSize struct {
	timestamp time.Time
	size      int
}

// LogSizeGatherer is a worker which grabs a WorkItem from the channel and does assigned work.
type LogSizeGatherer struct {
	stopChannel chan bool
	data        *LogsSizeData
	wg          *sync.WaitGroup
	workChannel chan WorkItem
}

// LogsSizeVerifier gathers data about log files sizes from master and node machines.
// It oversees a <workersNo> workers which do the gathering.
type LogsSizeVerifier struct {
	client      clientset.Interface
	stopChannel chan bool
	// data stores LogSizeData groupped per IP and log_path
	data          *LogsSizeData
	masterAddress string
	nodeAddresses []string
	wg            sync.WaitGroup
	workChannel   chan WorkItem
	workers       []*LogSizeGatherer
}

// LogSizeDataTimeseries is map of timestamped size.
type LogSizeDataTimeseries map[string]map[string][]TimestampedSize

// LogsSizeData is a structure for handling timeseries of log size data and lock.
type LogsSizeData struct {
	data LogSizeDataTimeseries
	lock sync.Mutex
}

// WorkItem is a command for a worker that contains an IP of machine from which we want to
// gather data and paths to all files we're interested in.
type WorkItem struct {
	ip                string
	paths             []string
	backoffMultiplier int
}
