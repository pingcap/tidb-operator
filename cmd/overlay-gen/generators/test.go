// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package generators

import (
	"io"
	"log/slog"
	"strconv"
	"strings"

	"k8s.io/gengo/v2/generator"
	"k8s.io/gengo/v2/namer"
	"k8s.io/gengo/v2/types"
)

type overlayTestGenerator struct {
	generator.GoGenerator
	targetPackage string
	imports       namer.ImportTracker
	funcTracker   map[string]map[string]struct{}
}

func NewOverlayTestGenerator(outputFilename, targetPackage string) generator.Generator {
	g := &overlayTestGenerator{
		GoGenerator: generator.GoGenerator{
			OutputFilename: outputFilename,
		},
		targetPackage: targetPackage,
		imports:       generator.NewImportTrackerForPackage(targetPackage),
		funcTracker:   map[string]map[string]struct{}{},
	}

	g.trackFunc("construct", types.Ref("k8s.io/apimachinery/pkg/api/resource", "Quantity"))
	g.trackFunc("construct", types.Ref("k8s.io/apimachinery/pkg/apis/meta/v1", "ObjectMeta"))
	g.trackFunc("construct", types.Ref("", "map[string]string"))
	return g
}

// Filter returns true if this Generator cares about this type.
// This will be called for every type which made it through this Package's
// Filter method.
func (*overlayTestGenerator) Filter(_ *generator.Context, t *types.Type) bool {
	// We only handle exported structs.
	return t.Name.Name == "PodSpec"
}

// Namers returns a set of NameSystems which will be merged with the namers
// provided when executing this package. In case of a name collision, the
// values produced here will win.
func (g *overlayTestGenerator) Namers(*generator.Context) namer.NameSystems {
	return namer.NameSystems{
		// This elides package names when the name is in "this" package.
		"raw":    namer.NewRawNamer(g.targetPackage, g.imports),
		"public": namer.NewPublicNamer(0),
	}
}

// GenerateType should emit code for the specified type.  This will be called
// for every type which made it through this Generator's Filter method.
func (g *overlayTestGenerator) GenerateType(c *generator.Context, t *types.Type, w io.Writer) error {
	slog.Info("generating overlay test", "type", t.String())

	sw := generator.NewSnippetWriter(w, c, "$", "$")

	p := NewParams(".Spec", t, nil)
	g.generateConstructFunc(sw, p)
	return sw.Error()
}

func (g *overlayTestGenerator) Imports(_ *generator.Context) (imports []string) {
	importLines := []string{}
	importLines = append(importLines, g.imports.ImportLines()...)

	return importLines
}

func (g *overlayTestGenerator) trackFunc(ns string, t *types.Type) (hasAdded bool) {
	tracker, ok := g.funcTracker[ns]
	if !ok {
		tracker = map[string]struct{}{}
		g.funcTracker[ns] = tracker
	}

	if _, ok := tracker[t.Name.String()]; ok {
		return true
	}

	tracker[t.Name.String()] = struct{}{}

	return false
}

func (g *overlayTestGenerator) generateConstructFunc(sw *generator.SnippetWriter, p Params, ignoreKeys ...string) {
	if g.trackFunc("construct"+p.AnnotateType().FuncPrefix()+IgnoreKeys(ignoreKeys...), p.Type()) {
		return
	}
	switch p.Type().Kind {
	case types.Struct:
		g.generateConstructStructFunc(sw, p, ignoreKeys...)
	case types.Slice, types.Array:
		g.generateConstructSliceFunc(sw, p)
	case types.Map:
		g.generateConstructMapFunc(sw, p)
	case types.Pointer:
		g.generateConstructPointerFunc(sw, p)
	case types.Builtin:
		g.generateConstructBuiltinFunc(sw, p)
	case types.Alias:
		g.generateConstructAliasFunc(sw, p)
	default:
		Panic("unhandled func generation", "type", p.Type(), "kind", p.Type().Kind)
	}
}

func isMatched(m *types.Member, fs []StructField) bool {
	if m.Embedded && m.Type.Kind == types.Struct {
		for _, mm := range m.Type.Members {
			if isMatched(&mm, fs) {
				return true
			}
		}
	}
	for _, f := range fs {
		if f.Name == m.Name {
			return true
		}
	}

	return false
}

