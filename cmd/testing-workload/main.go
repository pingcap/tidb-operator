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
	"database/sql"
	"flag"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

var (
	action   string
	host     string
	user     string
	password string

	durationInMinutes int
	maxConnections    int
	sleepIntervalSec  int
	longTxnSleepSec   int
)

//nolint:mnd // default values
func main() {
	flag.StringVar(&action, "action", "ping", "ping, workload")
	flag.StringVar(&host, "host", "", "host")
	flag.StringVar(&user, "user", "root", "db user")
	flag.StringVar(&password, "password", "", "db password")

	flag.IntVar(&durationInMinutes, "duration", 10, "duration in minutes")
	flag.IntVar(&maxConnections, "max-connections", 30, "max connections")
	flag.IntVar(&sleepIntervalSec, "sleep-interval", 1, "sleep interval in seconds")
	flag.IntVar(&longTxnSleepSec, "long-txn-sleep", 10, "how many seconds to sleep to simulate a long transaction")
	flag.Parse()

	// enable "cleartext client side plugin" for `tidb_auth_token`.
	// ref: https://github.com/go-sql-driver/mysql?tab=readme-ov-file#allowcleartextpasswords
	params := []string{
		"charset=utf8mb4",
		"allowCleartextPasswords=true",
	}

	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@(%s:4000)/test?%s", user, password, host, strings.Join(params, "&")))
	if err != nil {
		panic(err)
	}
	defer db.Close()

	switch action {
	case "ping":
		if err := Ping(db); err != nil {
			panic(err)
		}
	case "workload":
		if err := Workload(db); err != nil {
			panic(err)
		}

	default:
		panic("unknown action: " + action)
	}

	fmt.Println("workload is done")
}
