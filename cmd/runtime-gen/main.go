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

package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"
	_ "k8s.io/api/core/v1"
	_ "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/gengo/v2"
	"k8s.io/gengo/v2/generator"
	"k8s.io/gengo/v2/namer"

	"github.com/pingcap/tidb-operator/v2/cmd/runtime-gen/generators"
)

func main() {
	args := &Args{}

	// Collect and parse flags.
	args.AddFlags(pflag.CommandLine)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	if err := args.Validate(); err != nil {
		slog.Error("validate args failed", "err", err)
		os.Exit(1)
	}

	targets := func(context *generator.Context) []generator.Target {
		return getTargets(context, args)
	}

	// Run the tool.
	if err := gengo.Execute(
		getNameSystems(),
		getDefaultNameSystem(),
		targets,
		gengo.StdBuildTag,
		pflag.Args(),
	); err != nil {
		slog.Error("execute failed", "err", err)
		os.Exit(1)
	}
	slog.Info("completed successfully")
}

type Args struct {
	outputFilePrefix string
	outputDir        string
	goHeaderFile     string
}

// AddFlags adds this tool's flags to the flagset.
func (args *Args) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&args.outputFilePrefix, "output-file-prefix", "zz_generated",
		"the name of the file to be generated (without extensions)")
	fs.StringVar(&args.outputDir, "output-dir", "",
		"the output dir, default is dir of input")
	fs.StringVar(&args.goHeaderFile, "go-header-file", "",
		"the path to a file containing boilerplate header text; the string \"YEAR\" will be replaced with the current 4-digit year")
}

// Validate checks the arguments.
func (args *Args) Validate() error {
	if args.goHeaderFile == "" {
		return fmt.Errorf("--go-header-file must be specified")
	}

	return nil
}

// getNameSystems returns the name system used by the generators in this package.
func getNameSystems() namer.NameSystems {
	return namer.NameSystems{
		"raw": namer.NewRawNamer("", nil),
	}
}

// getDefaultNameSystem returns the default name system for ordering the types to be
// processed by the generators in this package.
func getDefaultNameSystem() string {
	return "raw"
}

// getTargets is called after the inputs have been loaded.  It is expected to
// examine the provided context and return a list of Packages which will be
// executed further.
func getTargets(_ *generator.Context, args *Args) []generator.Target {
	boilerplate, err := gengo.GoBoilerplate(args.goHeaderFile, gengo.StdBuildTag, gengo.StdGeneratedBy)
	if err != nil {
		slog.Error("failed loading boilerplate", "err", err)
		os.Exit(1)
	}

	targets := []generator.Target{}

	kinds := []string{
		"pd",
		"tikv",
		"tidb",
		"tiflash",
		"ticdc",
		"tso",
		"scheduling",
		"scheduler",
		"tiproxy",
		"tikvworker",
	}

	runtimeTargetPkg := "github.com/pingcap/tidb-operator/v2/pkg/runtime"
	runtimePrefix := args.outputFilePrefix + ".runtime"
	targets = append(targets, &generator.SimpleTarget{
		PkgName:       "runtime",
		PkgPath:       runtimeTargetPkg,
		PkgDir:        args.outputDir,
		HeaderComment: boilerplate,
		GeneratorsFunc: func(_ *generator.Context) []generator.Generator {
			var gs []generator.Generator
			for _, kind := range kinds {
				gs = append(gs, generators.NewRuntimeGenerator(runtimePrefix, kind, runtimeTargetPkg))
			}
			return gs
		},
	})

	scopeTargetPkg := filepath.Join(runtimeTargetPkg, "scope")
	scopePrefix := args.outputFilePrefix + ".scope"
	targets = append(targets, &generator.SimpleTarget{
		PkgName:       "scope",
		PkgPath:       scopeTargetPkg,
		PkgDir:        filepath.Join(args.outputDir, "scope"),
		HeaderComment: boilerplate,

		GeneratorsFunc: func(_ *generator.Context) []generator.Generator {
			var gs []generator.Generator
			for _, kind := range kinds {
				gs = append(gs, generators.NewScopeGenerator(scopePrefix, kind, scopeTargetPkg))
			}
			return gs
		},
	})

	return targets
}
