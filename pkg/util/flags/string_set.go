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

package flags

import (
	"bytes"
	"encoding/csv"
	"flag"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
)

type stringSetValue struct {
	value   *sets.String
	changed bool
}

// NewStringSetValue returns a flag.Value which holds a list of features
func NewStringSetValue(val sets.String, p *sets.String) flag.Value {
	ssv := new(stringSetValue)
	ssv.value = p
	*ssv.value = val
	return ssv
}

func (s *stringSetValue) Set(val string) error {
	v, err := readAsCSV(val)
	if err != nil {
		return err
	}
	newSet := sets.NewString(v...)
	if !s.changed {
		*s.value = newSet
	} else {
		for key := range newSet {
			(*s.value).Insert(key)
		}
	}
	s.changed = true
	return nil
}

func (s *stringSetValue) String() string {
	if s.value == nil {
		return ""
	}
	v, _ := writeAsCSV((*s.value).List())
	return v
}

func readAsCSV(val string) ([]string, error) {
	if val == "" {
		return []string{}, nil
	}
	stringReader := strings.NewReader(val)
	csvReader := csv.NewReader(stringReader)
	return csvReader.Read()
}

func writeAsCSV(vals []string) (string, error) {
	b := &bytes.Buffer{}
	w := csv.NewWriter(b)
	err := w.Write(vals)
	if err != nil {
		return "", err
	}
	w.Flush()
	return strings.TrimSuffix(b.String(), "\n"), nil
}
