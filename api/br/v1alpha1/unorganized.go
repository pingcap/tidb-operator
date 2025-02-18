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

package v1alpha1

import (
	"fmt"
	"hash/fnv"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/util/rand"
)

const (
	// DefaultTidbUser is the default tidb user for login tidb cluster
	DefaultTidbUser = "root"
)

const (
	// BackupNameTimeFormat is the time format for generate backup CR name
	BackupNameTimeFormat = "2006-01-02t15-04-05"

	BackupTimestampFormat = "2006-01-02 15:04:05"
)

// ParseTSString supports TSO or datetime, e.g. '400036290571534337', '2006-01-02 15:04:05'
func ParseTSString(ts string) (uint64, error) {
	if len(ts) == 0 {
		return 0, nil
	}
	if tso, err := strconv.ParseUint(ts, 10, 64); err == nil {
		return tso, nil
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05", ts, time.Local)
	if err != nil {
		t, err = time.Parse(time.RFC3339, ts)
		if err != nil {
			return 0, fmt.Errorf("cannot parse ts string %s, err: %v", ts, err)
		}
	}
	return GoTimeToTS(t), nil
}

// GoTimeToTS converts a Go time to uint64 timestamp.
// port from tidb.
func GoTimeToTS(t time.Time) uint64 {
	ts := (t.UnixNano() / int64(time.Millisecond)) << 18
	return uint64(ts)
}

// HashContents hashes the contents using FNV hashing. The returned hash will be a safe encoded string to avoid bad words.
func HashContents(contents []byte) string {
	hf := fnv.New32()
	if len(contents) > 0 {
		hf.Write(contents)
	}
	return rand.SafeEncodeString(fmt.Sprint(hf.Sum32()))
}
