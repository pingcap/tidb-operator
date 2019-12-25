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

package util

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestGetOrdinalFromPodName(t *testing.T) {
	g := NewGomegaWithT(t)

	i, err := GetOrdinalFromPodName("pod-1")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(i).To(Equal(int32(1)))

	i, err = GetOrdinalFromPodName("pod-notint")
	g.Expect(err).To(HaveOccurred())
	g.Expect(i).To(Equal(int32(0)))
}

func TestGetNextOrdinalPodName(t *testing.T) {
	g := NewGomegaWithT(t)
	g.Expect(GetNextOrdinalPodName("pod-1", 1)).To(Equal("pod-2"))
}

func TestIsSubMapOf(t *testing.T) {
	g := NewGomegaWithT(t)

	g.Expect(IsSubMapOf(
		nil,
		map[string]string{
			"k1": "v1",
		})).To(BeTrue())
	g.Expect(IsSubMapOf(
		map[string]string{
			"k1": "v1",
		},
		map[string]string{
			"k1": "v1",
		})).To(BeTrue())
	g.Expect(IsSubMapOf(
		map[string]string{
			"k1": "v1",
		},
		map[string]string{
			"k1": "v1",
			"k2": "v2",
		})).To(BeTrue())
	g.Expect(IsSubMapOf(
		map[string]string{},
		map[string]string{
			"k1": "v1",
		})).To(BeTrue())
	g.Expect(IsSubMapOf(
		map[string]string{
			"k1": "v1",
			"k2": "v2",
		},
		map[string]string{
			"k1": "v1",
		})).To(BeFalse())
}
