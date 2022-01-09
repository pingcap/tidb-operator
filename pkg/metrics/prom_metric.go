package metrics

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"k8s.io/klog/v2"
	"net/http"
	"strconv"
	"sync"
)

type MetricCache struct {
	TiKVFlowBytesRateMap map[string]*sync.Map
	TiKVQPSMap           map[string]*sync.Map
	TiDBQPSMap           map[string]*sync.Map
}

func NewPromCache() *MetricCache {
	return &MetricCache{TiKVFlowBytesRateMap: make(map[string]*sync.Map), TiKVQPSMap: make(map[string]*sync.Map), TiDBQPSMap: make(map[string]*sync.Map)}
}

func (b *MetricCache) GetTiKVFlowBytes(tcCluster string, pod string) float64 {
	if _, ok := b.TiKVFlowBytesRateMap[tcCluster]; ok {
		if value, ok := b.TiKVFlowBytesRateMap[tcCluster].Load(pod); ok {
			return value.(float64)
		} else {
			return float64(0)
		}
	} else {
		return float64(0)
	}
}

func (b *MetricCache) GetTiDBQPSRate(tcCluster string, pod string) float64 {
	if _, ok := b.TiDBQPSMap[tcCluster]; ok {
		if value, ok := b.TiDBQPSMap[tcCluster].Load(pod); ok {
			return value.(float64)
		} else {
			return float64(0)
		}
	} else {
		return float64(0)
	}
}

func (b *MetricCache) SyncTiKVFlowByte(tcCluster string, prometheusAddr string) error {

	response := &struct {
		Status string `json:"status"`
		Data   struct {
			Result []struct {
				Metrics struct {
					Instance string `json:"instance"`
				} `json:"metric"`
				Values []interface{} `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}{}
	err := b.queryPrometheusQuery(prometheusAddr, fmt.Sprintf("sum(rate(tikv_engine_flow_bytes{tidb_cluster=~\"%s\",db=\"kv\"}[1m]))by(instance)", tcCluster), response)
	if response.Status != "success" {
		return err
	}
	if _, ok := b.TiKVFlowBytesRateMap[tcCluster]; !ok {
		b.TiKVFlowBytesRateMap[tcCluster] = &sync.Map{}
	}
	for _, data := range response.Data.Result {
		vStr := data.Values[1].(string)
		v, err := strconv.ParseFloat(vStr, 64)
		if err != nil {
			return err
		}
		klog.Errorf("sync tikvwrite metric %s:%f", data.Metrics.Instance, v)
		b.TiKVFlowBytesRateMap[tcCluster].Store(data.Metrics.Instance, v)
	}
	return nil
}

func (b *MetricCache) SyncTiDBQPS(tcCluster string, prometheusAddr string) error {
	response := &struct {
		Status string `json:"status"`
		Data   struct {
			Result []struct {
				Metrics struct {
					Instance string `json:"instance"`
				} `json:"metric"`
				Values []interface{} `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}{}
	err := b.queryPrometheusQuery(prometheusAddr, fmt.Sprintf("sum(rate(tidb_executor_statement_total{tidb_cluster=~\"%s\"}[1m]))by(instance)", tcCluster), response)
	if response.Status != "success" {
		return err
	}
	if _, ok := b.TiDBQPSMap[tcCluster]; !ok {
		b.TiDBQPSMap[tcCluster] = &sync.Map{}
	}
	for _, data := range response.Data.Result {
		vStr := data.Values[1].(string)
		v, err := strconv.ParseFloat(vStr, 64)
		if err != nil {
			return err
		}
		klog.Errorf("sync tidqps metric %s:%f", data.Metrics.Instance, v)
		b.TiDBQPSMap[tcCluster].Store(data.Metrics.Instance, v)
	}
	return nil
}

func (b *MetricCache) queryPrometheusQuery(prometheusAddr string, query string, result interface{}) error {
	prometheusSvc := fmt.Sprintf("http://%s/api/v1/query?query=%s", prometheusAddr, query)
	klog.Infof("cwtest:%s", prometheusSvc)
	resp, err := http.Get(prometheusSvc)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp == nil {
		return fmt.Errorf("resp is nil")
	}
	if resp.StatusCode != 200 {
		klog.Infof("cwtest resp:%v", resp)
		return fmt.Errorf("resp is failed")
	}
	err = json.Unmarshal(body, result)
	if err != nil {
		return err
	}

	return nil
}
