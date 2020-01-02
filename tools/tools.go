// Copyright 2019 PingCAP, Inc.
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

// +build tools

// Tool dependencies are tracked here to make go module happy
// Refer https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
package tools

import (
	_ "k8s.io/code-generator"
	_ "sigs.k8s.io/apiserver-builder-alpha/cmd/apiregister-gen"

	// workaround for https://github.com/pingcap/tidb-operator/issues/1095
	// TODO remove this if we 1) avoid the issue in a better way or 2) go both
	// in local development and ci have the bug fixed
	_ "github.com/fatih/color"
)
