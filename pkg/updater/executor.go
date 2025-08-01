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

	"github.com/go-logr/logr"
)

type Actor interface {
	ScaleOut(ctx context.Context) (created bool, _ error)
	Update(ctx context.Context) error
	ScaleInUpdate(ctx context.Context) (operated bool, unavailable bool, _ error)
	ScaleInOutdated(ctx context.Context) (operated bool, unavailable bool, _ error)

	// Cleanup deletes all instances marked as defer deletion
	Cleanup(ctx context.Context) error
}

// Executor is an executor that updates the instances.
// TODO: return instance list after Do
type Executor interface {
	Do(ctx context.Context) (bool, error)
}

type executor struct {
	update              int
	outdated            int
	offlining           int
	offlined            int
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
	offlining,
	offlined,
	desired,
	unavailableUpdate,
	unavailableOutdated,
	maxSurge,
	maxUnavailable int,
) Executor {
	return &executor{
		update:              update,
		outdated:            outdated,
		offlining:           offlining,
		offlined:            offlined,
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
	logger := logr.FromContextOrDiscard(ctx).WithName("Updater")

	logger.Info("before the loop", "update", ex.update, "outdated", ex.outdated,
		"desired", ex.desired, "offlining", ex.offlining, "offlined", ex.offlined)
	for ex.update != ex.desired || ex.outdated != 0 || ex.offlining != 0 {
		actual := ex.update + ex.outdated
		onlineActual := actual - ex.offlining
		available := actual - ex.unavailableUpdate - ex.unavailableOutdated
		maximum := ex.desired + min(ex.maxSurge, ex.outdated)
		minimum := ex.desired - ex.maxUnavailable
		logger.Info("loop",
			"update", ex.update, "outdated", ex.outdated, "onlineActual", onlineActual, "desired", ex.desired,
			"unavailableUpdate", ex.unavailableUpdate, "unavailableOutdated", ex.unavailableOutdated,
			"actual", actual, "available", available, "maximum", maximum, "minimum", minimum)
		switch {
		case actual < maximum || onlineActual < maximum:
			logger.Info("scale out")
			created, err := ex.act.ScaleOut(ctx)
			if err != nil {
				return false, err
			}
			if created {
				ex.update += 1
				ex.unavailableUpdate += 1
			} else if ex.offlining > 0 {
				// ScaleOut canceled an offlining operation instead of creating new instance
				// The offlining instance was already counted in update, just reduce offlining count
				ex.offlining -= 1
			}

		case actual == maximum:
			if ex.update < ex.desired {
				// update will always prefer unavailable one so available will not changed if there are
				// unavailable and outdated instances
				if ex.unavailableOutdated > 0 {
					logger.Info("update unavailable outdated")
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
						logger.Info("wait because available is less than minimum",
							"available", available, "minimum", minimum)
						return true, nil
					}

					logger.Info("update available outdated")
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
					logger.Info("wait because available is less than minimum",
						"available", available, "minimum", minimum)
					return true, nil
				}

				logger.Info("scale in outdated")
				operated, unavailable, err := ex.act.ScaleInOutdated(ctx)
				if err != nil {
					return false, err
				}
				if !operated {
					// No operation performed, wait for next reconcile
					return true, nil
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
				logger.Info("scale in update")
				operated, unavailable, err := ex.act.ScaleInUpdate(ctx)
				if err != nil {
					return false, err
				}
				if !operated {
					// No operation performed, wait for next reconcile
					return true, nil
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
				logger.Info("scale in outdated")
				operated, unavailable, err := ex.act.ScaleInOutdated(ctx)
				if err != nil {
					return false, err
				}
				if !operated {
					// No operation performed, wait for next reconcile
					return true, nil
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
				logger.Info("wait because available is less than minimum",
					"available", available, "minimum", minimum)
				return true, nil
			}
		}
	}

	if ex.unavailableUpdate > 0 {
		// wait until update are all available
		logger.Info("wait because unavailable update is not zero",
			"unavailableUpdate", ex.unavailableUpdate, "update", ex.update, "outdated", ex.outdated, "desired", ex.desired)
		return true, nil
	}

	if err := ex.act.Cleanup(ctx); err != nil {
		return false, err
	}

	return false, nil
}
