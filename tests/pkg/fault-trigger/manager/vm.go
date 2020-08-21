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

package manager

import (
	"strings"
)

type VMManager interface {
	Name() string
	ListVMs() ([]*VM, error)
	StopVM(*VM) error
	StartVM(*VM) error
}

func stripEmpty(data string) string {
	stripLines := []string{}
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		stripFields := []string{}
		fields := strings.Split(line, " ")
		for _, field := range fields {
			if len(field) > 0 {
				stripFields = append(stripFields, field)
			}
		}
		stripLine := strings.Join(stripFields, " ")
		stripLines = append(stripLines, stripLine)
	}
	return strings.Join(stripLines, "\n")
}
