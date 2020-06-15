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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	"github.com/pingcap/tidb-operator/tests/pkg/metrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
)

func CheckTidbMonitor(monitor *v1alpha1.TidbMonitor, cli versioned.Interface, kubeCli kubernetes.Interface, fw portforward.PortForward) error {

	if err := checkTidbMonitorPod(monitor, kubeCli); err != nil {
		klog.Errorf("tm[%s/%s] failed to check pod:%v", monitor.Namespace, monitor.Name, err)
		return err
	}
	if err := checkTidbMonitorFunctional(monitor, fw); err != nil {
		klog.Errorf("tm[%s/%s] failed to check functional:%v", monitor.Namespace, monitor.Name, err)
		return err
	}
	return checkTidbClusterStatus(monitor, cli)
}

func checkTidbClusterStatus(tm *v1alpha1.TidbMonitor, cli versioned.Interface) error {
	return wait.Poll(5*time.Second, 5*time.Minute, func() (done bool, err error) {
		tcRef := tm.Spec.Clusters[0]
		tc, err := cli.PingcapV1alpha1().TidbClusters(tcRef.Namespace).Get(tcRef.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if tc.Status.Monitor == nil {
			return false, nil
		}
		if tc.Status.Monitor.Name != tm.Name || tc.Status.Monitor.Namespace != tm.Namespace {
			return false, fmt.Errorf("tidbcluster's monitorRef status is wrong")
		}
		return true, nil
	})
}

// checkTidbMonitorPod check the pod of TidbMonitor whether it is ready
func checkTidbMonitorPod(tm *v1alpha1.TidbMonitor, kubeCli kubernetes.Interface) error {
	namespace := tm.Namespace
	svcName := fmt.Sprintf("%s-prometheus", tm.Name)
	monitorLabel, err := label.NewMonitor().Instance(tm.Name).Monitor().Selector()
	if err != nil {
		return err
	}

	return wait.Poll(5*time.Second, 20*time.Minute, func() (done bool, err error) {

		pods, err := kubeCli.CoreV1().Pods(namespace).List(metav1.ListOptions{
			LabelSelector: monitorLabel.String(),
		})
		if err != nil {
			klog.Infof("tm[%s/%s]'s pod is failed to fetch", tm.Namespace, tm.Name)
			return false, nil
		}
		if len(pods.Items) < 1 || len(pods.Items) > 1 {
			klog.Infof("tm[%s/%s] have incorrect count[%d] of pods", tm.Namespace, tm.Name, len(pods.Items))
			return false, nil
		}
		pod := &pods.Items[0]

		if !podutil.IsPodReady(pod) {
			klog.Infof("tm[%s/%s]'s pod[%s/%s] is not ready", tm.Namespace, tm.Name, pod.Namespace, pod.Name)
			return false, nil
		}
		if tm.Spec.Grafana != nil && len(pod.Spec.Containers) != 3 {
			return false, fmt.Errorf("tm[%s/%s]'s pod didn't have 3 containers with grafana enabled", tm.Namespace, tm.Name)
		}
		if tm.Spec.Grafana == nil && len(pod.Spec.Containers) != 2 {
			return false, fmt.Errorf("tm[%s/%s]'s pod didnt' have 2 containers with grafana disabled", tm.Namespace, tm.Name)
		}
		klog.Infof("tm[%s/%s]'s pod[%s/%s] is ready", tm.Namespace, tm.Name, pod.Namespace, pod.Name)
		_, err = kubeCli.CoreV1().Services(namespace).Get(svcName, metav1.GetOptions{})
		if err != nil {
			klog.Infof("tm[%s/%s]'s service[%s/%s] failed to fetch", tm.Namespace, tm.Name, tm.Namespace, svcName)
			return false, nil
		}
		return true, err
	})
}

// checkTidbMonitorFunctional check whether TidbMonitor's Prometheus and Grafana are working now
func checkTidbMonitorFunctional(monitor *v1alpha1.TidbMonitor, fw portforward.PortForward) error {
	if err := checkPrometheusCommon(monitor.Name, monitor.Namespace, fw); err != nil {
		klog.Errorf("tm[%s/%s]'s prometheus check error:%v", monitor.Namespace, monitor.Namespace, err)
		return err
	}
	klog.Infof("tidbmonitor[%s/%s]'s prometheus is ready", monitor.Name, monitor.Namespace)
	if monitor.Spec.Grafana != nil {
		var grafanaClient *metrics.Client
		if _, err := checkGrafanaDataCommon(monitor.Name, monitor.Namespace, grafanaClient, fw); err != nil {
			klog.Errorf("tm[%s/%s]'s grafana check error:%v", monitor.Namespace, monitor.Namespace, err)
			return err
		}
		klog.Infof("tidbmonitor[%s/%s]'s grafana is ready", monitor.Name, monitor.Namespace)
	}
	return nil
}

// checkPrometheusCommon check the Prometheus working status by querying `up` api and `targets` api.
func checkPrometheusCommon(name, namespace string, fw portforward.PortForward) error {
	var prometheusAddr string
	if fw != nil {
		localHost, localPort, cancel, err := portforward.ForwardOnePort(fw, namespace, fmt.Sprintf("svc/%s-prometheus", name), 9090)
		if err != nil {
			return err
		}
		defer cancel()
		prometheusAddr = fmt.Sprintf("%s:%d", localHost, localPort)
	} else {
		prometheusAddr = fmt.Sprintf("%s-prometheus.%s:9090", name, namespace)
	}
	err := wait.PollImmediate(5*time.Second, 5*time.Minute, func() (done bool, err error) {
		prometheusSvc := fmt.Sprintf("http://%s/api/v1/query?query=up", prometheusAddr)
		resp, err := http.Get(prometheusSvc)
		if err != nil {
			klog.Error(err.Error())
			return false, nil
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			klog.Error(err.Error())
			return false, nil
		}
		response := &struct {
			Status string `json:"status"`
		}{}
		err = json.Unmarshal(body, response)
		if err != nil {
			klog.Error(err.Error())
			return false, nil
		}
		if response.Status != "success" {
			klog.Errorf("the prometheus's api[%s] has not ready", prometheusSvc)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		klog.Error(err.Error())
		return err
	}
	klog.Infof("prometheus[%s/%s] is up", namespace, name)

	return wait.PollImmediate(5*time.Second, 5*time.Minute, func() (done bool, err error) {
		prometheusTargets := fmt.Sprintf("http://%s/api/v1/targets", prometheusAddr)
		targetResponse, err := http.Get(prometheusTargets)
		if err != nil {
			klog.Error(err.Error())
			return false, nil
		}
		defer targetResponse.Body.Close()
		body, err := ioutil.ReadAll(targetResponse.Body)
		if err != nil {
			klog.Error(err.Error())
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
			klog.Error(err.Error())
			return false, nil
		}
		if data.Status != "success" || len(data.Data.ActiveTargets) < 1 {
			klog.Errorf("monitor[%s/%s]'s prometheus targets error", namespace, name)
			return false, nil
		}
		for _, target := range data.Data.ActiveTargets {
			klog.Infof("monitor[%s/%s]'s target[%s]", namespace, name, target.DiscoveredLabels.PodName)
		}
		return true, nil
	})
}

// checkGrafanaDataCommon check the Grafana working status by sending a query request
func checkGrafanaDataCommon(name, namespace string, grafanaClient *metrics.Client, fw portforward.PortForward) (*metrics.Client, error) {
	svcName := fmt.Sprintf("%s-grafana", name)

	var addr string
	if fw != nil {
		localHost, localPort, cancel, err := portforward.ForwardOnePort(fw, namespace, fmt.Sprintf("svc/%s-grafana", name), 3000)
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
			klog.Error(err)
			return false, err
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
		values.Set("query", "histogram_quantile(0.999, sum(rate(tidb_server_handle_query_duration_seconds_bucket[1m])) by (le))")
		values.Set("start", fmt.Sprintf("%d", start.Unix()))
		values.Set("end", fmt.Sprintf("%d", end.Unix()))
		values.Set("step", "30")
		u := fmt.Sprintf("http://%s/api/datasources/proxy/%d/api/v1/query_range?%s", addr, datasourceID, values.Encode())
		klog.Infof("tm[%s/%s]'s grafana query url is %s", namespace, name, u)

		req, err := http.NewRequest(http.MethodGet, u, nil)
		if err != nil {
			return false, nil
		}
		req.SetBasicAuth(grafanaUsername, grafanaPassword)
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			klog.Errorf("tm[%s/%s]'s grafana response error:%v", namespace, name, err)
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
			klog.Errorf("invalid response: status: %s, result: %v", data.Status, data.Data.Result)
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
