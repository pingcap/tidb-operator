package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"gopkg.in/yaml.v3"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/cmd/collector/pkg/collect"
	"github.com/pingcap/tidb-operator/cmd/collector/pkg/config"
	"github.com/pingcap/tidb-operator/cmd/collector/pkg/dump"
	"github.com/pingcap/tidb-operator/cmd/collector/pkg/encode"
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
		encoderType encode.Encoding = encode.YAML
		dumperType  dump.Dumping    = dump.Zip
		encoder     encode.Encoder
		dumper      dump.Dumper
		collectors  []collect.Collector
	)

	switch encoderType {
	case encode.YAML:
		encoder = encode.NewYAMLEncoder()
	default:
		panic(fmt.Errorf("unknown encoder type: %v", encoderType))
	}

	switch dumperType {
	case dump.Zip:
		dumper, err = dump.NewZipDumper(outputPath)
		if err != nil {
			panic(err)
		}
	}

	collectors = config.ConfigCollectors(cli, collectorCfg)

	err = dumpFiles(collect.MergedCollectors(collectors...), encoder, dumper)
	if err != nil {
		panic(err)
	}
}

// dumpFiles dumps the encoded contents to dumper.
func dumpFiles(c collect.Collector, encoder encode.Encoder, dumper dump.Dumper) error {
	defer dumper.Close()()

	objCh, err := c.Objects()
	if err != nil {
		return err
	}

	for obj := range objCh {
		kind, ns, name := obj.GetObjectKind().GroupVersionKind().Kind, obj.GetNamespace(), obj.GetName()
		path := fmt.Sprintf("%s.%s", filepath.Join("/", kind, ns, name), encoder.Extension())
		contents, err := encoder.Encode(obj)
		if err != nil {
			return err
		}
		dumper.Write(path, contents)
	}

	return nil
}
