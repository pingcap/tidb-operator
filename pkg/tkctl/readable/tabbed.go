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

package readable

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// Indent level for PrefixWriter
const (
	LEVEL_0 = iota
	LEVEL_1
	LEVEL_2
	LEVEL_3
)

const (
	indentSpace = "  "
)

// PrefixWriter can write text at various indentation levels
type PrefixWriter interface {
	// Write writes text with the specified indentation level.
	Write(level int, format string, a ...interface{})

	// WriteLine writes line with the specified indentation level.
	WriteLine(level int, format string, a ...interface{})
}

// prefixWriter implements PrefixWriter
type prefixWriter struct {
	out io.Writer
}

var _ PrefixWriter = &prefixWriter{}

// NewPrefixWriter creates a new PrefixWriter.
func NewPrefixWriter(out io.Writer) PrefixWriter {
	return &prefixWriter{out: out}
}

func (pw *prefixWriter) Write(indents int, format string, a ...interface{}) {
	sb := strings.Builder{}
	for i := 0; i < indents; i++ {
		sb.WriteString(indentSpace)
	}
	sb.WriteString(format)
	fmt.Fprintf(pw.out, sb.String(), a...)
}

func (pw *prefixWriter) WriteLine(indents int, format string, a ...interface{}) {
	pw.Write(indents, format+"\n", a...)
}

// TabbedString format the input as tabbed string with pretty indent
func TabbedString(f func(io.Writer) error) (string, error) {
	out := new(tabwriter.Writer)
	buf := &bytes.Buffer{}
	out.Init(buf, 0, 8, 2, ' ', 0)

	err := f(out)
	if err != nil {
		return "", err
	}

	out.Flush()
	str := string(buf.String())
	return str, nil
}
