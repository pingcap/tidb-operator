// Copyright 2022 PingCAP, Inc.
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

package volumes

import (
	"fmt"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
)

type selectorFactory struct {
	cache map[v1alpha1.MemberType]*labels.Requirement
}

func (sf *selectorFactory) NewSelector(instance string, mt v1alpha1.MemberType) (labels.Selector, error) {
	selector, err := label.New().Instance(instance).Selector()
	if err != nil {
		return nil, err
	}
	r, ok := sf.cache[mt]
	if !ok {
		return nil, fmt.Errorf("can't get selector for %v", mt)
	}

	return selector.Add(*r), nil
}

// pd => pd
// tidb => tidb
// tikv => tikv
// tiflash => tiflash
// ticdc => ticdc
// pump => pump
func convertMemberTypeToLabelVal(mt v1alpha1.MemberType) string {
	return string(mt)
}

func NewSelectorFactory() (*selectorFactory, error) {
	mts := []v1alpha1.MemberType{
		v1alpha1.PDMemberType,
		v1alpha1.TiProxyMemberType,
		v1alpha1.TiDBMemberType,
		v1alpha1.TiKVMemberType,
		v1alpha1.TiFlashMemberType,
		v1alpha1.TiCDCMemberType,
		v1alpha1.PumpMemberType,
	}

	sf := &selectorFactory{
		cache: make(map[v1alpha1.MemberType]*labels.Requirement),
	}

	for _, mt := range mts {
		req, err := labels.NewRequirement(label.ComponentLabelKey, selection.Equals, []string{
			convertMemberTypeToLabelVal(mt),
		})
		if err != nil {
			return nil, err
		}
		sf.cache[mt] = req
	}
	return sf, nil
}

func MustNewSelectorFactory() *selectorFactory {
	sf, err := NewSelectorFactory()
	if err != nil {
		panic(err)
	}

	return sf
}
