// Copyright 2018 PingCAP, Inc.
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

package version

import (
	"fmt"

	"github.com/golang/glog"
)

var (
	// GitSHA will be set during make
	GitSHA = "None"
	// BuildTS and BuildTS will be set during make
	BuildTS = "None"
)

// PrintVersionInfo show version info to Stdout
func PrintVersionInfo() {
	fmt.Println("Git Commit Hash:", GitSHA)
	fmt.Println("UTC Build Time: ", BuildTS)
}

// LogVersionInfo print version info at startup
func LogVersionInfo() {
	glog.Infof("Welcome to TiDB Operator.")
	glog.Infof("Git Commit Hash: %s", GitSHA)
	glog.Infof("UTC Build Time:  %s", BuildTS)
}
