package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/cmd/collector/pkg/collect"
	"github.com/pingcap/tidb-operator/cmd/collector/pkg/config"
	"github.com/pingcap/tidb-operator/cmd/collector/pkg/dump"
)

func main() {
	var (
		kubeconfig string
		configPath string
		outputPath string
	)
	flag.StringVar(&kubeconfig, "kubeconfig", "", "kubeconfig path")
	flag.StringVar(&configPath, "config", "", "config path")
	flag.StringVar(&outputPath, "output", "", "output path")
	flag.Parse()

	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err)
	}

	cli, err := client.New(cfg, client.Options{Scheme: collect.GetScheme()})
	if err != nil {
		panic(err)
	}

	collectorCfg := config.CollectorConfig{}
	configContents, err := ioutil.ReadFile(configPath)
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(configContents, &collectorCfg)
	if err != nil {
		panic(err)
	}

	// TODO: make encoder and dumper configurable
	var (
		// encoderType encode.Encoding = encode.YAML
		encoding                = "yaml"
		dumperType dump.Dumping = dump.Zip
		dumper     dump.Dumper
		collectors []collect.Collector
		printer    printers.ResourcePrinter
	)

	switch dumperType {
	case dump.Zip:
		dumper, err = dump.NewZipDumper(outputPath)
		if err != nil {
			panic(err)
		}
	}

	switch encoding {
	case "yaml":
		printer = &printers.TypeSetterPrinter{
			Delegate: &printers.YAMLPrinter{},
			Typer:    collect.GetScheme(),
		}
	}
	// Set types for the objects
	printer, err = printers.NewTypeSetter(collect.GetScheme()).WrapToPrinter(printer, nil)
	if err != nil {
		panic(err)
	}

	collectors = config.ConfigCollectors(cli, collectorCfg)

	err = dumpFiles(collect.MergedCollectors(collectors...), dumper, printer, encoding)
	if err != nil {
		panic(err)
	}
}

// dumpFiles dumps the encoded contents to dumper.
func dumpFiles(c collect.Collector, dumper dump.Dumper, printer printers.ResourcePrinter, encoding string) error {
	defer dumper.Close()()

	objCh, err := c.Objects()
	if err != nil {
		return err
	}

	for obj := range objCh {
		// TODO: remove hack after https://github.com/kubernetes/kubernetes/issues/3030 is resolved
		gvks, _, err := collect.GetScheme().ObjectKinds(obj)
		if err != nil {
			return err
		}
		gvk := schema.GroupVersionKind{}
		for _, g := range gvks {
			if g.Empty() {
				continue
			}
			gvk = g
		}
		kind, ns, name := gvk.Kind, obj.GetNamespace(), obj.GetName()
		path := fmt.Sprintf("%s.%s", filepath.Join(kind, ns, name), encoding)
		writer, err := dumper.Open(path)
		if err != nil {
			return err
		}

		// omit managed fields
		if a, err := meta.Accessor(obj); err == nil {
			a.SetManagedFields(nil)
		}

		err = printer.PrintObj(obj, writer)
		if err != nil {
			return err
		}
	}

	return nil
}
