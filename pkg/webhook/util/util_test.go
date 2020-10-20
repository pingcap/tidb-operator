// Copyright 2020 PingCAP, Inc.
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
	"encoding/json"
	"errors"
	"testing"

	. "github.com/onsi/gomega"
	"gomodules.xyz/jsonpatch/v2"
	admission "k8s.io/api/admission/v1beta1"
	v1 "k8s.io/api/core/v1"
)

func TestAR(t *testing.T) {
	g := NewGomegaWithT(t)
	var resp *admission.AdmissionResponse

	resp = ARFail(errors.New("err"))
	g.Expect(resp.Allowed).Should(BeFalse())

	resp = ARSuccess()
	g.Expect(resp.Allowed).Should(BeTrue())

	resp = ARPatch(nil)
	g.Expect(resp.Allowed).Should(BeTrue())
}

func TestCreateJsonPatch(t *testing.T) {
	g := NewGomegaWithT(t)

	original := &v1.Pod{}
	original.Name = "old"
	current := &v1.Pod{}
	current.Name = "new"

	data, err := CreateJsonPatch(original, current)
	g.Expect(err).Should(BeNil())

	var ops []jsonpatch.Operation
	err = json.Unmarshal(data, &ops)
	g.Expect(err).Should(BeNil())
	op := ops[0]
	g.Expect(op).Should(Equal(jsonpatch.Operation{
		Operation: "replace",
		Path:      "/metadata/name",
		Value:     "new",
	}))
}
