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

package ginkgo

import (
	"regexp"
	"strings"

	"github.com/onsi/ginkgo"
	ginkgoconfig "github.com/onsi/ginkgo/config"
)

// ItWhenFocus run It only when It is focused
func ItWhenFocus(text string, body interface{}, timeout ...float64) bool {
	if !matchesFocus(text, ginkgoconfig.GinkgoConfig.FocusStrings) {
		return ginkgo.PIt(text, body)
	}
	return ginkgo.It(text, body)
}

// ContextWhenFocus run Context only when Context is focused
func ContextWhenFocus(text string, body func()) bool {
	if !matchesFocus(text, ginkgoconfig.GinkgoConfig.FocusStrings) {
		return ginkgo.PContext(text, body)
	}
	return ginkgo.Context(text, body)
}

func matchesFocus(text string, focusStrings []string) bool {
	// Ginkgo combines repeated --focus flags with OR.
	focusString := strings.Join(focusStrings, "|")
	return focusString != "" && regexp.MustCompile(focusString).MatchString(text)
}
