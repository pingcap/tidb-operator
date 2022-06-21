// Copyright 2019 PingCAP, Inc.
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

package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/monitor/monitor"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	"github.com/pingcap/tidb-operator/tests/pkg/metrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
	"k8s.io/kubernetes/test/e2e/framework/log"
)

func CheckTidbMonitor(monitor *v1alpha1.TidbMonitor, cli versioned.Interface, kubeCli kubernetes.Interface, fw portforward.PortForward) error {

	if err := checkTidbMonitorPod(monitor, kubeCli); err != nil {
		log.Logf("ERROR: tm[%s/%s] failed to check pod:%v", monitor.Namespace, monitor.Name, err)
		return err
	}
	if err := checkTidbMonitorFunctional(monitor, fw); err != nil {
		log.Logf("ERROR: tm[%s/%s] failed to check functional:%v", monitor.Namespace, monitor.Name, err)
		return err
	}
	return nil
}

func CheckTidbMonitorConfigurationUpdate(monitor *v1alpha1.TidbMonitor, kubeCli kubernetes.Interface, fw portforward.PortForward, expectActiveTargets int) error {

	if err := checkTidbMonitorPod(monitor, kubeCli); err != nil {
		log.Logf("ERROR: tm[%s/%s] failed to check pod:%v", monitor.Namespace, monitor.Name, err)
		return err
	}
	if err := checkPrometheusCommon(monitor.Name, monitor.Namespace, fw, expectActiveTargets, 0); err != nil {
		log.Logf("ERROR: tm[%s/%s]'s prometheus check error:%v", monitor.Namespace, monitor.Namespace, err)
		return err
	}
	return nil
}

// checkTidbMonitorPod check the pod of TidbMonitor whether it is ready
func checkTidbMonitorPod(tm *v1alpha1.TidbMonitor, kubeCli kubernetes.Interface) error {
	namespace := tm.Namespace

	return wait.Poll(5*time.Second, 20*time.Minute, func() (done bool, err error) {
		//validate all shard pods
		for shard := int32(0); shard < tm.GetShards(); shard++ {
			svcName := monitor.PrometheusName(tm.Name, shard)
			instanceName := monitor.GetMonitorInstanceName(tm, shard)
			monitorLabel, err := label.NewMonitor().Instance(instanceName).Monitor().Selector()
			if err != nil {
				return false, err
			}
			pods, err := kubeCli.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
				LabelSelector: monitorLabel.String(),
			})
			if err != nil {
				log.Logf("ERROR: tm[%s/%s]'s pod is failed to fetch", tm.Namespace, tm.Name)
				return false, nil
			}
			if len(pods.Items) < 1 || len(pods.Items) > 1 {
				log.Logf("ERROR: tm[%s/%s] have incorrect count[%d] of pods", tm.Namespace, tm.Name, len(pods.Items))
				return false, nil
			}
			pod := &pods.Items[0]

			if !podutil.IsPodReady(pod) {
				log.Logf("ERROR: tm[%s/%s]'s pod[%s/%s] is not ready", tm.Namespace, tm.Name, pod.Namespace, pod.Name)
				return false, nil
			}
			if tm.Spec.Grafana != nil && len(pod.Spec.Containers) != 3 && tm.Spec.Thanos == nil {
				return false, fmt.Errorf("tm[%s/%s]'s pod didn't have 3 containers with grafana enabled", tm.Namespace, tm.Name)
			}
			if tm.Spec.Grafana == nil && len(pod.Spec.Containers) != 2 && tm.Spec.Thanos == nil {
				return false, fmt.Errorf("tm[%s/%s]'s pod didnt' have 2 containers with grafana disabled", tm.Namespace, tm.Name)
			}
			log.Logf("tm[%s/%s]'s pod[%s/%s] is ready", tm.Namespace, tm.Name, pod.Namespace, pod.Name)
			_, err = kubeCli.CoreV1().Services(namespace).Get(context.TODO(), svcName, metav1.GetOptions{})
			if err != nil {
				log.Logf("ERROR: tm[%s/%s]'s service[%s/%s] failed to fetch", tm.Namespace, tm.Name, tm.Namespace, svcName)
				return false, nil
			}
		}
		return true, err
	})
}

// checkTidbMonitorFunctional check whether TidbMonitor's Prometheus and Grafana are working now
func checkTidbMonitorFunctional(tm *v1alpha1.TidbMonitor, fw portforward.PortForward) error {
	for shard := int32(0); shard < tm.GetShards(); shard++ {
		stsName := monitor.GetMonitorShardName(tm.Name, shard)
		if err := checkPrometheusCommon(tm.Name, tm.Namespace, fw, 1, shard); err != nil {
			log.Logf("ERROR: tm[%s/%s]'s prometheus check error:%v", tm.Namespace, stsName, err)
			return err
		}
		log.Logf("tidbmonitor[%s/%s]'s prometheus is ready", stsName, tm.Namespace)
		if tm.Spec.Grafana != nil {
			var grafanaClient *metrics.Client
			if _, err := checkGrafanaDataCommon(tm.Name, tm.Namespace, grafanaClient, fw, tm.Spec.DM != nil, shard); err != nil {
				log.Logf("ERROR: tm[%s/%s]'s grafana check error:%v", tm.Namespace, err)
				return err
			}
			log.Logf("tidbmonitor[%s/%s]'s grafana is ready", stsName, tm.Namespace)
		}
	}
	return nil
}

