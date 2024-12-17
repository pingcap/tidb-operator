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

	"github.com/spf13/pflag"
	_ "k8s.io/api/core/v1"
	_ "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/gengo/v2"
	"k8s.io/gengo/v2/generator"
	"k8s.io/gengo/v2/namer"
	"k8s.io/gengo/v2/types"

	"github.com/pingcap/tidb-operator/cmd/overlay-gen/generators"
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
	outputFile   string
	outputDir    string
	goHeaderFile string
}

// AddFlags adds this tool's flags to the flagset.
func (args *Args) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&args.outputFile, "output-file", "zz_generated.overlay",
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
func getTargets(c *generator.Context, args *Args) []generator.Target {
	boilerplate, err := gengo.GoBoilerplate(args.goHeaderFile, gengo.StdBuildTag, gengo.StdGeneratedBy)
	if err != nil {
		slog.Error("failed loading boilerplate", "err", err)
		os.Exit(1)
	}

	targetPackage := "github.com/pingcap/tidb-operator/pkg/overlay"
	targets := []generator.Target{}
	for _, input := range c.Inputs {
		slog.Info("processing", "pkg", input)

		pkg := c.Universe[input]
		outputDir := pkg.Dir
		if args.outputDir != "" {
			outputDir = args.outputDir
		}

		targets = append(targets, &generator.SimpleTarget{
			PkgName:       "overlay",
			PkgPath:       targetPackage,
			PkgDir:        outputDir,
			HeaderComment: boilerplate,

			// FilterFunc returns true if this Package cares about this type.
			// Each Generator has its own Filter method which will be checked
			// subsequently.  This will be called for every type in every
			// loaded package, not just things in our inputs.
			FilterFunc: func(_ *generator.Context, t *types.Type) bool {
				return t.Name.Package == pkg.Path
			},

			// GeneratorsFunc returns a list of Generators, each of which is
			// responsible for a single output file (though multiple generators
			// may write to the same one).
			GeneratorsFunc: func(_ *generator.Context) []generator.Generator {
				return []generator.Generator{
					generators.NewOverlayGenerator(args.outputFile+".go", targetPackage),
					generators.NewOverlayTestGenerator(args.outputFile+"_test.go", targetPackage),
				}
			},
		})
	}

	return targets
}
