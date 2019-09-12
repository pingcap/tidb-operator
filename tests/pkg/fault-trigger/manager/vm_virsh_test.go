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
