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

package kubefeat

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/prometheus/common/expfmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var defaultFeatureGates Gates

func Stage(key Feature) StagedFeature {
	if defaultFeatureGates == nil {
		panic("please init featureGates before use it")
	}

	return defaultFeatureGates.Stage(key)
}

type Gates interface {
	Stage(key Feature) StagedFeature
}

type StagedFeature interface {
	Enabled(FeatureStage) bool
}

type featureGates struct {
	feats map[Feature]spec
}

type FeatureStage int

const (
	INVAILD FeatureStage = 0

	ALPHA FeatureStage = 1 << iota
	BETA
	STABLE
	ANY = ALPHA | BETA | STABLE
)

func stageFromString(s string) FeatureStage {
	switch s {
	case "ALPHA":
		return ALPHA
	case "BETA":
		return BETA
	case "":
		return STABLE
	}

	return INVAILD
}

type spec struct {
	enabled bool
	stage   FeatureStage
}

func (s spec) Enabled(stage FeatureStage) bool {
	if s.stage&stage == 0 {
		return false
	}
	return s.enabled
}

func (g *featureGates) Stage(key Feature) StagedFeature {
	return g.feats[key]
}

func MustInitFeatureGates(cfg *rest.Config) {
	gates, err := NewFeatureGates(cfg)
	if err != nil {
		// TODO: use a common panic util to panic
		panic(err)
	}

	fmt.Println("init feature gates")

	defaultFeatureGates = gates
}

func NewFeatureGates(cfg *rest.Config) (Gates, error) {
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("cannot new client: %w", err)
	}

	//nolint:mnd // refactor to a constant if needed
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	metricsPath := "/metrics"
	resp, err := clientset.RESTClient().Get().RequestURI(metricsPath).DoRaw(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	return parseFeatureGates(resp)
}

func parseFeatureGates(metrics []byte) (Gates, error) {
	// Parse the metrics using Prometheus expfmt
	parser := expfmt.TextParser{}
	metricFamilies, err := parser.TextToMetricFamilies(bytes.NewReader(metrics))
	if err != nil {
		return nil, fmt.Errorf("failed to parse metrics: %w", err)
	}

	featureGatesMetric, ok := metricFamilies["kubernetes_feature_enabled"]
	if !ok {
		// this metric is supported after v1.26
		// TODO: fix it if we hope to support the version before v1.26
		return nil, fmt.Errorf("no kubernetes_feature_enabled metric")
	}

	gates := &featureGates{
		feats: map[Feature]spec{},
	}

	for _, metric := range featureGatesMetric.GetMetric() {
		if metric.GetGauge().GetValue() == 1 {
			feat := spec{
				enabled: true,
			}
			name := ""
			for _, label := range metric.GetLabel() {
				switch label.GetName() {
				case "name":
					name = label.GetValue()
				case "stage":
					feat.stage = stageFromString(label.GetValue())
				}
			}
			if name != "" {
				gates.feats[Feature(name)] = feat
			}
		}
	}

	return gates, nil
}
