package compatibility

import (
	"fmt"

	"github.com/Masterminds/semver/v3"
)

func Check(v *semver.Version, constraints ...Constraints) bool {
	for _, c := range constraints {
		if !c.check(v) {
			return false
		}
	}

	return true
}

type Constraints interface {
	check(v *semver.Version) bool
}

type constraints struct {
	*semver.Constraints
}

func (c *constraints) check(v *semver.Version) bool {
	return c.Check(v)
}

func MustNewConstraints(expr string) Constraints {
	v, err := semver.NewConstraint(expr)
	if err != nil {
		panic(fmt.Errorf("cannot parse constraints %v: %w", expr, err))
	}

	v.IncludePrerelease = true

	return &constraints{
		Constraints: v,
	}
}
