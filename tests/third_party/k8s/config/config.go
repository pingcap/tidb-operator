/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package config simplifies the declaration of configuration options.
// Right now the implementation maps them directly to command line
// flags. When combined with test/e2e/framework/viperconfig in a test
// suite, those flags then can also be read from a config file.
//
// The command line flags all get stored in a private flag set. The
// developer of the E2E test suite decides how they are exposed. Options
// include:
//   - exposing as normal flags in the actual command line:
//     CopyFlags(Flags, flag.CommandLine)
//   - populate via test/e2e/framework/viperconfig:
//     viperconfig.ViperizeFlags("my-config.yaml", "", Flags)
//   - a combination of both:
//     CopyFlags(Flags, flag.CommandLine)
//     viperconfig.ViperizeFlags("my-config.yaml", "", flag.CommandLine)
//
// Instead of defining flags one-by-one, test developers annotate a
// structure with tags and then call a single function. This is the
// same approach as in https://godoc.org/github.com/jessevdk/go-flags,
// but implemented so that a test suite can continue to use the normal
// "flag" package.
//
// For example, a file storage/csi.go might define:
//
//	var scaling struct {
//	        NumNodes int  `default:"1" description:"number of nodes to run on"`
//	        Master string
//	}
//	_ = config.AddOptions(&scaling, "storage.csi.scaling")
//
// This defines the following command line flags:
//
//	-storage.csi.scaling.numNodes=<int>  - number of nodes to run on (default: 1)
//	-storage.csi.scaling.master=<string>
//
// All fields in the structure must be exported and have one of the following
// types (same as in the `flag` package):
// - bool
// - time.Duration
// - float64
// - string
// - int
// - int64
// - uint
// - uint64
// - and/or nested or embedded structures containing those basic types.
//
// Each basic entry may have a tag with these optional keys:
//
//	usage:   additional explanation of the option
//	default: the default value, in the same format as it would
//	         be given on the command line and true/false for
//	         a boolean
//
// The names of the final configuration options are a combination of an
// optional common prefix for all options in the structure and the
// name of the fields, concatenated with a dot. To get names that are
// consistent with the command line flags defined by `ginkgo`, the
// initial character of each field name is converted to lower case.
//
// There is currently no support for aliases, so renaming the fields
// or the common prefix will be visible to users of the test suite and
// may breaks scripts which use the old names.
//
// The variable will be filled with the actual values by the test
// suite before running tests. Beware that the code which registers
// Ginkgo tests cannot use those config options, because registering
// tests and options both run before the E2E test suite handles
// parameters.

// this file is copied from k8s.io/kubernetes/test/e2e/framework/config/config.go @v1.23.17

package config

import (
	"flag"
)

// Flags is the flag set that AddOptions adds to. Test authors should
// also use it instead of directly adding to the global command line.
var Flags = flag.NewFlagSet("", flag.ContinueOnError)

// CopyFlags ensures that all flags that are defined in the source flag
// set appear in the target flag set as if they had been defined there
// directly. From the flag package it inherits the behavior that there
// is a panic if the target already contains a flag from the source.
func CopyFlags(source *flag.FlagSet, target *flag.FlagSet) {
	source.VisitAll(func(flag *flag.Flag) {
		// We don't need to copy flag.DefValue. The original
		// default (from, say, flag.String) was stored in
		// the value and gets extracted by Var for the help
		// message.
		target.Var(flag.Value, flag.Name, flag.Usage)
	})
}
