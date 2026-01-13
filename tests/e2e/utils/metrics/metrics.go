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

package metrics

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/prometheus/common/expfmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/pingcap/tidb-operator/v2/tests/e2e/utils/portforwarder"
)

const (
	// OperatorPodLabelSelector is the label selector for operator pods
	OperatorPodLabelSelector = "app.kubernetes.io/component=controller"
	// OperatorMetricsPort is the port where operator exposes metrics
	OperatorMetricsPort = 8080
)

// CheckPanicMetrics checks the operator panic metrics and returns the panic count.
// It retrieves the operator pod, port forwards to its metrics endpoint, fetches the metrics,
// and parses the panic count from the Prometheus metrics.
func CheckPanicMetrics(ctx context.Context, restConfig *rest.Config, ns, deploy string) error {
	// get operator pod
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	d, err := clientset.AppsV1().Deployments(ns).Get(
		ctx, deploy, metav1.GetOptions{})
	if err != nil {
		return err
	}

	pods, err := clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(d.Spec.Selector),
	})
	if err != nil {
		return fmt.Errorf("failed to list operator pods: %w", err)
	}
	if len(pods.Items) == 0 {
		return fmt.Errorf("no operator pod found in namespace %s with label %s", ns, metav1.FormatLabelSelector(d.Spec.Selector))
	}

	pf := portforwarder.New(restConfig)

	for i := range pods.Items {
		pod := &pods.Items[i]
		panicCount, err := checkPodMetrics(ctx, pf, pod)
		if err != nil {
			return fmt.Errorf("cannot get metrics of pod %v: %w", pod.Name, err)
		}

		if panicCount > 0 {
			return fmt.Errorf("panic count %v is greater than 0", panicCount)
		}
	}

	return nil
}

func checkPodMetrics(ctx context.Context, pf portforwarder.PortForwarder, pod *corev1.Pod) (float64, error) {
	// port forward to operator pod metrics port
	forwarded, err := pf.ForwardPod(ctx, pod, fmt.Sprintf(":%d", OperatorMetricsPort))
	if err != nil {
		return 0, fmt.Errorf("failed to port forward to operator pod: %w", err)
	}
	defer forwarded.Cancel()

	// get local port
	localPort, ok := forwarded.Local(OperatorMetricsPort)
	if !ok {
		return 0, fmt.Errorf("failed to get local port for metrics")
	}

	// fetch metrics
	metricsURL := fmt.Sprintf("http://localhost:%d/metrics", localPort)
	req, err := http.NewRequestWithContext(ctx, "GET", metricsURL, nil)
	if err != nil {
		return 0, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch metrics from %s: %w", metricsURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("metrics endpoint returned status %d", resp.StatusCode)
	}

	// parse prometheus metrics
	panicCount, err := parseOperatorPanicMetric(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to parse panic metric: %w", err)
	}

	return panicCount, nil
}

// parseOperatorPanicMetric parses Prometheus metrics and extracts the panic count.
// It looks for metrics containing "panic_total" in their name and returns the counter/gauge value.
// If the metric is not found, it returns 0 (indicating no panics have occurred).
func parseOperatorPanicMetric(metricsReader io.Reader) (float64, error) {
	parser := expfmt.TextParser{}
	metricFamilies, err := parser.TextToMetricFamilies(metricsReader)
	if err != nil {
		return 0, fmt.Errorf("failed to parse metrics: %w", err)
	}

	// Look for tidb_operator_controller_panic_total metric
	for name, mf := range metricFamilies {
		if strings.Contains(name, "panic_total") {
			for _, metric := range mf.GetMetric() {
				if metric.Counter != nil {
					return metric.Counter.GetValue(), nil
				}
				if metric.Gauge != nil {
					return metric.Gauge.GetValue(), nil
				}
			}
		}
	}

	// If metric not found, it means no panics have occurred (metric not yet emitted)
	return 0, fmt.Errorf("no metrics emitted")
}
