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

package dump

import (
	"archive/zip"
	"io"
	"os"
)

const Zip Dumping = "zip"

type ZipDumper struct {
	*BaseDumper
	writer *zip.Writer
}

var _ Dumper = (*ZipDumper)(nil)

func (z *ZipDumper) Open(path string) (io.Writer, error) {
	iowriter, err := z.writer.Create(path)
	if err != nil {
		return nil, err
	}
	return iowriter, nil
}

// NewZipDumper takes returns a dumper that writes files to input path.
func NewZipDumper(path string) (*ZipDumper, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	writer := zip.NewWriter(file)
	return &ZipDumper{
		BaseDumper: NewBaseDumper(writer.Close),
		writer:     writer,
	}, nil
}
