// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// di
package utils

type Integer interface {
	int | int32 | int64
}

// ValuesDiffer determines whether the values pointed by two integer pointers are different.
// If either a or b is nil, returns false. Otherwise, returns true if *a != *b.
func ValuesDiffer[T Integer](a, b *T) bool {
	if a == nil || b == nil {
		return false
	}
	return *a != *b
}
