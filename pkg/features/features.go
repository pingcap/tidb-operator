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

package features

import (
	"flag"
	"fmt"
	"strings"

	utilflags "github.com/pingcap/tidb-operator/pkg/util/flags"
	"k8s.io/apimachinery/pkg/util/sets"
)

var (
	allFeatures = sets.NewString(StableScheduling)
	// DefaultFeatureGate is a shared global FeatureGate.
	DefaultFeatureGate FeatureGate = NewFeatureGate()
)

const (
	// StableScheduling controls stable scheduling of TiDB members.
	StableScheduling string = "StableScheduling"
)

type FeatureGate interface {
	// AddFlag adds a flag for setting global feature gates to the specified FlagSet.
	AddFlag(flagset *flag.FlagSet)
	// Enabled returns true if the key is enabled.
	Enabled(key string) bool
}

type featureGate struct {
	defaultFeatures sets.String
	enabledFeatures sets.String
}

func (f *featureGate) AddFlag(flagset *flag.FlagSet) {
	flag.Var(utilflags.NewStringSetValue(f.defaultFeatures, &f.enabledFeatures), "features", fmt.Sprintf("features to enable, comma-separated list of string, available: %s", strings.Join(allFeatures.List(), ",")))
}

func (f *featureGate) Enabled(key string) bool {
	return f.enabledFeatures.Has(key)
}

func NewFeatureGate() FeatureGate {
	return &featureGate{
		defaultFeatures: sets.NewString(),
		enabledFeatures: sets.NewString(),
	}
}
