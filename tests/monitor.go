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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	"github.com/pingcap/tidb-operator/tests/pkg/metrics"
	"github.com/pingcap/tidb-operator/tests/slack"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
	"net/http"
	"net/url"
	"time"
)

func (oa *operatorActions) CheckTidbMonitor(monitor *v1alpha1.TidbMonitor) error {

	if err := oa.checkTidbMonitorPod(monitor); err != nil {
		klog.Errorf("tm[%s/%s] failed to check pod:%v", monitor.Namespace, monitor.Name, err)
		return err
	}
	if err := oa.checkTidbMonitorFunctional(monitor); err != nil {
		klog.Errorf("tm[%s/%s] failed to check functional:%v", monitor.Namespace, monitor.Name, err)
		return err
	}
	return nil
}

func (oa *operatorActions) CheckTidbMonitorOrDie(monitor *v1alpha1.TidbMonitor) {
	if err := oa.CheckTidbMonitor(monitor); err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (oa *operatorActions) checkTidbMonitorPod(tm *v1alpha1.TidbMonitor) error {
	namespace := tm.Namespace
	svcName := fmt.Sprintf("%s-prometheus", tm.Name)
	monitorLabel, err := label.NewMonitor().Instance(tm.Name).Monitor().Selector()
	if err != nil {
		return err
	}

	return wait.Poll(5*time.Second, 20*time.Minute, func() (done bool, err error) {

		pods, err := oa.kubeCli.CoreV1().Pods(namespace).List(metav1.ListOptions{
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
		_, err = oa.kubeCli.CoreV1().Services(namespace).Get(svcName, metav1.GetOptions{})
		if err != nil {
			klog.Infof("tm[%s/%s]'s service[%s/%s] failed to fetch", tm.Namespace, tm.Name, tm.Namespace, svcName)
			return false, nil
		}
		return true, err
	})
}

func (oa *operatorActions) checkTidbMonitorFunctional(monitor *v1alpha1.TidbMonitor) error {
	if err := oa.checkPrometheusCommon(monitor.Name, monitor.Namespace); err != nil {
		klog.Errorf("tm[%s/%s]'s prometheus check error:%v", monitor.Namespace, monitor.Namespace, err)
		return err
	}
	klog.Infof("tidbmonitor[%s/%s]'s prometheus is ready", monitor.Name, monitor.Namespace)
	if monitor.Spec.Grafana != nil {
		var grafanaClient *metrics.Client
		if _, err := oa.checkGrafanaDataCommon(monitor.Name, monitor.Namespace, grafanaClient); err != nil {
			klog.Errorf("tm[%s/%s]'s grafana check error:%v", monitor.Namespace, monitor.Namespace, err)
			return err
		}
		klog.Infof("tidbmonitor[%s/%s]'s grafana is ready", monitor.Name, monitor.Namespace)
	}
	return nil
}

func (oa *operatorActions) checkPrometheusCommon(name, namespace string) error {
	var prometheusAddr string
	if oa.fw != nil {
		localHost, localPort, cancel, err := portforward.ForwardOnePort(oa.fw, namespace, fmt.Sprintf("svc/%s-prometheus", name), 9090)
		if err != nil {
			return err
		}
		defer cancel()
		prometheusAddr = fmt.Sprintf("%s:%d", localHost, localPort)
	} else {
		prometheusAddr = fmt.Sprintf("%s-prometheus.%s:9090", name, namespace)
	}
	prometheusSvc := fmt.Sprintf("http://%s/api/v1/query?query=up", prometheusAddr)
	resp, err := http.Get(prometheusSvc)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	response := &struct {
		Status string `json:"status"`
	}{}
	err = json.Unmarshal(body, response)
	if err != nil {
		return err
	}
	if response.Status != "success" {
		return fmt.Errorf("the prometheus's api[%s] has not ready", prometheusSvc)
	}
	return nil
}

func (oa *operatorActions) checkGrafanaDataCommon(name, namespace string, grafanaClient *metrics.Client) (*metrics.Client, error) {
	svcName := fmt.Sprintf("%s-grafana", name)
	end := time.Now()
	start := end.Add(-5 * time.Minute)
	values := url.Values{}
	values.Set("query", "histogram_quantile(0.999, sum(rate(tidb_server_handle_query_duration_seconds_bucket[1m])) by (le))")
	values.Set("start", fmt.Sprintf("%d", start.Unix()))
	values.Set("end", fmt.Sprintf("%d", end.Unix()))
	values.Set("step", "30")

	var addr string
	if oa.fw != nil {
		localHost, localPort, cancel, err := portforward.ForwardOnePort(oa.fw, namespace, fmt.Sprintf("svc/%s-grafana", name), 3000)
		if err != nil {
			return nil, err
		}
		defer cancel()
		addr = fmt.Sprintf("%s:%d", localHost, localPort)
	} else {
		addr = fmt.Sprintf("%s.%s.svc.cluster.local:3000", svcName, namespace)
	}

	datasourceID, err := getDatasourceID(addr)
	if err != nil {
		return nil, err
	}

	u := fmt.Sprintf("http://%s/api/datasources/proxy/%d/api/v1/query_range?%s", addr, datasourceID, values.Encode())
	klog.Infof("tm[%s/%s]'s grafana query url is %s", namespace, name, u)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(grafanaUsername, grafanaPassword)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		klog.Errorf("tm[%s/%s]'s grafana response error:%v", namespace, name, err)
		return nil, err
	}
	defer resp.Body.Close()
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	klog.Infof("tm[%s/%s]'s grafana response:%s", namespace, name, buf)
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
		return nil, err
	}
	if data.Status != "success" || len(data.Data.Result) < 1 {
		return nil, fmt.Errorf("invalid response: status: %s, result: %v", data.Status, data.Data.Result)
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
