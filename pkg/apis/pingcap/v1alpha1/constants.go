// Copyright 2021 PingCAP, Inc.
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

package v1alpha1

import "strconv"

const (
	// BackupNameTimeFormat is the time format for generate backup CR name
	BackupNameTimeFormat = "2006-01-02t15-04-05"

	// DefaultTidbUser is the default tidb user for login tidb cluster
	DefaultTidbUser = "root"
)

var (
	// DefaultTiDBServerPort is the default tidb cluster port for connecting
	// It is used both fo Pods and Services.
	DefaultTiDBServerPort = int32(4000)
	// `string` type so that they can be set by `go build -ldflags "-X ..."`
	customPortTiDBServer = "4000"
)

func init() {
	if port, err := strconv.Atoi(customPortTiDBServer); err == nil {
		DefaultTiDBServerPort = int32(port)
	} else {
		panic(err)
	}
}