//nolint:gocyclo // refactor if possible
func (g *overlayTestGenerator) generateConstructStructFunc(sw *generator.SnippetWriter, p Params, ignoreKeys ...string) {
	fields, err := getFieldsByKeys(p.Type(), ignoreKeys...)
	if err != nil {
		Panic("get fields failed", "err", err, "type", p.Type(), "keys", ignoreKeys)
	}
	g.generateConstructFuncName(sw, p.Type(), p.AnnotateType(), ignoreKeys...)
	DoLine(sw, "", "cases := []Case[$.|raw$] {}", p.Type())
	ms := p.Members()
	for index, m := range ms {
		isKey := isMatched(m.Member(), fields)
		policy := "NoLimit"
		if isKey {
			policy = "NoNotEqual | NoZero | NoNil"
		}

		if m.Member().Embedded && isKey {
			g.generateConstructFuncCall(sw, "cs"+strconv.Itoa(index)+" :=", policy, m.Type(), m.AnnotateType(), ignoreKeys...)
		} else {
			g.generateConstructFuncCall(sw, "cs"+strconv.Itoa(index)+" :=", policy, m.Type(), m.AnnotateType())
		}
	}
	DoLine(sw, "", "maxCount := max(", nil)
	for i := range ms {
		DoLine(sw, "", "len(cs$.$),", i)
	}
	DoLine(sw, "", ")", nil)

	for i := range ms {
		DoLine(sw, "", "k$.$ := 0", i)
	}

	DoLine(sw, "", "for i := range maxCount {", nil)
	DoLine(sw, "", "nc := Case[$.|raw$]{}", p.Type())
	for index, m := range ms {
		isKey := isMatched(m.Member(), fields)
		DoLine(sw, "", "if i / len(cs$.$) > k$.$ {", index)
		policy := "NoLimit"
		if isKey {
			policy = "NoNotEqual | NoZero | NoNil"
		}

		if m.Member().Embedded && isKey {
			g.generateConstructFuncCall(sw, "cs"+strconv.Itoa(index)+" =", policy, m.Type(), m.AnnotateType(), ignoreKeys...)
		} else {
			g.generateConstructFuncCall(sw, "cs"+strconv.Itoa(index)+" =", policy, m.Type(), m.AnnotateType())
		}

		DoLine(sw, "", "k$.$ += 1", index)
		DoLine(sw, "", "}", nil)
		DoLine(sw, "", "c$.$ := &cs$.$[i % len(cs$.$)]", index)

		switch p.AnnotateType() {
		case AtomicStruct:
			// all fields will be overrided by src
			DoLine(sw, "", "nc.expected.$.name$ = c$.index$.src", generator.Args{
				"name":  m.Member().Name,
				"index": index,
			})
			DoLine(sw, "", "nc.dst.$.name$ = c$.index$.dst", generator.Args{
				"name":  m.Member().Name,
				"index": index,
			})
			DoLine(sw, "", "nc.src.$.name$ = c$.index$.src", generator.Args{
				"name":  m.Member().Name,
				"index": index,
			})
		case GranularStruct:
			DoLine(sw, "", "nc.expected.$.name$ = c$.index$.expected", generator.Args{
				"name":  m.Member().Name,
				"index": index,
			})
			DoLine(sw, "", "nc.dst.$.name$ = c$.index$.dst", generator.Args{
				"name":  m.Member().Name,
				"index": index,
			})
			DoLine(sw, "", "nc.src.$.name$ = c$.index$.src", generator.Args{
				"name":  m.Member().Name,
				"index": index,
			})
		}
	}
	DoLine(sw, "", "cases = append(cases, nc)", nil)
	DoLine(sw, "", "}", nil)

	DoLine(sw, "", "return cases", nil)
	DoLine(sw, "", "}", nil)

	for _, m := range ms {
		isKey := isMatched(m.Member(), fields)
		if m.Member().Embedded && isKey {
			g.generateConstructFunc(sw, m, ignoreKeys...)
		} else {
			g.generateConstructFunc(sw, m)
		}
	}
}

func IgnoreKeys(keys ...string) string {
	key := ""
	if len(keys) != 0 {
		key = "Ignore_" + strings.Join(keys, "_")
	}

	return key
}