// checkPrometheusCommon check the Prometheus working status by querying `up` api and `targets` api.
func checkPrometheusCommon(name, namespace string, fw portforward.PortForward, expectActiveTargets int, shard int32) error {
	var prometheusAddr string
	tmName := monitor.GetMonitorShardName(name, shard)
	if fw != nil {
		localHost, localPort, cancel, err := portforward.ForwardOnePort(fw, namespace, fmt.Sprintf("svc/%s", monitor.PrometheusName(name, shard)), 9090)
		if err != nil {
			return err
		}
		defer cancel()
		prometheusAddr = fmt.Sprintf("%s:%d", localHost, localPort)
	} else {
		prometheusAddr = fmt.Sprintf("%s.%s:9090", monitor.PrometheusName(name, shard), namespace)
	}
	err := wait.PollImmediate(5*time.Second, 5*time.Minute, func() (done bool, err error) {
		prometheusSvc := fmt.Sprintf("http://%s/api/v1/query?query=up", prometheusAddr)
		resp, err := http.Get(prometheusSvc)
		if err != nil {
			log.Logf("ERROR: %v", err)
			return false, nil
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Logf("ERROR: %v", err)
			return false, nil
		}
		response := &struct {
			Status string `json:"status"`
		}{}
		err = json.Unmarshal(body, response)
		if err != nil {
			log.Logf("ERROR: %v", err)
			return false, nil
		}
		if response.Status != "success" {
			log.Logf("ERROR: he prometheus's api[%s] has not ready", prometheusSvc)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		log.Logf("ERROR: %v", err)
		return err
	}
	log.Logf("prometheus[%s/%s] is up", namespace, monitor.PrometheusName(name, shard))

	return wait.PollImmediate(5*time.Second, 5*time.Minute, func() (done bool, err error) {
		prometheusTargets := fmt.Sprintf("http://%s/api/v1/targets", prometheusAddr)
		targetResponse, err := http.Get(prometheusTargets)
		if err != nil {
			log.Logf("ERROR: %v", err)
			return false, nil
		}
		defer targetResponse.Body.Close()
		body, err := ioutil.ReadAll(targetResponse.Body)
		if err != nil {
			log.Logf("ERROR: %v", err)
			return false, nil
		}
		data := struct {
			Status string `json:"status"`
			Data   struct {
				ActiveTargets []struct {
					DiscoveredLabels struct {
						Job     string `json:"job"`
						PodName string `json:"__meta_kubernetes_pod_name"`
					} `json:"discoveredLabels"`
					Health string `json:"health"`
				} `json:"activeTargets"`
			} `json:"data"`
		}{}
		if err := json.Unmarshal(body, &data); err != nil {
			log.Logf("ERROR: %v", err)
			return false, nil
		}
		if data.Status != "success" || len(data.Data.ActiveTargets) < expectActiveTargets {
			log.Logf("ERROR: monitor[%s/%s]'s prometheus targets error %s, ActiveTargets:%d", namespace, name, prometheusAddr, data.Data.ActiveTargets)
			return false, nil
		}
		for _, target := range data.Data.ActiveTargets {
			log.Logf("monitor[%s/%s]'s target[%s]", namespace, tmName, target.DiscoveredLabels.PodName)
		}
		return true, nil
	})
}

// checkGrafanaDataCommon check the Grafana working status by sending a query request
func checkGrafanaDataCommon(name, namespace string, grafanaClient *metrics.Client, fw portforward.PortForward, dmMonitor bool, shard int32) (*metrics.Client, error) {
	svcName := monitor.GrafanaName(name, shard)

	var addr string
	if fw != nil {
		localHost, localPort, cancel, err := portforward.ForwardOnePort(fw, namespace, fmt.Sprintf("svc/%s", svcName), 3000)
		if err != nil {
			return nil, err
		}
		defer cancel()
		addr = fmt.Sprintf("%s:%d", localHost, localPort)
	} else {
		addr = fmt.Sprintf("%s.%s.svc.cluster.local:3000", svcName, namespace)
	}

	var datasourceID int
	err := wait.Poll(5*time.Second, 5*time.Minute, func() (done bool, err error) {
		datasourceID, err = getDatasourceID(addr)
		if err != nil {
			log.Logf("ERROR: %v", err)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	err = wait.PollImmediate(5*time.Second, 20*time.Minute, func() (done bool, err error) {

		end := time.Now()
		start := end.Add(-time.Minute)
		values := url.Values{}
		if dmMonitor {
			values.Set("query", "sum by (worker) (dm_master_worker_state)")
		} else {
			values.Set("query", "histogram_quantile(0.999, sum(rate(tidb_server_handle_query_duration_seconds_bucket[1m])) by (le))")
		}
		values.Set("start", fmt.Sprintf("%d", start.Unix()))
		values.Set("end", fmt.Sprintf("%d", end.Unix()))
		values.Set("step", "30")
		u := fmt.Sprintf("http://%s/api/datasources/proxy/%d/api/v1/query_range?%s", addr, datasourceID, values.Encode())
		log.Logf("tm[%s/%s]'s grafana query url is %s", namespace, monitor.GetMonitorShardName(name, shard), u)

		req, err := http.NewRequest(http.MethodGet, u, nil)
		if err != nil {
			return false, nil
		}
		req.SetBasicAuth(grafanaUsername, grafanaPassword)
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Logf("ERROR: tm[%s/%s]'s grafana response error:%v", namespace, monitor.GetMonitorShardName(name, shard), err)
			return false, nil
		}
		defer resp.Body.Close()
		buf, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return false, nil
		}
		data := struct {
			Status string `json:"status"`
			Data   struct {
				ResultType string `json:"resultType"`
				Result     []struct {
					Metric struct {
						Job string `json:"job"`
					} `json:"metric"`
					Values []interface{} `json:"values"`
				} `json:"result"`
			}
		}{}
		if err := json.Unmarshal(buf, &data); err != nil {
			return false, nil
		}
		if data.Status != "success" || len(data.Data.Result) < 1 {
			log.Logf("ERROR: tm[%s/%s]'s invalid response: status: %s, result: %v", namespace, monitor.GetMonitorShardName(name, shard), data.Status, data.Data.Result)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	// Grafana ready, init grafana client, no more sync logic because race condition is okay here
	if grafanaClient == nil {
		grafanaURL := fmt.Sprintf("http://%s.%s:3000", svcName, namespace)
		client, err := metrics.NewClient(grafanaURL, grafanaUsername, grafanaPassword)
		if err != nil {
			return nil, err
		}
		return client, nil
	}
	return nil, nil
}

// CheckThanosQueryData check the Thanos Query working status by querying `up` api and `targets` api.
func CheckThanosQueryData(name, namespace string, fw portforward.PortForward, expectNumber int) error {
	var thanosAddr string
	if fw != nil {
		localHost, localPort, cancel, err := portforward.ForwardOnePort(fw, namespace, fmt.Sprintf("svc/%s", name), 9090)
		if err != nil {
			return err
		}
		defer cancel()
		thanosAddr = fmt.Sprintf("%s:%d", localHost, localPort)
	} else {
		thanosAddr = fmt.Sprintf("%s.%s:9090", "thanos-query", namespace)
	}

	return wait.PollImmediate(5*time.Second, 10*time.Minute, func() (done bool, err error) {

		storesUrl := fmt.Sprintf("http://%s/api/v1/stores", thanosAddr)
		storesResponse, err := http.Get(storesUrl)
		if err != nil {
			log.Logf("ERROR: %v", err)
			return false, nil
		}
		defer storesResponse.Body.Close()
		storesBody, err := ioutil.ReadAll(storesResponse.Body)
		if err != nil {
			log.Logf("ERROR: %v", err)
			return false, nil
		}
		storeData := struct {
			Status string `json:"status"`
			Data   struct {
				Receive []map[string]interface{} `json:"receive"`
			} `json:"data"`
		}{}
		if err := json.Unmarshal(storesBody, &storeData); err != nil {
			log.Logf("ERROR: %v", err)
			return false, nil
		}
		if storeData.Status != "success" || len(storeData.Data.Receive) != 1 {
			log.Logf("ERROR: thanos[%s/%s]'s stores error %s, store:%v , status: %s", namespace, name, thanosAddr, storeData.Data.Receive, storeData.Status)
			return false, nil
		}
		metrcis := fmt.Sprintf("http://%s/api/v1/query?query=up", thanosAddr)
		instanceUpResponse, err := http.Get(metrcis)
		if err != nil {
			log.Logf("ERROR: %v", err)
			return false, nil
		}
		defer instanceUpResponse.Body.Close()
		instanceUpResponseBody, err := ioutil.ReadAll(instanceUpResponse.Body)
		if err != nil {
			log.Logf("ERROR: %v", err)
			return false, nil
		}
		instanceUpData := struct {
			Status string `json:"status"`
			Data   struct {
				Result []map[string]interface{} `json:"result"`
			} `json:"data"`
		}{}
		if err := json.Unmarshal(instanceUpResponseBody, &instanceUpData); err != nil {
			log.Logf("ERROR: %v", err)
			return false, nil
		}
		if instanceUpData.Status != "success" || len(instanceUpData.Data.Result) != expectNumber {
			log.Logf("ERROR: thanos[%s/%s]'s targets error %s, metrics data:%v , status: %s", namespace, name, thanosAddr, instanceUpData.Data.Result, instanceUpData.Status)
			return false, nil
		}
		for _, target := range instanceUpData.Data.Result {
			log.Logf("thanos[%s/%s]'s target[%s]", namespace, name, target)
		}
		return true, nil
	})
}
