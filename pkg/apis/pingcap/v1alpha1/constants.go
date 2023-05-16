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

	DefaultTiDBStatusPort = int32(10080)
	customPortTiDBStatus  = "10080"

	DefaultPDClientPort = int32(2379)
	customPortPDClient  = "2379"
	DefaultPDPeerPort   = int32(2380)
	customPortPDPeer    = "2380"

	DefaultTiKVServerPort = int32(20160)
	customPortTiKVServer  = "20160"
	DefaultTiKVStatusPort = int32(20180)
	customPortTiKVStatus  = "20180"

	DefaultTiFlashTcpPort = int32(9000)
	customPortTiFlashTcp  = "9000"
	DefaultTiFlashHttpPort = int32(8123)
	customPortTiFlashHttp  = "8123"
	DefaultTiFlashFlashPort = int32(3930)
	customPortTiFlashFlash  = "3930"
)

func init() {
	if port, err := strconv.ParseUint(customPortTiDBServer, 10, 32); err == nil {
		DefaultTiDBServerPort = int32(port)
	} else {
		panic(err)
	}
	if port, err := strconv.ParseUint(customPortTiDBStatus, 10, 32); err == nil {
		DefaultTiDBStatusPort = int32(port)
	} else {
		panic(err)
	}

	if port, err := strconv.ParseUint(customPortPDClient, 10, 32); err == nil {
		DefaultPDClientPort = int32(port)
	} else {
		panic(err)
	}
	if port, err := strconv.ParseUint(customPortPDPeer, 10, 32); err == nil {
		DefaultPDPeerPort = int32(port)
	} else {
		panic(err)
	}

	if port, err := strconv.ParseUint(customPortTiKVServer, 10, 32); err == nil {
		DefaultTiKVServerPort = int32(port)
	} else {
		panic(err)
	}
	if port, err := strconv.ParseUint(customPortTiKVStatus, 10, 32); err == nil {
		DefaultTiKVStatusPort = int32(port)
	} else {
		panic(err)
	}

	if port, err := strconv.ParseUint(customPortTiFlashTcp, 10, 32); err == nil {
		DefaultTiFlashTcpPort = int32(port)
	} else {
		panic(err)
	}
	if port, err := strconv.ParseUint(customPortTiFlashHttp, 10, 32); err == nil {
		DefaultTiFlashHttpPort = int32(port)
	} else {
		panic(err)
	}
	if port, err := strconv.ParseUint(customPortTiFlashFlash, 10, 32); err == nil {
		DefaultTiFlashFlashPort = int32(port)
	} else {
		panic(err)
	}
}
