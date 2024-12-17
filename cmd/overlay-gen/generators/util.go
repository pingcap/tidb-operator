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
	"context"
	"log/slog"
	"os"
	"runtime"
	"time"

	"k8s.io/gengo/v2"
	"k8s.io/gengo/v2/generator"
	"k8s.io/gengo/v2/types"
)

func DoLine(sw *generator.SnippetWriter, indent, format string, args any) {
	for range indent {
		sw.Do("\t", nil)
	}
	sw.Do(format, args)
	sw.Do("\n", nil)
}

// Params defines the messages of overlay params
type Params interface {
	Name() string
	// Type returns the type dst and src
	Type() *types.Type
	// Dst returns the variable name of dst
	Dst() string
	// Src returns the variable name of src
	Src() string

	// AnnotateType returns the type from comments
	AnnotateType() AnnotateType
	// Keys returns additional keys of specified annotate type
	Keys() []string

	// some types need check nil before get address
	CheckNil()

	Member() *types.Member

	Members() []Params
	Pointer() Params
	Slice(dst, src string) Params
	Map() Params
	Underlying() Params
}

type Refer interface {
	Ref(root string) string
}

// for slice, array, map
type indexRefer struct {
	index string
}

func (r *indexRefer) Ref(root string) string {
	return "(" + root + ")[" + r.index + "]"
}

// for struct
type fieldRefer struct {
	field string
}

func (r *fieldRefer) Ref(root string) string {
	return root + "." + r.field
}

type pointerRefer struct{}

func (*pointerRefer) Ref(root string) string {
	return "*" + root
}

type addrRefer struct {
	notNil *bool
}

func (r *addrRefer) Ref(root string) string {
	if *r.notNil {
		return "&(" + root + ")"
	}
	return root
}

type prefixRefer struct {
	prefix string
}

func (r *prefixRefer) Ref(root string) string {
	return r.prefix + root
}

func Index(i string) Refer {
	return &indexRefer{
		index: i,
	}
}

func Field(f string) Refer {
	return &fieldRefer{
		field: f,
	}
}

func Pointer() Refer {
	return &pointerRefer{}
}

func Addr(notNil *bool) Refer {
	return &addrRefer{
		notNil: notNil,
	}
}

func Prefix(p string) Refer {
	return &prefixRefer{prefix: p}
}

type AnnotateType int

const (
	None AnnotateType = iota
	AtomicList
	SetList
	MapList

	AtomicMap
	GranularMap

	AtomicStruct
	GranularStruct
)

func (at AnnotateType) FuncPrefix() string {
	switch at {
	case AtomicList, AtomicMap, AtomicStruct:
		return "Atomic"
	case SetList:
		return "Set"
	case MapList:
		return "Map"
	case GranularMap:
		return ""
	case GranularStruct:
		return ""
	}
	return ""
}

type params struct {
	t      *types.Type
	refers []Refer
	// if srcRefers is empty, use refers to generate src
	srcRefers []Refer

	// name is field name from root
	name string

	// means nil has been checked, addr can be called
	checkNil bool

	m *types.Member

	annotateType AnnotateType
	keys         []string
}

// TODO: handle panic
func NewParams(name string, t *types.Type, comments []string) Params {
	return newParams(name, t, comments)
}

func newParams(name string, t *types.Type, comments []string) *params {
	p := parseComment(t, comments)
	p.t = t
	p.name = name
	return p
}

const (
	typeAtomic   = "atomic"
	typeMap      = "map"
	typeGranular = "granular"
	typeSet      = "set"
)

//nolint:gocyclo // refactor if possible
func parseComment(t *types.Type, comments []string) *params {
	p := &params{}
	switch t.Kind {
	case types.Alias:
		return parseComment(t.Underlying, comments)
	case types.Pointer:
		return parseComment(t.Elem, comments)
	case types.Map:
		tags := gengo.ExtractCommentTags("+", comments)
		ts := tags["mapType"]
		if len(ts) > 1 {
			panic("value of map type cannot be set twice")
		}

		if len(ts) == 0 {
			p.annotateType = GranularMap
			return p
		}

		switch ts[0] {
		case typeAtomic:
			p.annotateType = AtomicMap
		case typeGranular:
			p.annotateType = GranularMap
		default:
			panic("unknown map type " + ts[0])
		}
		return p

	case types.Slice, types.Array:
		// ignore []byte
		if t.Elem == types.Byte {
			p.annotateType = AtomicList
			return p
		}
		tags := gengo.ExtractCommentTags("+", comments)
		ts := tags["listType"]
		if len(ts) != 1 {
			panic("value of list type must be set exactly once")
		}

		switch ts[0] {
		case typeAtomic:
			p.annotateType = AtomicList
		case typeMap:
			p.annotateType = MapList
			p.keys = tags["listMapKey"]
		case typeSet:
			p.annotateType = SetList
		default:
			panic("unknown list type " + ts[0])
		}

		return p
	case types.Struct:
		sp := parseCommentForStruct(comments)
		if sp == nil {
			// get tag from struct's comment
			sp = parseCommentForStruct(t.CommentLines)
			if sp != nil {
				return sp
			}
		}

		// cannot get tag from both member comments and struct comments
		p.annotateType = GranularStruct
	}

	return p
}

