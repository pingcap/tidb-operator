// Copyright 2023 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

// this file is copied (with some modify) from https://github.com/kubernetes/kubernetes/blob/v1.22.17/staging/src/k8s.io/apimachinery/pkg/util/json/json.go

package k8s

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Marshal delegates to json.Marshal
// It is only here so this package can be a drop-in for common encoding/json uses
func Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// limit recursive depth to prevent stack overflow errors
const maxDepth = 10000

// Unmarshal unmarshals the given data
// If v is a *map[string]interface{}, *[]interface{}, or *interface{} numbers
// are converted to int64 or float64
func Unmarshal(data []byte, v interface{}) error {
	switch v := v.(type) {
	case *map[string]interface{}:
		// Build a decoder from the given data
		decoder := json.NewDecoder(bytes.NewBuffer(data))
		// Preserve numbers, rather than casting to float64 automatically
		decoder.UseNumber()
		// Run the decode
		if err := decoder.Decode(v); err != nil {
			return err
		}
		// If the decode succeeds, post-process the map to convert json.Number objects to int64 or float64
		return ConvertMapNumbers(*v, 0)

	case *[]interface{}:
		// Build a decoder from the given data
		decoder := json.NewDecoder(bytes.NewBuffer(data))
		// Preserve numbers, rather than casting to float64 automatically
		decoder.UseNumber()
		// Run the decode
		if err := decoder.Decode(v); err != nil {
			return err
		}
		// If the decode succeeds, post-process the map to convert json.Number objects to int64 or float64
		return ConvertSliceNumbers(*v, 0)

	case *interface{}:
		// Build a decoder from the given data
		decoder := json.NewDecoder(bytes.NewBuffer(data))
		// Preserve numbers, rather than casting to float64 automatically
		decoder.UseNumber()
		// Run the decode
		if err := decoder.Decode(v); err != nil {
			return err
		}
		// If the decode succeeds, post-process the map to convert json.Number objects to int64 or float64
		return ConvertInterfaceNumbers(v, 0)

	default:
		return json.Unmarshal(data, v)
	}
}

// ConvertInterfaceNumbers converts any json.Number values to int64 or float64.
// Values which are map[string]interface{} or []interface{} are recursively visited
func ConvertInterfaceNumbers(v *interface{}, depth int) error {
	var err error
	switch v2 := (*v).(type) {
	case json.Number:
		*v, err = convertNumber(v2)
	case map[string]interface{}:
		err = ConvertMapNumbers(v2, depth+1)
	case []interface{}:
		err = ConvertSliceNumbers(v2, depth+1)
	}
	return err
}

// ConvertMapNumbers traverses the map, converting any json.Number values to int64 or float64.
// values which are map[string]interface{} or []interface{} are recursively visited
func ConvertMapNumbers(m map[string]interface{}, depth int) error {
	if depth > maxDepth {
		return fmt.Errorf("exceeded max depth of %d", maxDepth)
	}

	var err error
	for k, v := range m {
		switch v := v.(type) {
		case json.Number:
			m[k], err = convertNumber(v)
		case map[string]interface{}:
			err = ConvertMapNumbers(v, depth+1)
		case []interface{}:
			err = ConvertSliceNumbers(v, depth+1)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// ConvertSliceNumbers traverses the slice, converting any json.Number values to int64 or float64.
// values which are map[string]interface{} or []interface{} are recursively visited
func ConvertSliceNumbers(s []interface{}, depth int) error {
	if depth > maxDepth {
		return fmt.Errorf("exceeded max depth of %d", maxDepth)
	}

	var err error
	for i, v := range s {
		switch v := v.(type) {
		case json.Number:
			s[i], err = convertNumber(v)
		case map[string]interface{}:
			err = ConvertMapNumbers(v, depth+1)
		case []interface{}:
			err = ConvertSliceNumbers(v, depth+1)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// convertNumber converts a json.Number to an int64 or float64, or returns an error
func convertNumber(n json.Number) (interface{}, error) {
	// Attempt to convert to an int64 first
	if i, err := n.Int64(); err == nil {
		return i, nil
	}
	// Return a float64 (default json.Decode() behavior)
	// An overflow will return an error
	return n.Float64()
}
