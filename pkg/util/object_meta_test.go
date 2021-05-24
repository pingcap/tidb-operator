package util

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestCombineStringMap(t *testing.T) {
	g := NewGomegaWithT(t)

	a := map[string]string{
		"a": "av",
	}
	b := map[string]string{
		"b": "bv",
	}
	c := map[string]string{
		"c": "cv",
	}
	overwrite := map[string]string{
		"a": "aov",
	}

	expect := map[string]string{
		"a": "aov",
		"b": "bv",
		"c": "cv",
	}

	res := CombineStringMap(a, b, c, overwrite)
	g.Expect(res).Should(Equal(expect))

	res = CombineStringMap(nil, b, c, overwrite)
	g.Expect(res).Should(Equal(expect))

}

func TestCopyStringMap(t *testing.T) {
	g := NewGomegaWithT(t)

	src := map[string]string{
		"a": "av",
	}

	// modify res and check src unchanged
	res := CopyStringMap(src)
	res["test"] = "v"
	res["a"] = "overwrite"
	g.Expect(src).Should(Equal(map[string]string{"a": "av"}))
}
