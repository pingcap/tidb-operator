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

// This package is copied from tikv/pd
package keyspace

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
)

type KeyRange struct {
	Type        string
	StartKeyHex string
	EndKeyHex   string
}

func MakeRegionBound(str string) ([]KeyRange, error) {
	keyspaceID64, err := strconv.ParseUint(str, 10, 32)
	if err != nil {
		return nil, err
	}
	keyspaceID := uint32(keyspaceID64)
	if keyspaceID < DefaultKeyspaceID || keyspaceID > MaxValidKeyspaceID {
		return nil, fmt.Errorf("invalid keyspace id %d. It must be in the range of [%d, %d]",
			keyspaceID, DefaultKeyspaceID, MaxValidKeyspaceID)
	}

	bound := makeRegionBound(keyspaceID)
	return []KeyRange{
		{
			Type:        "raw",
			StartKeyHex: hex.EncodeToString(bound.RawLeftBound),
			EndKeyHex:   hex.EncodeToString(bound.RawRightBound),
		},
		{
			Type:        "txn",
			StartKeyHex: hex.EncodeToString(bound.TxnLeftBound),
			EndKeyHex:   hex.EncodeToString(bound.TxnRightBound),
		},
	}, nil
}

const (
	encGroupSize = 8
	encMarker    = byte(0xFF)
)

const (
	// RawKeyspaceModePrefix is the raw keyspace prefix mode byte.
	RawKeyspaceModePrefix = byte('r')
	// TxnKeyspaceModePrefix is the txn keyspace prefix mode byte.
	TxnKeyspaceModePrefix = byte('x')
	// KeyspacePrefixLen is the raw keyspace prefix length before memcomparable encoding.
	KeyspacePrefixLen = 4

	// DefaultKeyspaceID is the default keyspace ID.
	// Valid keyspace id range is [0, 0xFFFFFF](uint24max, or 16777215)
	// 0 is reserved for default keyspace with the name "DEFAULT", It's initialized
	// when PD bootstrap and reserved for users who haven't been assigned keyspace.
	DefaultKeyspaceID = uint32(0)
	// MaxValidKeyspaceID is the max valid keyspace id.
	// Valid keyspace id range is [0, 0xFFFFFF](uint24max, or 16777215)
	// In kv encode, the first byte is represented by r/x, which means txn kv/raw kv, so there are 24 bits left.
	MaxValidKeyspaceID = uint32(0xFFFFFF)
)

var pads = make([]byte, encGroupSize)

// Key represents high-level Key type.
type Key []byte

func MakeKeyspacePrefix(mode byte, id uint32) []byte {
	prefix := make([]byte, KeyspacePrefixLen)
	binary.BigEndian.PutUint32(prefix, id)
	prefix[0] = mode
	return prefix
}

// RegionBound represents the region boundary of the given keyspace.
// For a keyspace with id ['a', 'b', 'c'], it has four boundaries:
//
//	Lower bound for raw mode: ['r', 'a', 'b', 'c']
//	Upper bound for raw mode: ['r', 'a', 'b', 'c + 1']
//	Lower bound for txn mode: ['x', 'a', 'b', 'c']
//	Upper bound for txn mode: ['x', 'a', 'b', 'c + 1']
//	For the max valid keyspace ID, the upper bound advances the mode byte as an exclusive fencepost.
//
// From which it shares the lower bound with keyspace with id ['a', 'b', 'c-1'].
// And shares upper bound with keyspace with id ['a', 'b', 'c + 1'].
// These repeated bound will not cause any problem, as repetitive bound will be ignored during rangeListBuild,
// but provides guard against hole in keyspace allocations should it occur.
type RegionBound struct {
	RawLeftBound  []byte
	RawRightBound []byte
	TxnLeftBound  []byte
	TxnRightBound []byte
}

// makeRegionBound constructs the correct region boundaries of the given keyspace.
func makeRegionBound(id uint32) *RegionBound {
	rawLeftBound := MakeKeyspacePrefix(RawKeyspaceModePrefix, id)
	rawRightBound := MakeKeyspacePrefix(RawKeyspaceModePrefix, id+1)
	txnLeftBound := MakeKeyspacePrefix(TxnKeyspaceModePrefix, id)
	txnRightBound := MakeKeyspacePrefix(TxnKeyspaceModePrefix, id+1)
	if id == MaxValidKeyspaceID {
		// The right bound is an exclusive fencepost, not a real keyspace prefix.
		rawRightBound = []byte{'s', 0, 0, 0}
		txnRightBound = []byte{'y', 0, 0, 0}
	}
	return &RegionBound{
		RawLeftBound:  EncodeBytes(rawLeftBound),
		RawRightBound: EncodeBytes(rawRightBound),
		TxnLeftBound:  EncodeBytes(txnLeftBound),
		TxnRightBound: EncodeBytes(txnRightBound),
	}
}

// EncodeBytes guarantees the encoded value is in ascending order for comparison,
// encoding with the following rule:
//
//	[group1][marker1]...[groupN][markerN]
//	group is 8 bytes slice which is padding with 0.
//	marker is `0xFF - padding 0 count`
//
// For example:
//
//	[] -> [0, 0, 0, 0, 0, 0, 0, 0, 247]
//	[1, 2, 3] -> [1, 2, 3, 0, 0, 0, 0, 0, 250]
//	[1, 2, 3, 0] -> [1, 2, 3, 0, 0, 0, 0, 0, 251]
//	[1, 2, 3, 4, 5, 6, 7, 8] -> [1, 2, 3, 4, 5, 6, 7, 8, 255, 0, 0, 0, 0, 0, 0, 0, 0, 247]
//
// Refer: https://github.com/facebook/mysql-5.6/wiki/MyRocks-record-format#memcomparable-format
func EncodeBytes(data []byte) Key {
	// Allocate more space to avoid unnecessary slice growing.
	// Assume that the byte slice size is about `(len(data) / encGroupSize + 1) * (encGroupSize + 1)` bytes,
	// that is `(len(data) / 8 + 1) * 9` in our implement.
	dLen := len(data)
	result := make([]byte, 0, (dLen/encGroupSize+1)*(encGroupSize+1))
	for idx := 0; idx <= dLen; idx += encGroupSize {
		remain := dLen - idx
		padCount := 0
		if remain >= encGroupSize {
			result = append(result, data[idx:idx+encGroupSize]...)
		} else {
			padCount = encGroupSize - remain
			result = append(result, data[idx:]...)
			result = append(result, pads[:padCount]...)
		}

		marker := encMarker - byte(padCount)
		result = append(result, marker)
	}
	return result
}
