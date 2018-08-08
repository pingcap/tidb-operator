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

package faketikv

import (
	"github.com/pingcap/pd/pkg/faketikv/cases"
)

// Event that affect the status of the cluster
type Event interface {
	Run(tick int64, cs *ClusterInfo) bool
}

// EventRunner includes all events
type EventRunner struct {
	events []Event
}

// NewEventRunner news a event runner
func NewEventRunner(events []cases.EventInner) *EventRunner {
	er := &EventRunner{events: make([]Event, 0, len(events))}
	for _, e := range events {
		event := parserEvent(e)
		if event != nil {
			er.events = append(er.events, event)
		}
	}
	return er
}

func parserEvent(e cases.EventInner) Event {
	switch v := e.(type) {
	case *cases.WriteFlowOnSpotInner:
		return &WriteFlowOnSpot{in: v}
	case *cases.WriteFlowOnRegionInner:
		return &WriteFlowOnRegion{in: v}
	case *cases.ReadFlowOnRegionInner:
		return &ReadFlowOnRegion{in: v}
	}
	return nil
}

// Tick ticks the event run
func (er *EventRunner) Tick(tick int64, cs *ClusterInfo) {
	var finishedIndex int
	for i, e := range er.events {
		isFinished := e.Run(tick, cs)
		if isFinished {
			er.events[i], er.events[finishedIndex] = er.events[finishedIndex], er.events[i]
			finishedIndex++
		}
	}
	er.events = er.events[finishedIndex:]
}

// WriteFlowOnSpot writes bytes in some range
type WriteFlowOnSpot struct {
	in *cases.WriteFlowOnSpotInner
}

// Run implements the event interface
func (w *WriteFlowOnSpot) Run(tick int64, cs *ClusterInfo) bool {
	res := w.in.Step(tick)
	cs.updateRegionSize(res)
	return false
}

// WriteFlowOnRegion writes bytes in some region
type WriteFlowOnRegion struct {
	in *cases.WriteFlowOnRegionInner
}

// Run implements the event interface
func (w *WriteFlowOnRegion) Run(tick int64, cs *ClusterInfo) bool {
	res := w.in.Step(tick)
	cs.updateRegionWriteBytes(res)
	return false
}

// ReadFlowOnRegion reads bytes in some region
type ReadFlowOnRegion struct {
	in *cases.ReadFlowOnRegionInner
}

// Run implements the event interface
func (w *ReadFlowOnRegion) Run(tick int64, cs *ClusterInfo) bool {
	res := w.in.Step(tick)
	cs.updateRegionReadBytes(res)
	return false
}
