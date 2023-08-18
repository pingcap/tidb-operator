// Copyright 2023 PingCAP, Inc.
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

package tasks

import (
	"fmt"
	"os"
)

type ReadType interface {
	fmt.Stringer
}

type ReadOffsetAndLength struct {
	Offset int64
	Length int64
}

func (r ReadOffsetAndLength) String() string {
	return fmt.Sprintf("ReadOffsetAndLength(%d, %d)", r.Offset, r.Length)
}

type ReadLastNBytes int

func (r ReadLastNBytes) String() string {
	return fmt.Sprintf("ReadLastNBytes(%d)", r)
}

type ReadFull struct{}

func (r ReadFull) String() string {
	return "ReadFull"
}

type Sync struct {
	C chan<- struct{}
}

func (r Sync) String() string {
	return "Sync"
}

type ReadFile struct {
	Type     ReadType
	File     os.FileInfo
	FilePath string
	Direct   bool
}

func (r ReadFile) String() string {
	return fmt.Sprintf("ReadFile(%q, %s)", r.FilePath, r.Type)
}
