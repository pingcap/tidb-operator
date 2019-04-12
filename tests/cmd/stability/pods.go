package main

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/golang/glog"
	"k8s.io/api/admission/v1beta1"
	"github.com/pingcap/tidb-operator/tests/pkg/client"
)

// only allow pods to pull images from specific registry.
func admitPods(ar v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	glog.Infof("admitting pods")
	podResource := metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	if ar.Request.Resource != podResource {
		err := fmt.Errorf("expect resource to be %s", podResource)
		glog.Errorf("%v",err)
		return toAdmissionResponse(err)
	}

	_, kubeCli := client.NewCliOrDie()

	name := ar.Request.Name
	nameSpace := ar.Request.Namespace

	reviewResponse := v1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true

	pod , err := kubeCli.CoreV1().Pods(nameSpace).Get(name,metav1.GetOptions{})
	if err != nil {
		reviewResponse.Allowed = false
		glog.Infof("%v",err)
		return &reviewResponse
	}

	glog.Infof("delete pod %#v",pod)

	if !reviewResponse.Allowed {
	}
	return &reviewResponse
}
