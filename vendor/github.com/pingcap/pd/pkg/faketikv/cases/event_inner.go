// Copyright 2018 PingCAP, Inc.
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

package cases

// EventInner is a detail template for custom events
type EventInner interface {
	Type() string
}

// WriteFlowOnSpotInner writes bytes in some range
type WriteFlowOnSpotInner struct {
	Step func(tick int64) map[string]int64
}

// Type implements the EventInner interface
func (w *WriteFlowOnSpotInner) Type() string {
	return "write-flow-on-spot"
}

// WriteFlowOnRegionInner writes bytes in some region
type WriteFlowOnRegionInner struct {
	Step func(tick int64) map[uint64]int64
}

// Type implements the EventInner interface
func (w *WriteFlowOnRegionInner) Type() string {
	return "write-flow-on-region"
}

// ReadFlowOnRegionInner reads bytes in some region
type ReadFlowOnRegionInner struct {
	Step func(tick int64) map[uint64]int64
}

// Type implements the EventInner interface
func (w *ReadFlowOnRegionInner) Type() string {
	return "read-flow-on-region"
}

// AddNode adds a node
type AddNode struct {
	Tick   int64
	NodeID uint64
}

// RemoveNode removes a node
type RemoveNode struct {
	Tick   int64
	NodeID uint64
}