func (g *overlayTestGenerator) generateConstructSliceFunc(sw *generator.SnippetWriter, p Params) {
	g.generateConstructFuncName(sw, p.Type(), p.AnnotateType())
	elem := p.Slice("", "")
	DoLine(sw, "", "cases := []Case[$.|raw$] {", p.Type())
	DoLine(sw, "", "{", nil)
	DoLine(sw, "", "expected: nil,", nil)
	DoLine(sw, "", "dst: nil,", nil)
	DoLine(sw, "", "src: nil,", nil)
	DoLine(sw, "", "},", nil)
	DoLine(sw, "", "}", nil)
	g.generateConstructFuncCall(sw, "cs :=", "NoLimit", elem.Type(), elem.AnnotateType(), p.Keys()...)
	DoLine(sw, "", "var nc Case[$.|raw$]", p.Type())

	switch p.AnnotateType() {
	case AtomicList:
		// always use src
		DoLine(sw, "", "nc = Case[$.|raw$]{}", p.Type())
		DoLine(sw, "", "for _, c := range cs {", nil)
		DoLine(sw, "", "nc.expected = append(nc.expected, c.src)", nil)
		DoLine(sw, "", "nc.dst = append(nc.dst, c.dst)", nil)
		DoLine(sw, "", "nc.src = append(nc.src, c.src)", nil)
		DoLine(sw, "", "}", nil)
		DoLine(sw, "", "cases = append(cases, nc)", nil)

		// no overlay if src is empty
		DoLine(sw, "", "nc = Case[$.|raw$]{}", p.Type())
		DoLine(sw, "", "for _, c := range cs {", nil)
		DoLine(sw, "", "nc.expected = append(nc.expected, c.dst)", nil)
		DoLine(sw, "", "nc.dst = append(nc.dst, c.dst)", nil)
		DoLine(sw, "", "}", nil)
		DoLine(sw, "", "nc.src = $.|raw${}", p.Type())
		DoLine(sw, "", "cases = append(cases, nc)", nil)

	case MapList:
		// all kesy in src are in dst
		// all kesy in dst are also in src
		DoLine(sw, "", "nc = Case[$.|raw$]{}", p.Type())
		DoLine(sw, "", "for _, c := range cs {", nil)
		DoLine(sw, "", "nc.expected = append(nc.expected, c.expected)", nil)
		DoLine(sw, "", "nc.dst = append(nc.dst, c.dst)", nil)
		DoLine(sw, "", "nc.src = append(nc.src, c.src)", nil)
		DoLine(sw, "", "}", nil)
		DoLine(sw, "", "cases = append(cases, nc)", nil)

		// some keys in src are in dst, some keys are not
		// some keys in dst are also not in src
		DoLine(sw, "", "nc = Case[$.|raw$]{}", p.Type())
		DoLine(sw, "", "srcs := $.|raw${}", p.Type())
		DoLine(sw, "", "for i, c := range cs {", nil)
		DoLine(sw, "", "switch i % 3 {", nil)
		DoLine(sw, "", "case 0:", nil)
		DoLine(sw, "", "nc.expected = append(nc.expected, c.expected)", nil)
		DoLine(sw, "", "nc.dst = append(nc.dst, c.dst)", nil)
		DoLine(sw, "", "nc.src = append(nc.src, c.src)", nil)
		DoLine(sw, "", "case 1:", nil)
		DoLine(sw, "", "nc.expected = append(nc.expected, c.dst)", nil)
		DoLine(sw, "", "nc.dst = append(nc.dst, c.dst)", nil)
		DoLine(sw, "", "case 2:", nil)
		DoLine(sw, "", "srcs = append(srcs, c.src)", nil)
		DoLine(sw, "", "nc.src = append(nc.src, c.src)", nil)
		DoLine(sw, "", "}", nil)
		DoLine(sw, "", "}", nil)
		DoLine(sw, "", "nc.expected = append(nc.expected, srcs...)", nil)
		DoLine(sw, "", "cases = append(cases, nc)", nil)

		// all keys in dst are not in src
		// all keys in src are also not in dst
		DoLine(sw, "", "nc = Case[$.|raw$]{}", p.Type())
		DoLine(sw, "", "srcs = $.|raw${}", p.Type())
		DoLine(sw, "", "for i, c := range cs {", nil)
		DoLine(sw, "", "switch i % 2 {", nil)
		DoLine(sw, "", "case 0:", nil)
		DoLine(sw, "", "nc.expected = append(nc.expected, c.dst)", nil)
		DoLine(sw, "", "nc.dst = append(nc.dst, c.dst)", nil)
		DoLine(sw, "", "case 1:", nil)
		DoLine(sw, "", "srcs = append(srcs, c.src)", nil)
		DoLine(sw, "", "nc.src = append(nc.src, c.src)", nil)
		DoLine(sw, "", "}", nil)
		DoLine(sw, "", "}", nil)
		DoLine(sw, "", "nc.expected = append(nc.expected, srcs...)", nil)
		DoLine(sw, "", "cases = append(cases, nc)", nil)

		// dst is empty
		DoLine(sw, "", "nc = Case[$.|raw$]{}", p.Type())
		DoLine(sw, "", "for _, c := range cs {", nil)
		DoLine(sw, "", "nc.expected = append(nc.expected, c.src)", nil)
		DoLine(sw, "", "nc.src = append(nc.src, c.src)", nil)
		DoLine(sw, "", "}", nil)
		DoLine(sw, "", "cases = append(cases, nc)", nil)

		// src is empty
		DoLine(sw, "", "nc = Case[$.|raw$]{}", p.Type())
		DoLine(sw, "", "for _, c := range cs {", nil)
		DoLine(sw, "", "nc.expected = append(nc.expected, c.dst)", nil)
		DoLine(sw, "", "nc.dst = append(nc.dst, c.dst)", nil)
		DoLine(sw, "", "}", nil)
		DoLine(sw, "", "cases = append(cases, nc)", nil)
	}
	DoLine(sw, "", "return cases", nil)
	DoLine(sw, "", "}", nil)

	g.generateConstructFunc(sw, elem, p.Keys()...)
}

