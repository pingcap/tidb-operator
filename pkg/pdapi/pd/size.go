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

// This file is copied from https://github.com/tikv/pd/blob/v8.1.0/pkg/utils/typeutil/size.go

package pd

import (
	"strconv"

	"github.com/docker/go-units"
)

// ByteSize is a retype uint64 for TOML and JSON.
type ByteSize uint64

// ParseMBFromText parses MB from text.
func ParseMBFromText(text string, value uint64) uint64 {
	b := ByteSize(0)
	err := b.UnmarshalText([]byte(text))
	if err != nil {
		return value
	}
	return uint64(b / units.MiB)
}

// MarshalJSON returns the size as a JSON string.
func (b ByteSize) MarshalJSON() ([]byte, error) {
	return []byte(`"` + units.BytesSize(float64(b)) + `"`), nil
}

// UnmarshalJSON parses a JSON string into the byte size.
func (b *ByteSize) UnmarshalJSON(text []byte) error {
	s, err := strconv.Unquote(string(text))
	if err != nil {
		return err
	}
	v, err := units.RAMInBytes(s)
	if err != nil {
		return err
	}
	//nolint:gosec // expected type conversion
	*b = ByteSize(v)
	return nil
}

// UnmarshalText parses a Toml string into the byte size.
func (b *ByteSize) UnmarshalText(text []byte) error {
	v, err := units.RAMInBytes(string(text))
	if err != nil {
		return err
	}
	*b = ByteSize(v) //nolint:gosec // expected type conversion
	return nil
}
