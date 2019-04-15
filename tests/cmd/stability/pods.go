package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/tests/pkg/client"
	"k8s.io/api/admission/v1beta1"
)

// only allow pods to pull images from specific registry.
func admitPods(ar v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	glog.Infof("admitting pods")
	httpClient := &http.Client{}
	podResource := metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	if ar.Request.Resource != podResource {
		err := fmt.Errorf("expect resource to be %s", podResource)
		glog.Errorf("%v", err)
		return toAdmissionResponse(err)
	}

	_, kubeCli := client.NewCliOrDie()

	name := ar.Request.Name
	nameSpace := ar.Request.Namespace

	reviewResponse := v1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true

	pod, err := kubeCli.CoreV1().Pods(nameSpace).Get(name, metav1.GetOptions{})
	if err != nil {
		reviewResponse.Allowed = false
		glog.Infof("%v", err)
		return &reviewResponse
	}

	glog.Infof("delete pod %s", pod.Labels["app.kubernetes.io/component"])

	if pod.Labels["app.kubernetes.io/component"] == "tidb" {
		podIP := pod.Status.PodIP
		url := fmt.Sprintf("http://%s:10080/info", podIP)
		req, err := http.NewRequest("POST", url, nil)
		if err != nil {
			glog.Errorf("fail to generator request %v", err)
			return &reviewResponse
		}

		res, err := httpClient.Do(req)
		if err != nil {
			glog.Errorf("fail to send request %v", err)
			return &reviewResponse
		}
		defer res.Body.Close()

		content, err := ioutil.ReadAll(res.Body)
		if err != nil {
			glog.Errorf("fail to read response %v", err)
			return &reviewResponse
		}

		if strings.Contains(string(content), "\"is_owner\":false") {
			glog.Infof("savely delete pod namespace %s name %s", nameSpace, name)
		}

		if strings.Contains(string(content), "\"is_owner\":true") {
			glog.Errorf("tidb is ddl owner, can't be deleted namespace %s name %s", nameSpace, name)
			os.Exit(3)
		}
	}

	return &reviewResponse
}