func (*overlayTestGenerator) generateConstructFuncCall(sw *generator.SnippetWriter,
	prefix, policy string, t *types.Type, at AnnotateType, keys ...string) {
	DoLine(sw, "", "$.prefix$ construct$.annoType$$.type|public$$.key$($.policy$)", generator.Args{
		"prefix":   prefix,
		"policy":   policy,
		"type":     t,
		"annoType": at.FuncPrefix(),
		"key":      IgnoreKeys(keys...),
	})
}

func (*overlayTestGenerator) generateConstructFuncName(sw *generator.SnippetWriter, t *types.Type, at AnnotateType, keys ...string) {
	DoLine(sw, "", "func construct$.annoType$$.type|public$$.key$(p Policy) []Case[$.type|raw$] {", generator.Args{
		"type":     t,
		"annoType": at.FuncPrefix(),
		"key":      IgnoreKeys(keys...),
	})
}

func (g *overlayTestGenerator) generateConstructMapFunc(sw *generator.SnippetWriter, p Params) {
	g.generateConstructFuncName(sw, p.Type(), p.AnnotateType())

	elem := p.Map()
	DoLine(sw, "", "cases := []Case[$.|raw$] {", p.Type())
	DoLine(sw, "", "{", nil)
	DoLine(sw, "", "expected: nil,", nil)
	DoLine(sw, "", "dst: nil,", nil)
	DoLine(sw, "", "src: nil,", nil)
	DoLine(sw, "", "},", nil)
	DoLine(sw, "", "}", nil)
	g.generateConstructFuncCall(sw, "keys :=", "NoNil | NoZero | NoNotEqual", p.Type().Key, None)
	g.generateConstructFuncCall(sw, "vals :=", "NoLimit", elem.Type(), elem.AnnotateType())
	DoLine(sw, "", "keyIndex := 0", nil)
	DoLine(sw, "", "var nc Case[$.|raw$]", p.Type())

	switch p.AnnotateType() {
	case AtomicMap:
		// all keys in src are in dst
		DoLine(sw, "", "for _, val := range vals {", nil)
		DoLine(sw, "", "keyIndex += 1", nil)
		DoLine(sw, "", "if keyIndex >= len(keys) {", nil)
		g.generateConstructFuncCall(sw, "keys =", "NoNil | NoZero | NoNotEqual", p.Type().Key, None)
		DoLine(sw, "", "keyIndex = 0", nil)
		DoLine(sw, "", "}", nil)
		DoLine(sw, "", "key := keys[keyIndex]", nil)
		DoLine(sw, "", "nc = Case[$.|raw$]{}", p.Type())
		DoLine(sw, "", "nc.expected = make($.|raw$)", p.Type())
		DoLine(sw, "", "nc.dst = make($.|raw$)", p.Type())
		DoLine(sw, "", "nc.src = make($.|raw$)", p.Type())
		DoLine(sw, "", "nc.expected[key.expected] = val.src", nil)
		DoLine(sw, "", "nc.dst[key.expected] = val.dst", nil)
		DoLine(sw, "", "nc.src[key.expected] = val.src", nil)
		DoLine(sw, "", "}", nil)
		DoLine(sw, "", "cases = append(cases, nc)", nil)

		// src is empty
		DoLine(sw, "", "for _, val := range vals {", nil)
		DoLine(sw, "", "keyIndex += 1", nil)
		DoLine(sw, "", "if keyIndex >= len(keys) {", nil)
		g.generateConstructFuncCall(sw, "keys =", "NoNil | NoZero | NoNotEqual", p.Type().Key, None)
		DoLine(sw, "", "keyIndex = 0", nil)
		DoLine(sw, "", "}", nil)
		DoLine(sw, "", "key := keys[keyIndex]", nil)
		DoLine(sw, "", "nc = Case[$.|raw$]{}", p.Type())
		DoLine(sw, "", "nc.expected = make($.|raw$)", p.Type())
		DoLine(sw, "", "nc.dst = make($.|raw$)", p.Type())
		DoLine(sw, "", "nc.src = make($.|raw$)", p.Type())
		DoLine(sw, "", "nc.expected[key.expected] = val.dst", nil)
		DoLine(sw, "", "nc.dst[key.expected] = val.dst", nil)
		DoLine(sw, "", "}", nil)
		DoLine(sw, "", "cases = append(cases, nc)", nil)

	case GranularMap:
		// all keys in src are in dst
		// all keys in dst are also in src
		DoLine(sw, "", "for _, val := range vals {", nil)
		DoLine(sw, "", "keyIndex += 1", nil)
		DoLine(sw, "", "if keyIndex >= len(keys) {", nil)
		g.generateConstructFuncCall(sw, "keys =", "NoNil | NoZero | NoNotEqual", p.Type().Key, None)
		DoLine(sw, "", "keyIndex = 0", nil)
		DoLine(sw, "", "}", nil)
		DoLine(sw, "", "key := keys[keyIndex]", nil)
		DoLine(sw, "", "nc = Case[$.|raw$]{}", p.Type())
		DoLine(sw, "", "nc.expected = make($.|raw$)", p.Type())
		DoLine(sw, "", "nc.dst = make($.|raw$)", p.Type())
		DoLine(sw, "", "nc.src = make($.|raw$)", p.Type())
		DoLine(sw, "", "nc.expected[key.expected] = val.expected", nil)
		DoLine(sw, "", "nc.dst[key.expected] = val.dst", nil)
		DoLine(sw, "", "nc.src[key.expected] = val.src", nil)
		DoLine(sw, "", "}", nil)
		DoLine(sw, "", "cases = append(cases, nc)", nil)

		// some keys in src are in dst, some keys are not
		// some keys in dst are also not in src
		DoLine(sw, "", "for i, val := range vals {", nil)
		DoLine(sw, "", "keyIndex += 1", nil)
		DoLine(sw, "", "if keyIndex >= len(keys) {", nil)
		g.generateConstructFuncCall(sw, "keys =", "NoNil | NoZero | NoNotEqual", p.Type().Key, None)
		DoLine(sw, "", "keyIndex = 0", nil)
		DoLine(sw, "", "}", nil)
		DoLine(sw, "", "key := keys[keyIndex]", nil)
		DoLine(sw, "", "nc = Case[$.|raw$]{}", p.Type())
		DoLine(sw, "", "nc.expected = make($.|raw$)", p.Type())
		DoLine(sw, "", "nc.dst = make($.|raw$)", p.Type())
		DoLine(sw, "", "nc.src = make($.|raw$)", p.Type())
		DoLine(sw, "", "switch i % 3 {", nil)
		DoLine(sw, "", "case 0:", nil)
		DoLine(sw, "", "nc.expected[key.expected] = val.expected", nil)
		DoLine(sw, "", "nc.dst[key.expected] = val.dst", nil)
		DoLine(sw, "", "nc.src[key.expected] = val.src", nil)
		DoLine(sw, "", "case 1:", nil)
		DoLine(sw, "", "nc.expected[key.expected] = val.dst", nil)
		DoLine(sw, "", "nc.dst[key.expected] = val.dst", nil)
		DoLine(sw, "", "case 2:", nil)
		DoLine(sw, "", "nc.expected[key.expected] = val.src", nil)
		DoLine(sw, "", "nc.src[key.expected] = val.src", nil)
		DoLine(sw, "", "}", nil)
		DoLine(sw, "", "}", nil)
		DoLine(sw, "", "cases = append(cases, nc)", nil)

		// all keys in dst are not in src
		// all keys in src are also not in dst
		DoLine(sw, "", "for i, val := range vals {", nil)
		DoLine(sw, "", "keyIndex += 1", nil)
		DoLine(sw, "", "if keyIndex >= len(keys) {", nil)
		g.generateConstructFuncCall(sw, "keys =", "NoNil | NoZero | NoNotEqual", p.Type().Key, None)
		DoLine(sw, "", "keyIndex = 0", nil)
		DoLine(sw, "", "}", nil)
		DoLine(sw, "", "key := keys[keyIndex]", nil)
		DoLine(sw, "", "nc = Case[$.|raw$]{}", p.Type())
		DoLine(sw, "", "nc.expected = make($.|raw$)", p.Type())
		DoLine(sw, "", "nc.dst = make($.|raw$)", p.Type())
		DoLine(sw, "", "nc.src = make($.|raw$)", p.Type())
		DoLine(sw, "", "switch i % 2 {", nil)
		DoLine(sw, "", "case 0:", nil)
		DoLine(sw, "", "nc.expected[key.expected] = val.dst", nil)
		DoLine(sw, "", "nc.dst[key.expected] = val.dst", nil)
		DoLine(sw, "", "case 1:", nil)
		DoLine(sw, "", "nc.expected[key.expected] = val.src", nil)
		DoLine(sw, "", "nc.src[key.expected] = val.src", nil)
		DoLine(sw, "", "}", nil)
		DoLine(sw, "", "}", nil)
		DoLine(sw, "", "cases = append(cases, nc)", nil)

		// dst is empty
		DoLine(sw, "", "for _, val := range vals {", nil)
		DoLine(sw, "", "keyIndex += 1", nil)
		DoLine(sw, "", "if keyIndex >= len(keys) {", nil)
		g.generateConstructFuncCall(sw, "keys =", "NoNil | NoZero | NoNotEqual", p.Type().Key, None)
		DoLine(sw, "", "keyIndex = 0", nil)
		DoLine(sw, "", "}", nil)
		DoLine(sw, "", "key := keys[keyIndex]", nil)
		DoLine(sw, "", "nc = Case[$.|raw$]{}", p.Type())
		DoLine(sw, "", "nc.expected = make($.|raw$)", p.Type())
		DoLine(sw, "", "nc.dst = make($.|raw$)", p.Type())
		DoLine(sw, "", "nc.src = make($.|raw$)", p.Type())
		DoLine(sw, "", "nc.expected[key.expected] = val.src", nil)
		DoLine(sw, "", "nc.src[key.expected] = val.src", nil)
		DoLine(sw, "", "}", nil)
		DoLine(sw, "", "cases = append(cases, nc)", nil)

		// src is empty
		DoLine(sw, "", "for _, val := range vals {", nil)
		DoLine(sw, "", "keyIndex += 1", nil)
		DoLine(sw, "", "if keyIndex >= len(keys) {", nil)
		g.generateConstructFuncCall(sw, "keys =", "NoNil | NoZero | NoNotEqual", p.Type().Key, None)
		DoLine(sw, "", "keyIndex = 0", nil)
		DoLine(sw, "", "}", nil)
		DoLine(sw, "", "key := keys[keyIndex]", nil)
		DoLine(sw, "", "nc = Case[$.|raw$]{}", p.Type())
		DoLine(sw, "", "nc.expected = make($.|raw$)", p.Type())
		DoLine(sw, "", "nc.dst = make($.|raw$)", p.Type())
		DoLine(sw, "", "nc.src = make($.|raw$)", p.Type())
		DoLine(sw, "", "nc.expected[key.expected] = val.dst", nil)
		DoLine(sw, "", "nc.dst[key.expected] = val.dst", nil)
		DoLine(sw, "", "}", nil)
		DoLine(sw, "", "cases = append(cases, nc)", nil)
	}
	DoLine(sw, "", "return cases", nil)
	DoLine(sw, "", "}", nil)

	g.generateConstructFunc(sw, elem)
}

