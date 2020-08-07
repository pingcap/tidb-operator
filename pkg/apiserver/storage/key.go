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

package storage

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	labelDomain = "apiserver.pingcap.com"
)

var (
	groupLabel     = labelDomain + "/group"
	kindLabel      = labelDomain + "/kind"
	namespaceLabel = labelDomain + "/namespace"
	nameLabel      = labelDomain + "/name"
)

// objKey is a structured representation of key '/{group}/{kind}/{ns}/{name}'
type objKey struct {
	group     string
	kind      string
	namespace string
	name      string
}

func newObjectKey(key string) *objKey {
	s := strings.Split(strings.TrimPrefix(key, "/"), "/")
	if len(s) < 2 {
		panic("expect a string key in /{group}/{kind}/{ns?}/{name?} format")
	}
	k := &objKey{
		group: s[0],
		kind:  s[1],
	}
	if len(s) > 2 {
		k.namespace = s[2]
	}
	if len(s) > 3 {
		k.name = s[3]
	}
	return k
}

func (o *objKey) fullName() string {
	sb := strings.Builder{}
	sb.WriteString(strings.ReplaceAll(o.group, ".", "-"))
	sb.WriteRune('-')
	sb.WriteString(o.kind)
	if o.namespace != "" {
		sb.WriteRune('-')
		sb.WriteString(o.namespace)
	}
	if o.name != "" {
		sb.WriteRune('-')
		sb.WriteString(o.name)
	}
	return sb.String()
}

func (o *objKey) labelMap() map[string]string {
	m := map[string]string{
		groupLabel: o.group,
		kindLabel:  o.kind,
	}
	if o.namespace != "" {
		m[namespaceLabel] = o.namespace
	}
	if o.name != "" {
		m[nameLabel] = o.name
	}
	return m
}

func (o *objKey) objectMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:   o.fullName(),
		Labels: o.labelMap(),
	}
}

func (o *objKey) labelSelectorStr() string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("%s=%s,%s=%s",
		groupLabel, o.group,
		kindLabel, o.kind))
	if o.namespace != "" {
		sb.WriteString(fmt.Sprintf(",%s=%s", namespaceLabel, o.namespace))
	}
	if o.name != "" {
		sb.WriteString(fmt.Sprintf(",%s=%s", nameLabel, o.name))
	}
	return sb.String()
}
