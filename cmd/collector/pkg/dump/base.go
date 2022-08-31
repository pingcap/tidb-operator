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

package dump

import (
	"io"
)

// BaseDumper is a base dumper that implements the Dumper interface.
// It's used to host common objects required by all dumpers.
type BaseDumper struct {
	close func() error
}

var _ Dumper = (*BaseDumper)(nil)

func (b *BaseDumper) Open(string) (io.Writer, error) {
	panic("not implemented")
}

func (b *BaseDumper) Close() (closeFn func() error) {
	return b.close
}

// NewBaseDumper returns an instance of the BaseDumper.
func NewBaseDumper(closeFn func() error) *BaseDumper {
	return &BaseDumper{
		close: closeFn,
	}
}
