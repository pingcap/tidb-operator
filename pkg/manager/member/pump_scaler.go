// Copyright 2021 PingCAP, Inc.
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

package member

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/util"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
)

type pumpScaler struct {
	generalScaler
}

// NewPumpScaler returns a pump Scaler
func NewPumpScaler(deps *controller.Dependencies) *pumpScaler {
	return &pumpScaler{generalScaler: generalScaler{deps: deps}}
}

func (s *pumpScaler) Scale(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	scaling, _, _, _ := scaleOne(oldSet, newSet)
	if scaling > 0 {
		return s.ScaleOut(meta, oldSet, newSet)
	} else if scaling < 0 {
		return s.ScaleIn(meta, oldSet, newSet)
	}

	return nil
}

func (s *pumpScaler) ScaleOut(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	_, ordinal, replicas, deleteSlots := scaleOne(oldSet, newSet)
	resetReplicas(newSet, oldSet)
	obj, ok := meta.(runtime.Object)
	if !ok {
		return fmt.Errorf("cluster[%s/%s] can't convert to runtime.Object", meta.GetNamespace(), meta.GetName())
	}

	klog.Infof("scaling out pump statefulset %s/%s, ordinal: %d (replicas: %d, delete slots: %v)", oldSet.Namespace, oldSet.Name, ordinal, replicas, deleteSlots.List())
	var pvcName string
	switch meta.(type) {
	case *v1alpha1.TidbCluster:
		pvcName = fmt.Sprintf("data-%s-pump-%d", meta.GetName(), ordinal)
	default:
		return fmt.Errorf("pump.ScaleOut, failed to convert cluster %s/%s", meta.GetNamespace(), meta.GetName())
	}
	_, err := s.deps.PVCLister.PersistentVolumeClaims(meta.GetNamespace()).Get(pvcName)
	if err == nil {
		_, err = s.deleteDeferDeletingPVC(obj, v1alpha1.PumpMemberType, ordinal)
		if err != nil {
			return err
		}
		return controller.RequeueErrorf("pump.ScaleOut, cluster %s/%s ready to scale out, wait for next round", meta.GetNamespace(), meta.GetName())
	} else if !errors.IsNotFound(err) {
		return fmt.Errorf("pump.ScaleOut, cluster %s/%s failed to fetch pvc informaiton, err:%v", meta.GetNamespace(), meta.GetName(), err)
	}
	setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
	return nil
}

// parse the advertise address from pod start command.
// must respect to pump start script in template.go
func pumpAdvertiseAddr(pod *v1.Pod) string {
	for _, c := range pod.Spec.Containers {
		if c.Name != "pump" {
			continue
		}

		for _, cmd := range c.Command {
			if !strings.Contains(cmd, "-advertise-addr") {
				continue
			}

			// example:
			// v1:
			//   -advertise-addr=`echo ${HOSTNAME}`.basic-pump:8250 \
			// v2:
			//   -advertise-addr=${PUMP_POD_NAME}.start-script-test-pump.start-script-test-ns.svc:8250 \
			for _, str := range strings.Split(cmd, "\n") {
				if !strings.Contains(str, "-advertise-addr") {
					continue
				}

				str = strings.Replace(str, "`echo ${HOSTNAME}`", pod.Name, -1)
				str = strings.Replace(str, "${PUMP_POD_NAME}", pod.Name, -1)
				str = strings.Replace(str, "-advertise-addr=", "", -1)
				str = strings.TrimRight(str, "\\")
				str = strings.TrimSpace(str)
				return str
			}
		}
	}

	return ""
}

