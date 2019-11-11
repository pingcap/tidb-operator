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
	allFeatures     = sets.NewString(StableScheduling)
	defaultFeatures = map[string]bool{
		StableScheduling:    true,
		AdvancedStatefulSet: false,
	}
	// DefaultFeatureGate is a shared global FeatureGate.
	DefaultFeatureGate FeatureGate = NewFeatureGate()
)

const (
	// StableScheduling controls stable scheduling of TiDB members.
	StableScheduling string = "StableScheduling"

	// AdvancedStatefulSet controls whether to use AdvancedStatefulSet to manage pods
	AdvancedStatefulSet string = "AdvancedStatefulSet"
)

type FeatureGate interface {
	// AddFlag adds a flag for setting global feature gates to the specified FlagSet.
	AddFlag(flagset *flag.FlagSet)
	// Enabled returns true if the key is enabled.
	Enabled(key string) bool
}

type featureGate struct {
	enabledFeatures map[string]bool
}

func (f *featureGate) AddFlag(flagset *flag.FlagSet) {
	flag.Var(utilflags.NewMapStringBool(&f.enabledFeatures), "features", fmt.Sprintf("A set of key={true,false} pairs to enable/disable features, available features: %s", strings.Join(allFeatures.List(), ",")))
}

func (f *featureGate) Enabled(key string) bool {
	if b, ok := f.enabledFeatures[key]; ok {
		return b
	}
	return false
}

func NewFeatureGate() FeatureGate {
	f := &featureGate{
		enabledFeatures: make(map[string]bool),
	}
	for k, v := range defaultFeatures {
		f.enabledFeatures[k] = v
	}
	return f
}
