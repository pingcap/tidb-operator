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
	"math"

	"github.com/juju/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apis/util/config"
)

const (
	fromTSUnset  = math.MaxUint64
	untilTSUnset = 0
)

type CompactOpts struct {
	FromTS       uint64
	UntilTS      uint64
	Name         string
	Concurrency  uint64
	Namespace    string `json:"namespace"`
	ResourceName string `json:"resourceName"`
	TikvVersion  string `json:"tikvVersion"`
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

	opts.Name = compact.ObjectMeta.Name
	opts.Concurrency = uint64(compact.Spec.Concurrency)

	if err := opts.Verify(); err != nil {
		return err
	}

	return nil
}

func (c *CompactOpts) Verify() error {
	if c.UntilTS < c.FromTS {
		if c.UntilTS == untilTSUnset {
			return errors.New("until-ts must be set")
		}
		if c.FromTS == fromTSUnset {
			return errors.New("from-ts must be set")
		}
		return errors.Errorf("until-ts %d must be greater than from-ts %d", c.UntilTS, c.FromTS)
	}
	if c.Concurrency <= 0 {
		return errors.Errorf("concurrency %d must be greater than 0", c.Concurrency)
	}
	return nil
}
