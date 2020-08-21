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

package streams

import (
	"github.com/docker/docker/pkg/term"
)

// commonStream is an input stream used by the Cli tool to read user input
type commonStream struct {
	fd         uintptr
	isTerminal bool
	state      *term.State
}

// FD returns the file descriptor number for this stream
func (s *commonStream) FD() uintptr {
	return s.fd
}

// IsTerminal returns true if this stream is connected to a terminal
func (s *commonStream) IsTerminal() bool {
	return s.isTerminal
}

// RestoreTerminal restores normal mode to the terminal
func (s *commonStream) RestoreTerminal() {
	if s.state != nil {
		term.RestoreTerminal(s.fd, s.state)
	}
}

// SetIsTerminal sets the boolean used for isTerminal
func (s *commonStream) SetIsTerminal(isTerminal bool) {
	s.isTerminal = isTerminal
}
