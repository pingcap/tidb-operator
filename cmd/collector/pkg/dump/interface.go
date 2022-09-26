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

// Dumping is the dump type for the output.
// For now, zip is the only supported encoding type.
type Dumping string

// Dumper is an interface for dumping collected information to an output file.
type Dumper interface {
	// Open returns an io.Writer to a specified path.
	Open(path string) (io.Writer, error)
	// Close closes the dumper.
	Close() (closeFn func() error)
}
