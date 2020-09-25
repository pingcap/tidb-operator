package v1alpha1

import (
	"strconv"
	"testing"

	fuzz "github.com/google/gofuzz"
	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/util/toml"
)

func TestTiDBConfigWraper(t *testing.T) {
	g := NewGomegaWithT(t)

	f := fuzz.New().Funcs(
		// the toml pkg we use do support to unmarshal a value overflow int64
		// when unmarshal to go map[string]interface{
		// just set the value in uint32 range now for
		func(e *uint64, c fuzz.Continue) {
			*e = uint64(c.Uint32())
		},
		func(e *uint, c fuzz.Continue) {
			*e = uint(c.Uint32())
		},

		func(e *string, c fuzz.Continue) {
			*e = "s" + strconv.Itoa(c.Intn(100))
		},
	)
	for i := 0; i < 100; i++ {
		var tidbConfig TiDBConfig
		f.Fuzz(&tidbConfig)

		tomlData, err := toml.Marshal(&tidbConfig)
		g.Expect(err).Should(BeNil())

		tidbConfigWraper := NewTiDBConfig()
		err = tidbConfigWraper.UnmarshalTOML(tomlData)
		g.Expect(err).Should(BeNil())

		tomlDataBack, err := tidbConfigWraper.MarshalTOML()
		g.Expect(err).Should(BeNil())

		var tidbConfigBack TiDBConfig
		err = toml.Unmarshal(tomlDataBack, &tidbConfigBack)
		g.Expect(err).Should(BeNil())
		g.Expect(tidbConfigBack).Should(Equal(tidbConfig))
	}
}
