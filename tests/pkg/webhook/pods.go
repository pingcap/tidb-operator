package webhook

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/golang/glog"
	"github.com/pingcap/kvproto/pkg/pdpb"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/tests/pkg/client"
	"k8s.io/api/admission/v1beta1"
)

type dbInfo struct {
	IsOwner bool `json:"is_owner"`
}

var (
	kvLeaders map[string]int
	kvShoot   map[string]bool
)

func HttpHandler(url string, method string, httpClient *http.Client) (content []byte, err error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		glog.Errorf("fail to generator request %v", err)
		return nil, err
	}

	res, err := httpClient.Do(req)
	if err != nil {
		glog.Errorf("fail to send request %v", err)
		return nil, err
	}
	defer res.Body.Close()

	content, err = ioutil.ReadAll(res.Body)
	if err != nil {
		glog.Errorf("fail to read response %v", err)
		return nil, err
	}

	return content, nil

}

func getAllKVLeaders(tc *v1alpha1.TidbCluster, httpClient *http.Client) (ret map[string]int, err error) {
	ret = make(map[string]int)
	podIP := tc.Status.PD.Leader.ClientURL
	for _, store := range tc.Status.TiKV.Stores {
		url := fmt.Sprintf("%s/pd/api/v1/store/%s", podIP, store.ID)

		content, err := HttpHandler(url, "GET", httpClient)
		if err != nil {
			glog.Errorf("fail to read response %v", err)
			return nil, err
		}

		storeInfo := &controller.StoreInfo{}
		err = json.Unmarshal(content, storeInfo)
		if err != nil {
			glog.Errorf("unmarshal failed, %v", err)
			return nil, err
		}

		ret[store.PodName] = storeInfo.Status.LeaderCount
	}

	return ret, nil
}

// only allow pods to be delete when it is not ddlowner of tidb, not leader of pd and not
// master of tikv.
func admitPods(ar v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	glog.Infof("admitting pods")

	if kvShoot == nil {
		kvShoot = make(map[string]bool)
	}

	glog.Infof("kvshoot [%#v]",kvShoot)

	httpClient := &http.Client{}
	podResource := metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	if ar.Request.Resource != podResource {
		err := fmt.Errorf("expect resource to be %s", podResource)
		glog.Errorf("%v", err)
		return toAdmissionResponse(err)
	}

	versionCli, kubeCli := client.NewCliOrDie()

	name := ar.Request.Name
	nameSpace := ar.Request.Namespace
	shootName := fmt.Sprintf("%s:%s", nameSpace, name)

	reviewResponse := v1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true

	pod, err := kubeCli.CoreV1().Pods(nameSpace).Get(name, metav1.GetOptions{})
	if err != nil {
		reviewResponse.Allowed = false
		glog.Infof("%v", err)
		return &reviewResponse
	}

	glog.Infof("delete pod %s", pod.Labels[label.ComponentLabelKey])

	tc, err := versionCli.PingcapV1alpha1().TidbClusters(nameSpace).Get(pod.Labels["app.kubernetes.io/instance"], metav1.GetOptions{})
	if err != nil {
		reviewResponse.Allowed = false
		glog.Infof("%v", err)
		return &reviewResponse
	}

	if pod.Labels[label.ComponentLabelKey] == "tidb" {
		podIP := pod.Status.PodIP
		url := fmt.Sprintf("http://%s:10080/info", podIP)

		content, err := HttpHandler(url, "POST", httpClient)
		if err != nil {
			glog.Errorf("fail to read response %v", err)
			return &reviewResponse
		}

		info := dbInfo{}
		err = json.Unmarshal(content, &info)
		if err != nil {
			glog.Errorf("unmarshal failed,namespace %s name %s error:%v", nameSpace, name, err)
			return &reviewResponse
		}

		if info.IsOwner && tc.Status.TiDB.StatefulSet.Replicas > 1 {
			time.Sleep(10 * time.Second)
			glog.Errorf("tidb is ddl owner, can't be deleted namespace %s name %s", nameSpace, name)
			os.Exit(3)
		} else {
			glog.Infof("savely delete pod namespace %s name %s content %s", nameSpace, name, string(content))
		}

		kvShoot[shootName] = true

	} else if pod.Labels[label.ComponentLabelKey] == "pd" {
		podIP := tc.Status.PD.Leader.ClientURL
		url := fmt.Sprintf("%s/pd/api/v1/leader", podIP)

		content, err := HttpHandler(url, "GET", httpClient)
		if err != nil {
			glog.Errorf("fail to read response %v", err)
			return &reviewResponse
		}

		leader := &pdpb.Member{}
		err = json.Unmarshal(content, leader)
		if err != nil {
			glog.Errorf("unmarshal failed,namespace %s name %s error:%v", nameSpace, name, err)
			return &reviewResponse
		}

		if leader.Name == name && tc.Status.TiDB.StatefulSet.Replicas > 1 {
			time.Sleep(10 * time.Second)
			glog.Errorf("pd is leader, can't be deleted namespace %s name %s", nameSpace, name)
			os.Exit(3)
		} else {
			glog.Infof("savely delete pod namespace %s name %s leader name %s", nameSpace, name, leader.Name)
		}

		kvShoot[shootName] = true

	} else if pod.Labels[label.ComponentLabelKey] == "tikv" {

		if _, ok := kvShoot[shootName]; ok {
			return &reviewResponse
		}

		var storeID string
		podIP := tc.Status.PD.Leader.ClientURL
		for _, store := range tc.Status.TiKV.Stores {
			if store.PodName == name {
				storeID = store.ID
			}
		}

		url := fmt.Sprintf("%s/pd/api/v1/store/%s", podIP, storeID)

		content, err := HttpHandler(url, "GET", httpClient)
		if err != nil {
			glog.Errorf("fail to read response %v", err)
			return &reviewResponse
		}

		storeInfo := &controller.StoreInfo{}
		err = json.Unmarshal(content, storeInfo)
		if err != nil {
			glog.Errorf("unmarshal failed,namespace %s name %s error:%v", nameSpace, name, err)
			return &reviewResponse
		}

		beforeCount := kvLeaders[name]
		afterCount := storeInfo.Status.LeaderCount
		glog.Infof("before evict the leader is %d after evict the leader is %d", beforeCount, afterCount)

		if beforeCount != 0 && afterCount >= beforeCount && tc.Status.TiKV.StatefulSet.Replicas > 1 {
			time.Sleep(10 * time.Second)
			glog.Errorf("kv leader is not zero, can't be deleted namespace %s name %s leaderCount %d", nameSpace, name, storeInfo.Status.LeaderCount)
			os.Exit(3)
		} else {
			glog.Infof("savely delete pod namespace %s name %s", nameSpace, name)
		}

		kvShoot[shootName] = true

	}

	if kvLeaders, err = getAllKVLeaders(tc, httpClient); err != nil {
		glog.Errorf("fail to get kv infos #v", err)
	}

	glog.Infof("%#v", kvLeaders)

	return &reviewResponse
}