func (g *overlayTestGenerator) generateConstructPointerFunc(sw *generator.SnippetWriter, p Params) {
	g.generateConstructFuncName(sw, p.Type(), p.AnnotateType())

	elem := p.Pointer()
	DoLine(sw, "", "cases := []Case[$.|raw$] {", p.Type())
	DoLine(sw, "", "{", nil)
	DoLine(sw, "", "expected: nil,", nil)
	DoLine(sw, "", "dst: nil,", nil)
	DoLine(sw, "", "src: nil,", nil)
	DoLine(sw, "", "},", nil)
	DoLine(sw, "", "}", nil)

	g.generateConstructFuncCall(sw, "cs :=", "p", elem.Type(), elem.AnnotateType())

	DoLine(sw, "", "for _, c := range cs {", nil)
	DoLine(sw, "", "cases = append(cases, Case[$.|raw$] {", p.Type())
	DoLine(sw, "", "expected: &c.expected,", nil)
	DoLine(sw, "", "dst: &c.dst,", nil)
	DoLine(sw, "", "src: &c.src,", nil)
	DoLine(sw, "", "})", nil)
	DoLine(sw, "", "}", nil)
	DoLine(sw, "", "return cases", nil)
	DoLine(sw, "", "}", nil)

	g.generateConstructFunc(sw, elem)
}

