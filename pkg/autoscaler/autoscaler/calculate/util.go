package calculate

import (
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
)

// currently, we only choose one metrics to be computed.
// If there exists several metrics, we tend to choose ResourceMetricSourceType metric
func FilterMetrics(metrics []autoscalingv2beta2.MetricSpec) autoscalingv2beta2.MetricSpec {
	for _, m := range metrics {
		if m.Type == autoscalingv2beta2.ResourceMetricSourceType && m.Resource != nil {
			return m
		}
	}
	return metrics[0]
}

// genMetricType return the supported MetricType in Operator by kubernetes auto-scaling MetricType
func GenMetricType(tac *v1alpha1.TidbClusterAutoScaler, metric autoscalingv2beta2.MetricSpec) (MetricType, error) {
	if metric.Type == autoscalingv2beta2.ResourceMetricSourceType && metric.Resource != nil && metric.Resource.Name == corev1.ResourceCPU {
		return MetricTypeCPU, nil
	}
	return "", fmt.Errorf(InvalidTacMetricConfigureMsg, tac.Namespace, tac.Name)
}

func filterContainer(tac *v1alpha1.TidbClusterAutoScaler, sts *appsv1.StatefulSet, contarinerName string) (*corev1.Container, error) {
	for _, c := range sts.Spec.Template.Spec.Containers {
		if c.Name == contarinerName {
			return &c, nil
		}
	}
	return nil, fmt.Errorf("tac[%s/%s]'s Target Tidb have not tidb container", tac.Namespace, tac.Name)
}

func extractCpuRequestsRadio(c *corev1.Container) (float64, error) {
	if c.Resources.Requests.Cpu() == nil {
		return 0, fmt.Errorf("container[%s] cpu requests is empty", c.Name)
	}
	return float64(c.Resources.Requests.Cpu().MilliValue()) / 1000.0, nil
}
