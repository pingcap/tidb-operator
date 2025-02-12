package v1alpha1

import (
	"fmt"
	"hash/fnv"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/util/rand"
)

const (
	// BackupNameTimeFormat is the time format for generate backup CR name
	BackupNameTimeFormat = "2006-01-02t15-04-05"
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