//nolint:gocyclo // refactor if possible
func (g *overlayTestGenerator) generateConstructBuiltinFunc(sw *generator.SnippetWriter, p Params) {
	g.generateConstructFuncName(sw, p.Type(), p.AnnotateType())

	DoLine(sw, "", "cases := []Case[$.|raw$] {}", p.Type())
	t := p.Type()
	// dst==src && contains 0
	DoLine(sw, "", "if p&(NoZero) == 0 {", nil)
	switch t {
	case types.Int, types.Int64, types.Int32, types.Int16:
		DoLine(sw, "", `cases = append(cases, Case[$.|raw$]{expected: 0, dst: 0, src: 0})`, t)
	case types.Uint, types.Uint64, types.Uint32, types.Uint16, types.Byte:
	case types.Float, types.Float32, types.Float64:
	case types.String:
		DoLine(sw, "", `cases = append(cases, Case[$.|raw$]{expected: "", dst: "", src: ""})`, t)
	case types.Bool:
		DoLine(sw, "", `cases = append(cases, Case[$.|raw$]{expected: false, dst: false, src: false})`, t)
	}
	DoLine(sw, "", "}", nil)

	// dst!=src && no 0
	DoLine(sw, "", "if p&(NoNotEqual) == 0 {", nil)
	switch t {
	case types.Int, types.Int64, types.Int32, types.Int16:
		DoLine(sw, "", "if p&(NoNotEqual) == 0 {", nil)
		DoLine(sw, "", `cases = append(cases, Case[$.|raw$]{expected: 1, dst: 2, src: 1})`, t)
		DoLine(sw, "", `cases = append(cases, Case[$.|raw$]{expected: -1, dst: 2, src: -1})`, t)
		DoLine(sw, "", `cases = append(cases, Case[$.|raw$]{expected: 1, dst: -2, src: 1})`, t)
		DoLine(sw, "", `cases = append(cases, Case[$.|raw$]{expected: -1, dst: -2, src: -1})`, t)
		DoLine(sw, "", "}", nil)

	case types.Uint, types.Uint64, types.Uint32, types.Uint16, types.Byte:
	case types.Float, types.Float32, types.Float64:
	case types.String:
		DoLine(sw, "", `dst, src := randString(), randString()`, nil)
		DoLine(sw, "", `cases = append(cases, Case[$.|raw$]{expected: src, dst: dst, src: src})`, t)
	case types.Bool:
	}
	DoLine(sw, "", "}", nil)

	// dst!=src && contains 0
	DoLine(sw, "", "if p&(NoZero|NoNotEqual) == 0 {", nil)
	switch t {
	case types.Int, types.Int64, types.Int32, types.Int16:
		DoLine(sw, "", `cases = append(cases, Case[$.|raw$]{expected: 1, dst: 1, src: 0})`, t)
		DoLine(sw, "", `cases = append(cases, Case[$.|raw$]{expected: -1, dst: -1, src: 0})`, t)
		DoLine(sw, "", `cases = append(cases, Case[$.|raw$]{expected: 1, dst: 0, src: 1})`, t)
		DoLine(sw, "", `cases = append(cases, Case[$.|raw$]{expected: -1, dst: 0, src: -1})`, t)
	case types.Uint, types.Uint64, types.Uint32, types.Uint16, types.Byte:
	case types.Float, types.Float32, types.Float64:
	case types.String:
		DoLine(sw, "", `dst, src := randString(), randString()`, nil)
		DoLine(sw, "", `cases = append(cases, Case[$.|raw$]{expected: dst, dst: dst, src: ""})`, t)
		DoLine(sw, "", `cases = append(cases, Case[$.|raw$]{expected: src, dst: "", src: src})`, t)
	case types.Bool:
	}
	DoLine(sw, "", "}", nil)

	// dst==src && no 0
	switch t {
	case types.Int, types.Int64, types.Int32, types.Int16:
		DoLine(sw, "", `cases = append(cases, Case[$.|raw$]{expected: 1, dst: 1, src: 1})`, t)
		DoLine(sw, "", `cases = append(cases, Case[$.|raw$]{expected: -1, dst: -1, src: -1})`, t)

	case types.Uint, types.Uint64, types.Uint32, types.Uint16, types.Byte:
		DoLine(sw, "", `cases = append(cases, Case[$.|raw$]{expected: 1, dst: 1, src: 1})`, t)

	case types.Float, types.Float32, types.Float64:
		DoLine(sw, "", `cases = append(cases, Case[$.|raw$]{expected: 1.0, dst: 1.0, src: 1.0})`, t)
		DoLine(sw, "", `cases = append(cases, Case[$.|raw$]{expected: -1.0, dst: -1.0, src: -1.0})`, t)

	case types.String:
		DoLine(sw, "", `var val string`, nil)
		for range 3 {
			DoLine(sw, "", `val = randString()`, nil)
			DoLine(sw, "", `cases = append(cases, Case[$.|raw$]{expected: val, dst: val, src: val})`, t)
		}

	case types.Bool:
		DoLine(sw, "", `cases = append(cases, Case[$.|raw$]{expected: false, dst: false, src: false})`, t)
		DoLine(sw, "", `cases = append(cases, Case[$.|raw$]{expected: true, dst: true, src: true})`, t)
	}
	DoLine(sw, "", "return cases", nil)
	DoLine(sw, "", "}", nil)
}

func (g *overlayTestGenerator) generateConstructAliasFunc(sw *generator.SnippetWriter, p Params) {
	g.generateConstructFuncName(sw, p.Type(), p.AnnotateType())

	underlying := p.Underlying()
	DoLine(sw, "", "cases := []Case[$.|raw$] {}", p.Type())
	g.generateConstructFuncCall(sw, "cs :=", "p", underlying.Type(), underlying.AnnotateType())
	DoLine(sw, "", "for _, c := range cs {", nil)
	DoLine(sw, "", "cases = append(cases, Case[$.|raw$] {", p.Type())
	DoLine(sw, "", "expected: $.|raw$(c.expected),", p.Type())
	DoLine(sw, "", "dst: $.|raw$(c.dst),", p.Type())
	DoLine(sw, "", "src: $.|raw$(c.src),", p.Type())
	DoLine(sw, "", "})", nil)
	DoLine(sw, "", "}", nil)
	DoLine(sw, "", "return cases", nil)
	DoLine(sw, "", "}", nil)

	g.generateConstructFunc(sw, p.Underlying())
}
