package calculate

import (
	"fmt"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	promClient "github.com/prometheus/client_golang/api"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
)

func CalculateCpuMetrics(tac *v1alpha1.TidbClusterAutoScaler, sts *appsv1.StatefulSet,
	client promClient.Client, instances []string, metric autoscalingv2beta2.MetricSpec,
	queryPattern, timeWindow string, memberType v1alpha1.MemberType) (int32, error) {
	if metric.Resource == nil || metric.Resource.Target.AverageUtilization == nil {
		return 0, fmt.Errorf(InvalidTacMetricConfigureMsg, tac.Namespace, tac.Name)
	}
	currentReplicas := len(instances)
	c, err := filterContainer(tac, sts, memberType.String())
	if err != nil {
		return 0, err
	}
	cpuRequestsRadio, err := extractCpuRequestsRadio(c)
	if err != nil {
		return 0, err
	}
	now := time.Now()
	duration, err := time.ParseDuration(timeWindow)
	if err != nil {
		return 0, err
	}
	prvious := now.Truncate(duration)
	r := &Response{}
	err = queryMetricsFromPrometheus(tac, fmt.Sprintf(queryPattern, tac.Spec.Cluster.Name), client, now.Unix(), r)
	if err != nil {
		return 0, err
	}
	sum1, err := sumByInstanceFromResponse(instances, r)
	if err != nil {
		return 0, err
	}
	err = queryMetricsFromPrometheus(tac, fmt.Sprintf(queryPattern, tac.Spec.Cluster.Name), client, prvious.Unix(), r)
	if err != nil {
		return 0, err
	}
	sum2, err := sumByInstanceFromResponse(instances, r)
	if err != nil {
		return 0, err
	}
	if sum1-sum2 < 0 {
		return 0, fmt.Errorf(CpuSumMetricsErrorMsg, tac.Namespace, tac.Name, timeWindow)
	}
	cpuSecsTotal := sum1 - sum2
	durationSeconds := duration.Seconds()
	utilizationRadio := float64(*metric.Resource.Target.AverageUtilization) / 100.0
	expectedCpuSecsTotal := cpuRequestsRadio * durationSeconds * float64(currentReplicas) * utilizationRadio
	rc, err := calculate(cpuSecsTotal, expectedCpuSecsTotal, int32(currentReplicas))
	if err != nil {
		return 0, err
	}
	return rc, nil
}
