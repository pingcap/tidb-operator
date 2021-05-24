// Copyright 2021 PingCAP, Inc.
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

package feature

import (
	"sort"
	"strings"
)

type FeaturesBuilder interface {
	With(feat string) FeaturesBuilder

	Build() Features
}

type Features interface {
	Has(feat string) bool
	Value(feat string) string

	String() string
}

func New(feats ...string) FeaturesBuilder {
	b := &builder{
		fs: make(map[string]string),
	}
	for _, feat := range feats {
		b.With(feat)
	}
	return b
}

type builder struct {
	fs map[string]string
}

// With add feature string in format: key or key:value.
// Feature with invalid format will be ignored
func (b *builder) With(feat string) FeaturesBuilder {
	kv := strings.Split(feat, ":")
	if len(kv) == 1 {
		key := strings.TrimSpace(kv[0])
		if key == "" {
			return b
		}
		b.fs[key] = ""
	} else if len(kv) == 2 {
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		if key == "" {
			return b
		}
		b.fs[key] = val
	}
	return b
}

func (b *builder) Build() Features {
	boolOrder := []string{}
	kvOrder := []string{}
	for k, v := range b.fs {
		if v == "" {
			boolOrder = append(boolOrder, k)
		} else {
			kvOrder = append(kvOrder, k)
		}
	}
	sort.Strings(boolOrder)
	sort.Strings(kvOrder)
	order := make([]string, 0, len(boolOrder)+len(kvOrder))
	order = append(order, boolOrder...)
	order = append(order, kvOrder...)

	return &features{
		fs:    b.fs,
		order: order,
	}
}

type features struct {
	fs    map[string]string
	order []string
}

func (f *features) Has(feat string) bool {
	_, ok := f.fs[feat]
	return ok
}

func (f *features) Value(feat string) string {
	v, ok := f.fs[feat]
	if !ok {
		return ""
	}
	return v
}

func (f *features) String() string {
	s := strings.Builder{}
	for _, feat := range f.order {
		s.WriteString("[")
		s.WriteString(feat)
		val := f.Value(feat)
		if val != "" {
			s.WriteString(":")
			s.WriteString(val)
		}
		s.WriteString("]")
	}
	return s.String()
}
