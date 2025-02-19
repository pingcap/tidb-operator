package util

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
)

func TestAppendOverwriteEnv(t *testing.T) {
	g := NewGomegaWithT(t)

	a := []corev1.EnvVar{
		{
			Name:  "ak_1",
			Value: "ak_1",
		},
		{
			Name:  "ak_2",
			Value: "ak_2",
		},
	}
	b := []corev1.EnvVar{
		{
			Name:  "bk_1",
			Value: "bk_1",
		},
		{
			Name:  "ak_1",
			Value: "ak_10",
		},
		{
			Name:  "ak_2",
			Value: "ak_20",
		},
		{
			Name:  "bk_2",
			Value: "bk_2",
		},
	}

	expect := []corev1.EnvVar{
		{
			Name:  "ak_1",
			Value: "ak_10",
		},
		{
			Name:  "ak_2",
			Value: "ak_20",
		},
		{
			Name:  "bk_1",
			Value: "bk_1",
		},
		{
			Name:  "bk_2",
			Value: "bk_2",
		},
	}

	get := AppendOverwriteEnv(a, b)
	g.Expect(get).Should(Equal(expect))
}

func TestAppendEnvIfPresent(t *testing.T) {
	tests := []struct {
		name string
		a    []corev1.EnvVar
		envs map[string]string
		n    string
		want []corev1.EnvVar
	}{
		{
			"does not exist",
			[]corev1.EnvVar{
				{
					Name:  "foo",
					Value: "bar",
				},
			},
			nil,
			"TEST_ENV",
			[]corev1.EnvVar{
				{
					Name:  "foo",
					Value: "bar",
				},
			},
		},
		{
			"does exist",
			[]corev1.EnvVar{
				{
					Name:  "foo",
					Value: "bar",
				},
			},
			map[string]string{
				"TEST_ENV": "TEST_VAL",
			},
			"TEST_ENV",
			[]corev1.EnvVar{
				{
					Name:  "foo",
					Value: "bar",
				},
				{
					Name:  "TEST_ENV",
					Value: "TEST_VAL",
				},
			},
		},
		{
			"already exist",
			[]corev1.EnvVar{
				{
					Name:  "foo",
					Value: "bar",
				},
				{
					Name:  "TEST_ENV",
					Value: "TEST_OLD_VAL",
				},
			},
			map[string]string{
				"TEST_ENV": "TEST_VAL",
			},
			"TEST_ENV",
			[]corev1.EnvVar{
				{
					Name:  "foo",
					Value: "bar",
				},
				{
					Name:  "TEST_ENV",
					Value: "TEST_OLD_VAL",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			for k, v := range tt.envs {
				os.Setenv(k, v)
			}
			got := AppendEnvIfPresent(tt.a, tt.n)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("unwant (-want, +got): %s", diff)
			}
		})
	}
}

func TestCombineStringMap(t *testing.T) {
	g := NewGomegaWithT(t)

	a := map[string]string{
		"a": "av",
	}
	origA := map[string]string{
		"a": "av",
	}
	b := map[string]string{
		"b": "bv",
	}
	c := map[string]string{
		"c": "cv",
	}
	dropped := map[string]string{
		"a": "aov",
	}

	expect1 := map[string]string{
		"a": "av",
		"b": "bv",
		"c": "cv",
	}
	expect2 := map[string]string{
		"a": "aov",
		"b": "bv",
		"c": "cv",
	}

	res := CombineStringMap(a, b, c, dropped)
	g.Expect(res).Should(Equal(expect1))
	g.Expect(a).Should(Equal(origA))

	res = CombineStringMap(nil, b, c, dropped)
	g.Expect(res).Should(Equal(expect2))

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
