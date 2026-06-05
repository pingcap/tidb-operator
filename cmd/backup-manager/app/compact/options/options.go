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
// See the License for the specific language governing permissions and
// limitations under the License.

package options

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apis/util/config"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	fromTSUnset  = math.MaxUint64
	untilTSUnset = 0
)

type CompactOpts struct {
	FromTS                    uint64
	UntilTS                   uint64
	Name                      string
	Concurrency               uint64
	PhysicalFileCacheCapacity string
	ShardIndex                int
	ShardCount                int
	Sharded                   bool
	Namespace                 string `json:"namespace"`
	ResourceName              string `json:"resourceName"`
	TikvVersion               string `json:"tikvVersion"`
}

func ParseCompactOptions(compact *v1alpha1.CompactBackup, opts *CompactOpts) error {

	startTs, err := config.ParseTSString(compact.Spec.StartTs)
	if err != nil {
		return errors.Annotatef(err, "failed to parse startTs %s", compact.Spec.StartTs)
	}
	endTs, err := config.ParseTSString(compact.Spec.EndTs)
	if err != nil {
		return errors.Annotatef(err, "failed to parse endTs %s", compact.Spec.EndTs)
	}
	opts.FromTS = startTs
	opts.UntilTS = endTs

	opts.Name = strings.TrimSpace(compact.Spec.Name)
	opts.Concurrency = uint64(compact.Spec.Concurrency)
	opts.PhysicalFileCacheCapacity = strings.TrimSpace(compact.Spec.PhysicalFileCacheCapacity)
	if opts.PhysicalFileCacheCapacity == "" {
		opts.PhysicalFileCacheCapacity = "0"
	}
	opts.Sharded = compact.Spec.Mode == v1alpha1.CompactModeSharded
	opts.ShardIndex = 0
	opts.ShardCount = 0
	if compact.Spec.ShardCount != nil {
		opts.ShardCount = int(*compact.Spec.ShardCount)
	}
	if opts.Sharded {
		opts.ShardIndex, err = resolveShardIndex(opts.ShardCount)
		if err != nil {
			return err
		}
	}

	if err := opts.Verify(); err != nil {
		return err
	}

	return nil
}

func (c *CompactOpts) Verify() error {
	if c.FromTS == fromTSUnset {
		return errors.New("from-ts must be set")
	}
	// UntilTS unset is valid only in sharded CCR checkpoint mode: tikv-ctl
	// reads the until-ts from the log-backup global checkpoint via
	// --crr-checkpoint-prefix. Non-sharded compact keeps the existing
	// requirement that EndTs/UntilTS must be set explicitly.
	if c.UntilTS == untilTSUnset && !c.Sharded {
		return errors.New("until-ts must be set")
	}
	if c.UntilTS != untilTSUnset && c.UntilTS < c.FromTS {
		return errors.Errorf("until-ts %d must be greater than from-ts %d", c.UntilTS, c.FromTS)
	}
	if c.Concurrency <= 0 {
		return errors.Errorf("concurrency %d must be greater than 0", c.Concurrency)
	}
	if c.Sharded {
		c.PhysicalFileCacheCapacity = strings.TrimSpace(c.PhysicalFileCacheCapacity)
		if c.PhysicalFileCacheCapacity == "" {
			c.PhysicalFileCacheCapacity = "0"
		}
		capacity, err := resource.ParseQuantity(c.PhysicalFileCacheCapacity)
		if err != nil {
			return errors.Annotatef(err, "invalid physicalFileCacheCapacity %q", c.PhysicalFileCacheCapacity)
		}
		if capacity.Sign() < 0 {
			return errors.New("physicalFileCacheCapacity must be greater than or equal to 0")
		}
		if c.ShardCount <= 0 {
			return errors.Errorf("shard-count %d must be greater than 0", c.ShardCount)
		}
		if c.ShardIndex < 0 || c.ShardIndex >= c.ShardCount {
			return errors.Errorf("kubernetes shard-index %d must be in range [0, %d); tikv-ctl --shard uses 1..%d after conversion", c.ShardIndex, c.ShardCount, c.ShardCount)
		}
	}
	return nil
}

func resolveShardIndex(shardCount int) (int, error) {
	value, ok := os.LookupEnv("JOB_COMPLETION_INDEX")
	if !ok {
		return 0, fmt.Errorf("JOB_COMPLETION_INDEX must be set in sharded mode")
	}

	index, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("failed to parse JOB_COMPLETION_INDEX %q: %w", value, err)
	}
	if index < 0 || index >= shardCount {
		return 0, fmt.Errorf("JOB_COMPLETION_INDEX %d out of range [0, %d); tikv-ctl --shard uses 1..%d after conversion", index, shardCount, shardCount)
	}
	return index, nil
}
