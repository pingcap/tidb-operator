package manager

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestParseMac(t *testing.T) {
	g := NewGomegaWithT(t)

	data := `
Interface  Type       Source     Model       MAC
-------------------------------------------------------
vnet1      bridge     br0        virtio      52:54:00:d4:9e:bb
`
	output, err := parserMac(data)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(output).To(Equal("52:54:00:d4:9e:bb"))
}

func TestParseIP(t *testing.T) {
	g := NewGomegaWithT(t)

	data := `
Name       MAC address          Protocol     Address
-------------------------------------------------------------------------------
eth0       52:54:00:4c:5b:c0    ipv4         172.16.30.216/24
-          -                    ipv6         fe80::5054:ff:fe4c:5bc0/64
`
	output, err := parserIP(data)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(output).To(Equal("172.16.30.216"))
}

func TestParseIPFromIPNeign(t *testing.T) {
	g := NewGomegaWithT(t)

	data := `
172.16.30.216 dev br0 lladdr 52:54:00:4c:5b:c0 STALE
`
	output, err := parserIPFromIPNeign(data)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(output).To(Equal("172.16.30.216"))
}

func TestParseVMs(t *testing.T) {
	g := NewGomegaWithT(t)

	data := `
 Id    Name                           State
----------------------------------------------------
 6     vm2                            running
 11    vm3                            running
 12    vm1                            running
 -     vm-template                    shut off
`
	vms := parserVMs(data)

	var expectedVMs []*VM
	expectedVMs = append(expectedVMs, &VM{
		Name: "vm2",
	})
	expectedVMs = append(expectedVMs, &VM{
		Name: "vm3",
	})
	expectedVMs = append(expectedVMs, &VM{
		Name: "vm1",
	})
	g.Expect(vms).To(Equal(expectedVMs))
}
