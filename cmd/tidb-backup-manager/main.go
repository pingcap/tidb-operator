// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"os"

	"k8s.io/klog/v2"

	"github.com/pingcap/tidb-operator/v2/cmd/tidb-backup-manager/app"
)

func main() {
	klog.InitFlags(nil)
	// Opt into the new klog behavior so that -stderrthreshold is honored even
	// when -logtostderr=true (the default).
	// Ref: kubernetes/klog#212, kubernetes/klog#432
	if err := flag.Set("legacy_stderr_threshold_behavior", "false"); err != nil {
		fmt.Fprintf(os.Stderr, "failed to set legacy_stderr_threshold_behavior: %v\n", err)
		os.Exit(1)
	}
	if err := flag.Set("stderrthreshold", "INFO"); err != nil {
		fmt.Fprintf(os.Stderr, "failed to set stderrthreshold: %v\n", err)
		os.Exit(1)
	}
	if err := app.Run(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
