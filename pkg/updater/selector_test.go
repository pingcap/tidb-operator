package updater

import (
	"testing"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
	"github.com/stretchr/testify/assert"
)

func TestSelector(t *testing.T) {
	cases := []struct {
		desc     string
		ps       []PreferPolicy[*runtime.PD]
		allowed  []*runtime.PD
		expected string
	}{
		{
			desc: "no policy",
			allowed: []*runtime.PD{
				fakePD("aaa", true),
				fakePD("bbb", false),
				fakePD("ccc", true),
				fakePD("ddd", false),
			},
			expected: "aaa",
		},
		{
			desc: "prefer unavailable",
			ps: []PreferPolicy[*runtime.PD]{
				PreferUnavailable[*runtime.PD](),
			},
			allowed: []*runtime.PD{
				fakePD("aaa", true),
				fakePD("bbb", false),
				fakePD("ccc", true),
				fakePD("ddd", false),
			},
			expected: "bbb",
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			s := NewSelector(c.ps...)
			choosed := s.Choose(c.allowed)
			assert.Equal(tt, c.expected, choosed)
		})
	}
}

func fakePD(name string, ready bool) *runtime.PD {
	return runtime.FromPD(fake.FakeObj(name, func(obj *v1alpha1.PD) *v1alpha1.PD {
		obj.Generation = 2
		obj.Labels = map[string]string{
			v1alpha1.LabelKeyInstanceRevisionHash: "test",
		}
		obj.Status.CurrentRevision = "test"
		coreutil.SetStatusCondition[scope.PD](obj, *coreutil.Ready())
		if ready {
			obj.Status.ObservedGeneration = obj.Generation
		}
		return obj
	}))
}