func (s *pumpScaler) ScaleIn(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	ns := meta.GetNamespace()
	tcName := meta.GetName()

	// we can only remove one member at a time when scaling in
	_, ordinal, replicas, deleteSlots := scaleOne(oldSet, newSet)
	resetReplicas(newSet, oldSet)

	// ref: https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#deployment-and-scaling-guarantees
	// Before a scaling operation is applied to a Pod, all of its predecessors must be Running and Ready.
	//
	// after offline the pump, the pump will be not ready, so we only offline the next one after the successors pod are deleted.
	if *oldSet.Spec.Replicas != oldSet.Status.Replicas {
		return controller.RequeueErrorf("Wait for statefulset to be synced before scale in one more replica, desired: %d, running: %d", *oldSet.Spec.Replicas, oldSet.Status.Replicas)
	}

	klog.Infof("scaling in pump statefulset %s/%s, ordinal: %d (replicas: %d, delete slots: %v)", oldSet.Namespace, oldSet.Name, ordinal, replicas, deleteSlots.List())
	// We need remove member from cluster before reducing statefulset replicas
	var podName string

	switch meta.(type) {
	case *v1alpha1.TidbCluster:
		podName = ordinalPodName(v1alpha1.PumpMemberType, tcName, ordinal)
	default:
		return fmt.Errorf("pumpScaler.ScaleIn: failed to convert cluster %s/%s", meta.GetNamespace(), meta.GetName())
	}
	pod, err := s.deps.PodLister.Pods(ns).Get(podName)
	if err != nil {
		return fmt.Errorf("pumpScaler.ScaleIn: failed to get pods %s for cluster %s/%s, error: %s", podName, ns, tcName, err)
	}

	tc, _ := meta.(*v1alpha1.TidbCluster)

	client, err := buildBinlogClient(tc, s.deps.PDControl)
	if err != nil {
		return err
	}
	defer client.Close()

	// Since the advertise address may no contains the namespace
	// and operator do not run in the same namespace with tidb-cluster,
	// we can not use this advertise address to access pump.
	// so will add the namespace part to the address if need.
	client.HookAddr = func(addr string) string {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return addr
		}

		suffix := "." + ns

		if strings.Contains(addr, suffix) {
			return addr
		}

		host += suffix
		return host + ":" + port
	}

	addr := pumpAdvertiseAddr(pod)

	for _, node := range tc.Status.Pump.Members {
		if node.Host != addr {
			continue
		}

		if node.State == "online" {
			err := client.OfflinePump(context.TODO(), addr)
			if err != nil {
				return err
			}
			klog.Infof("pumpScaler.ScaleIn: send offline request to pump %s/%s successfully", ns, podName)
			return controller.RequeueErrorf("Pump %s/%s is still in cluster, state: %s", ns, podName, node.State)
		} else if node.State == "offline" {
			klog.Infof("Pump %s/%s becomes offline", ns, podName)
			pvcs, err := util.ResolvePVCFromPod(pod, s.deps.PVCLister)
			if err != nil {
				return fmt.Errorf("pumpScaler.ScaleIn: failed to get pvcs for pod %s/%s in tc %s/%s, error: %s", ns, pod.Name, ns, tcName, err)
			}
			for _, pvc := range pvcs {
				if err := addDeferDeletingAnnoToPVC(tc, pvc, s.deps.PVCControl); err != nil {
					return err
				}
			}
			setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
			return nil
		} else {
			return controller.RequeueErrorf("Pump %s/%s is still in cluster, state: %s", ns, podName, node.State)
		}
	}

	// When this pump pod not found in TidbCluster status, there are two possible situations:
	// 1. Pump has already joined cluster but status not synced yet.
	//    In this situation return error to wait for another round for safety.
	// 2. Pump pod is not ready, such as in pending state.
	//    In this situation we should delete this Pump pod immediately to avoid blocking the subsequent operations.
	if !podutil.IsPodReady(pod) {
		safeTimeDeadline := pod.CreationTimestamp.Add(5 * s.deps.CLIConfig.ResyncDuration)
		if time.Now().Before(safeTimeDeadline) {
			// Wait for 5 resync periods to ensure that the following situation does not occur:
			//
			// The pump pod starts for a while, but has not synced its status, and then the pod becomes not ready.
			// Here we wait for 5 resync periods to ensure that the status of this pump pod has been synced.
			// After this period of time, if there is still no information about this pump in TidbCluster status,
			// then we can be sure that this pump has never been added to the tidb cluster.
			// So we can scale in this pump pod safely.
			resetReplicas(newSet, oldSet)
			return fmt.Errorf("Pump %s/%s is not ready, wait for 5 resync periods to sync its status", ns, podName)
		}

		pvcs, err := util.ResolvePVCFromPod(pod, s.deps.PVCLister)
		if err != nil {
			return fmt.Errorf("pumpScaler.ScaleIn: failed to get pvcs for pod %s/%s in tc %s/%s, error: %s", ns, pod.Name, ns, tcName, err)
		}
		for _, pvc := range pvcs {
			if err := addDeferDeletingAnnoToPVC(tc, pvc, s.deps.PVCControl); err != nil {
				return err
			}
		}

		setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
		return nil
	}
	return fmt.Errorf("Pump %s/%s not found in cluster", ns, podName)
}

type fakePumpScaler struct{}

// NewFakePumpScaler returns a fake pump Scaler
func NewFakePumpScaler() Scaler {
	return &fakePumpScaler{}
}

func (s *fakePumpScaler) Scale(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	if *newSet.Spec.Replicas > *oldSet.Spec.Replicas {
		return s.ScaleOut(meta, oldSet, newSet)
	} else if *newSet.Spec.Replicas < *oldSet.Spec.Replicas {
		return s.ScaleIn(meta, oldSet, newSet)
	}
	return nil
}

func (s *fakePumpScaler) ScaleOut(_ metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	setReplicasAndDeleteSlots(newSet, *oldSet.Spec.Replicas+1, nil)
	return nil
}

func (s *fakePumpScaler) ScaleIn(_ metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	setReplicasAndDeleteSlots(newSet, *oldSet.Spec.Replicas-1, nil)
	return nil
}
