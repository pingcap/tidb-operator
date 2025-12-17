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

// Package image is defined to return image of components
package image

import (
	"fmt"

	"github.com/distribution/reference"
)

const (
	PD      Untagged = "pingcap/pd"
	TiDB    Untagged = "pingcap/tidb"
	TiKV    Untagged = "pingcap/tikv"
	TiFlash Untagged = "pingcap/tiflash"
	TiCDC   Untagged = "pingcap/ticdc"
	// TSO also use pd image
	TSO Untagged = "pingcap/pd"
	// Scheduling also use pd image
	Scheduling Untagged = "pingcap/pd"
	// Scheduler also use pd image
	Scheduler Untagged = "pingcap/pd"
	TiProxy   Untagged = "pingcap/tiproxy"

	Helper Tagged = "busybox:1.37.0"
)

var (
	PrestopChecker = Tagged("pingcap/tidb-operator-prestop-checker:latest")
	ResourceSyncer = Tagged("pingcap/tidb-operator-resource-syncer:latest")
)

// Tagged is image with image tag
type Tagged string

// Untagged is image without image tag
type Untagged string

// Image returns the image with the given tag
// Note: img must be validated before calling withVersion
func (t Tagged) Image(img *string) string {
	image := string(t)
	if img != nil {
		image = *img
	}

	return image
}

// String implements flag.Value
func (t *Tagged) String() string {
	return string(*t)
}

// Set implements flag.Value
func (t *Tagged) Set(s string) error {
	*t = Tagged(s)
	return nil
}

// Image returns the image with the given tag and version
// Note: img must be validated before calling withVersion
// TODO(liubo02): validate img and version
func (t Untagged) Image(img *string, version string) string {
	image := string(t)
	if img != nil {
		image = *img
	}
	s, err := withVersion(image, version)
	if err != nil {
		panic(err)
	}

	return s
}

func withVersion(img, version string) (string, error) {
	named, exist, err := validate(img)
	if err != nil {
		return "", err
	}
	if !exist {
		return img, nil
	}

	tagged, err := reference.WithTag(named, version)
	if err != nil {
		return "", fmt.Errorf("cannot override version: %w", err)
	}
	return tagged.String(), nil
}

func Validate(img string) error {
	_, _, err := validate(img)
	return err
}

func validate(img string) (_ reference.Named, isNamed bool, _ error) {
	repo, err := reference.Parse(img)
	if err != nil {
		return nil, false, err
	}

	if _, ok := repo.(reference.Tagged); ok {
		return nil, false, nil
	}

	if _, ok := repo.(reference.Digested); ok {
		return nil, false, nil
	}

	// It should always ok, only digest only ref is not named
	// digest only ref will return before this line
	// NOTE(liubo02): all reference.Parse is named and I cannot construct a digest only ref.
	// See https://github.com/distribution/reference/blob/v0.6.0/reference_test.go#L14
	named := repo.(reference.Named)

	return named, true, nil
}
