// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package updater

import (
	"context"
)

type Actor interface {
	ScaleOut(ctx context.Context) error
	Update(ctx context.Context) error
	ScaleInUpdate(ctx context.Context) (unavailable bool, _ error)
	ScaleInOutdated(ctx context.Context) (unavailable bool, _ error)

	// delete all instances marked as defer deletion
	Cleanup(ctx context.Context) error
}

// TODO: return instance list after Do
type Executor interface {
	Do(ctx context.Context) (bool, error)
}

type executor struct {
	update              int
	outdated            int
	desired             int
	unavailableUpdate   int
	unavailableOutdated int
	maxSurge            int
	maxUnavailable      int

	act Actor
}

func NewExecutor(
	act Actor,
	update,
	outdated,
	desired,
	unavailableUpdate,
	unavailableOutdated,
	maxSurge,
	maxUnavailable int,
) Executor {
	return &executor{
		update:              update,
		outdated:            outdated,
		desired:             desired,
		unavailableUpdate:   unavailableUpdate,
		unavailableOutdated: unavailableOutdated,
		maxSurge:            maxSurge,
		maxUnavailable:      maxUnavailable,
		act:                 act,
	}
}

// TODO: add scale in/out rate limit
//
//nolint:gocyclo // refactor if possible
func (ex *executor) Do(ctx context.Context) (bool, error) {
	for ex.update != ex.desired || ex.outdated != 0 {
		actual := ex.update + ex.outdated
		available := actual - ex.unavailableUpdate - ex.unavailableOutdated
		maximum := ex.desired + min(ex.maxSurge, ex.outdated)
		minimum := ex.desired - ex.maxUnavailable
		switch {
		case actual < maximum:
			if err := ex.act.ScaleOut(ctx); err != nil {
				return false, err
			}
			ex.update += 1
			ex.unavailableUpdate += 1

		case actual == maximum:
			if ex.update < ex.desired {
				// update will always prefer unavailable one so available will not changed if there are
				// unavailable and outdated instances
				if ex.unavailableOutdated > 0 {
					if err := ex.act.Update(ctx); err != nil {
						return false, err
					}
					ex.outdated -= 1
					ex.unavailableOutdated -= 1
					ex.update += 1
					ex.unavailableUpdate += 1
				} else {
					// DON'T decrease available if available is less than minimum
					if available <= minimum {
						return true, nil
					}

					if err := ex.act.Update(ctx); err != nil {
						return false, err
					}
					ex.outdated -= 1
					ex.update += 1
					ex.unavailableUpdate += 1
				}
			} else {
				// => ex.update + ex.outdated == ex.desired + min(ex.maxSurge, ex.outdated) and ex.update >= ex.desired
				// => ex.outdated <= min(ex.maxSurge, ex.outdated)
				// => ex.outdated <= ex.maxSurge
				// => ex.outdated = min(ex.maxSurge, ex.outdated)
				// => ex.update + ex.outdated >= ex.desired + ex.outdated
				// => ex.update == ex.desired
				// => ex.outdated != 0 (ex.update != ex.desired || ex.outdated != 0 in for loop condition)
				if available <= minimum {
					return true, nil
				}

				unavailable, err := ex.act.ScaleInOutdated(ctx)
				if err != nil {
					return false, err
				}
				// scale in may not choose an unavailable outdated so just descrease the outdated
				// and assume we always choose an available outdated.
				// And then wait if next available is less than minimum
				if unavailable {
					ex.outdated -= 1
					ex.unavailableOutdated -= 1
				} else {
					ex.outdated -= 1
				}
			}
		case actual > maximum:
			// Scale in op may choose an available instance.
			// Assume we always choose an unavailable one, we will scale once and wait until next reconcile
			checkAvail := false
			if ex.update > ex.desired {
				unavailable, err := ex.act.ScaleInUpdate(ctx)
				if err != nil {
					return false, err
				}
				if unavailable {
					ex.update -= 1
					ex.unavailableUpdate -= 1
				} else {
					ex.update -= 1
					available -= 1
					checkAvail = true
				}
			} else {
				// ex.update + ex.outdated > ex.desired + min(ex.maxSurge, ex.outdated) and ex.update <= ex.desired
				// => ex.outdated > min(ex.maxSurge, ex.outdated)
				// => ex.outdated > 0
				unavailable, err := ex.act.ScaleInOutdated(ctx)
				if err != nil {
					return false, err
				}
				if unavailable {
					ex.outdated -= 1
					ex.unavailableOutdated -= 1
				} else {
					ex.outdated -= 1
					available -= 1
					checkAvail = true
				}
			}
			// Wait if available is less than minimum
			if checkAvail && available <= minimum {
				return true, nil
			}
		}
	}

	if ex.unavailableUpdate > 0 {
		// wait until update are all available
		return true, nil
	}

	if err := ex.act.Cleanup(ctx); err != nil {
		return false, err
	}

	return false, nil
}
