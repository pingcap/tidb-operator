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

package utils

// SetIfChanged sets the destination pointer to the source value if they are different.
// Returns true if the value was changed, false otherwise.
func SetIfChanged[T comparable](dst *T, src T) bool {
	if *dst != src {
		*dst = src
		return true
	}

	return false
}

// NewAndSetIfChanged initializes a nil pointer destination and sets its value if different from source.
// Parameters:
//   - dst: A double pointer to the destination value that may be nil
//   - src: The source value to compare against and potentially set
//
// Returns:
//   - true if the destination was initialized or its value was changed
//   - false if no change was needed (either dst was nil and src was zero value, or dst already equals src)
//
// This function handles the case where the destination pointer is nil by creating a new instance.
// If the destination is nil, it creates a new instance and sets it to the source value.
func NewAndSetIfChanged[T comparable](dst **T, src T) bool {
	if *dst == nil {
		zero := new(T)
		if *zero == src {
			return false
		}
		*dst = zero
	}

	return SetIfChanged(*dst, src)
}
