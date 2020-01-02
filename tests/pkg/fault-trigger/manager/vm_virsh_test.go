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

package manager

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestVirshParseVMs(t *testing.T) {
	g := NewGomegaWithT(t)

	data := `
 Id    Name                           State
----------------------------------------------------
 6     vm2                            shut off
 11    vm3                            running
 12    vm1                            running
 -     vm-template                    shut off
`

	vmManager := VirshVMManager{}
	vms := vmManager.parserVMs(data)

	var expectedVMs []*VM
	expectedVMs = append(expectedVMs, &VM{
		Name:   "vm2",
		Status: "shut",
	})
	expectedVMs = append(expectedVMs, &VM{
		Name:   "vm3",
		Status: "running",
	})
	expectedVMs = append(expectedVMs, &VM{
		Name:   "vm1",
		Status: "running",
	})
	g.Expect(vms).To(Equal(expectedVMs))
}