func parseCommentForStruct(comments []string) *params {
	tags := gengo.ExtractCommentTags("+", comments)
	ts := tags["structType"]
	if len(ts) > 1 {
		panic("value of struct type cannot be set twice")
	}
	if len(ts) != 0 {
		p := &params{}
		switch ts[0] {
		case "atomic":
			p.annotateType = AtomicStruct
		case "granular":
			p.annotateType = GranularMap
		default:
			panic("unknown map type " + ts[0])
		}

		return p
	}

	return nil
}

func (p *params) Name() string {
	return p.name
}

func (p *params) Type() *types.Type {
	return p.t
}

func (p *params) Member() *types.Member {
	return p.m
}

func (p *params) Dst() string {
	return p.variable("dst", p.refers)
}

func (p *params) Src() string {
	if len(p.srcRefers) == 0 {
		return p.variable("src", p.refers)
	}
	return p.variable("src", p.srcRefers)
}

func (*params) variable(root string, refers []Refer) string {
	path := root
	for _, refer := range refers {
		path = refer.Ref(path)
	}
	return path
}

func (p *params) AnnotateType() AnnotateType {
	if p.annotateType == 0 {
		return None
	}
	return p.annotateType
}

func (p *params) Keys() []string {
	return p.keys
}

func (p *params) CheckNil() {
	p.checkNil = true
}

func (p *params) Members() []Params {
	ms := []Params{}
	for i, m := range p.t.Members {
		mp := newParams(p.name+"."+m.Name, m.Type, m.CommentLines)
		mp.m = &p.t.Members[i]
		mp.refers = append(mp.refers, Field(m.Name))
		// always pass addr to overlay func
		// for pointer, it's no need to addr
		// for builtin, it will be handled inline and it's no need to pass to overlay func
		switch m.Type.Kind {
		case types.Pointer, types.Builtin:
		default:
			mp.refers = append(mp.refers, Addr(&mp.checkNil))
		}
		ms = append(ms, mp)
	}
	return ms
}

func (p *params) Pointer() Params {
	switch p.t.Kind {
	case types.Pointer:
		// preserve annotateType
		ep := &params{
			name:         p.name,
			t:            p.t.Elem,
			annotateType: p.annotateType,
			keys:         p.keys,
			refers:       p.refers,
		}
		// builtin type will be handled inline but not passed into overlay function
		// e.g. for *string
		//   *dst.string = *src.string
		// but for *struct
		//   overlayXXX(dst.struct, src.struct)
		if p.t.Elem.Kind == types.Builtin {
			ep.refers = append(ep.refers, Pointer())
		}
		return ep
	default:
		panic("Pointer() called by unexpected kind" + p.t.Kind)
	}
}

func (p *params) Slice(dst, src string) Params {
	switch p.t.Kind {
	case types.Slice, types.Array:
		ep := newParams(p.name+"[]", p.t.Elem, nil)
		ep.refers = append(ep.refers, Pointer(), Index(dst))
		ep.srcRefers = append(ep.srcRefers, Pointer(), Index(src))
		switch p.t.Elem.Kind {
		case types.Struct, types.Alias:
			ep.refers = append(ep.refers, Addr(&ep.checkNil))
			ep.srcRefers = append(ep.srcRefers, Addr(&ep.checkNil))
		case types.Builtin:
			// do nothing
		default:
			panic("unexpected elem kind of slice: " + p.t.Kind)
		}
		return ep
	default:
		panic("Slice() called by unexpected kind" + p.t.Kind)
	}
}

func (p *params) Map() Params {
	switch p.t.Kind {
	case types.Map:
		ep := newParams(p.name+"[]", p.t.Elem, nil)
		ep.refers = append(ep.refers, Prefix("v"))
		switch p.t.Elem.Kind {
		case types.Struct, types.Alias:
			ep.refers = append(ep.refers, Addr(&ep.checkNil))
		case types.Builtin:
			// do nothing
		default:
			panic("unexpected elem kind of slice: " + p.t.Kind)
		}
		return ep
	default:
		panic("Map() called by unexpected kind" + p.t.Kind)
	}
}

func (p *params) Underlying() Params {
	switch p.t.Kind {
	case types.Alias:
		ep := &params{
			name:         p.name,
			t:            p.t.Underlying,
			annotateType: p.annotateType,
			keys:         p.keys,
		}
		ep.refers = append(ep.refers, Prefix("n"))
		if p.t.Underlying.Kind == types.Builtin {
			ep.refers = append(ep.refers, Pointer())
		}
		return ep
	default:
		panic("Underlying() called by unexpected kind" + p.t.Kind)
	}
}

func Panic(msg string, args ...any) {
	logger := slog.Default()
	if !logger.Enabled(context.Background(), slog.LevelError) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:]) // skip [Callers, Infof]
	r := slog.NewRecord(time.Now(), slog.LevelError, msg, pcs[0])
	r.Add(args...)
	_ = logger.Handler().Handle(context.Background(), r)
	os.Exit(1)
}
