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

package readable

import (
	"fmt"
	"io"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/tkctl/alias"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/kubernetes/pkg/printers"
	kubeprinters "k8s.io/kubernetes/pkg/printers"
	"k8s.io/kubernetes/pkg/util/node"
)

const (
	unset        = "<none>"
	podNameIndex = 0
)

// podBasicColumns holds common columns for all kinds of pod
type podBasicColumns struct {
	Name     string
	Ready    string
	Reason   string
	Restarts int64
	Memory   string
	Age      string
	PodIP    string
	NodeName string

	MemInfo string
	CPUInfo string
}

func AddHandlers(h printers.PrintHandler) {
	tidbClusterColumns := []metav1beta1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "PD", Type: "string", Description: "The PD nodes ready status"},
		{Name: "TiKV", Type: "string", Description: "The TiKV nodes ready status"},
		{Name: "TiDB", Type: "string", Description: "The TiDB nodes ready status"},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	h.TableHandler(tidbClusterColumns, printTidbClusterList)
	h.TableHandler(tidbClusterColumns, printTidbCluster)
	// TODO: separate different column definitions for PD/TiKV/TiDB Pod,
	// e.g. show store-id for tikv, show member-id for pd
	commonPodColumns := []metav1beta1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Ready", Type: "string", Description: "The aggregate readiness state of this pod for accepting traffic."},
		{Name: "Status", Type: "string", Description: "The aggregate status of the containers in this pod."},
		{Name: "Memory", Type: "string", Description: "The Pod total memory request and limit."},
		{Name: "CPU", Type: "string", Description: "The Pod total cpu request and limit."},
		{Name: "Restarts", Type: "integer", Description: "The number of times the containers in this pod have been restarted."},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
		{Name: "IP", Type: "string", Priority: 1, Description: apiv1.PodStatus{}.SwaggerDoc()["podIP"]},
		{Name: "Node", Type: "string", Priority: 1, Description: apiv1.PodSpec{}.SwaggerDoc()["nodeName"]},
	}
	h.TableHandler(commonPodColumns, printPod)
	h.TableHandler(commonPodColumns, printPodList)
	//tikv columns
	tikvPodColumns := commonPodColumns
	storeId := metav1beta1.TableColumnDefinition{
		Name:        "StoreID",
		Type:        "string",
		Description: "TiKV StoreID",
	}
	storeState := metav1beta1.TableColumnDefinition{
		Name:        "Store State",
		Type:        "string",
		Description: "TiKV Store State",
	}
	tikvPodColumns = append(tikvPodColumns, storeId, storeState)
	h.TableHandler(tikvPodColumns, printTikvList)
	// TODO: add available space for volume
	volumeColumns := []metav1beta1.TableColumnDefinition{
		{Name: "Volume", Type: "string", Format: "name", Description: "Volume name"},
		{Name: "Claim", Type: "string", Format: "name", Description: "Volume claim"},
		{Name: "Status", Type: "string", Description: "Volume status"},
		{Name: "Capacity", Type: "string", Description: "Volume capacity"},
		{Name: "StorageClass", Type: "string", Description: "Storage class of volume"},
		{Name: "Node", Type: "string", Priority: 1, Description: "Mounted node"},
		{Name: "Local", Type: "string", Priority: 1, Description: "Local path"},
	}
	h.TableHandler(volumeColumns, printVolume)
	h.TableHandler(volumeColumns, printVolumeList)
}

func printTidbClusterList(tcs *v1alpha1.TidbClusterList, options printers.GenerateOptions) ([]metav1beta1.TableRow, error) {
	rows := make([]metav1beta1.TableRow, 0, len(tcs.Items))
	for i := range tcs.Items {
		r, err := printTidbCluster(&tcs.Items[i], options)
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printTidbCluster(tc *v1alpha1.TidbCluster, options printers.GenerateOptions) ([]metav1beta1.TableRow, error) {
	row := metav1beta1.TableRow{
		Object: runtime.RawExtension{Object: tc},
	}
	pdReady := unset
	if tc.Status.PD.StatefulSet != nil {
		pdReady = fmt.Sprintf("%d/%d", tc.Status.PD.StatefulSet.ReadyReplicas, tc.Status.PD.StatefulSet.Replicas)
	}
	tikvReady := unset
	if tc.Status.TiKV.StatefulSet != nil {
		tikvReady = fmt.Sprintf("%d/%d", tc.Status.TiKV.StatefulSet.ReadyReplicas, tc.Status.TiKV.StatefulSet.Replicas)
	}
	tidbReady := unset
	if tc.Status.TiDB.StatefulSet != nil {
		tidbReady = fmt.Sprintf("%d/%d", tc.Status.TiDB.StatefulSet.ReadyReplicas, tc.Status.TiDB.StatefulSet.Replicas)
	}
	age := translateTimestampSince(tc.CreationTimestamp)

	row.Cells = append(row.Cells, tc.Name, pdReady, tikvReady, tidbReady, age)

	return []metav1beta1.TableRow{row}, nil
}

func printPodList(podList *v1.PodList, options printers.GenerateOptions) ([]metav1beta1.TableRow, error) {
	rows := make([]metav1beta1.TableRow, 0, len(podList.Items))
	for i := range podList.Items {
		r, err := printPod(&podList.Items[i], options)
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printPod(pod *v1.Pod, options printers.GenerateOptions) ([]metav1beta1.TableRow, error) {
	columns := basicPodColumns(pod)
	row := metav1beta1.TableRow{
		Object: runtime.RawExtension{Object: pod},
	}
	row.Cells = append(row.Cells,
		columns.Name,
		columns.Ready,
		columns.Reason,
		columns.MemInfo,
		columns.CPUInfo,
		columns.Restarts,
		columns.Age)

	if options.Wide {
		row.Cells = append(row.Cells, columns.PodIP, columns.NodeName)
	}
	return []metav1beta1.TableRow{row}, nil
}

// add more columns for tikv info
func printTikvList(tikvList *alias.TikvList, options printers.GenerateOptions) ([]metav1beta1.TableRow, error) {
	podList := tikvList.PodList
	metaTableRows, err := printPodList(podList, options)
	if err != nil {
		return nil, err
	}
	for id, row := range metaTableRows {
		podName := row.Cells[podNameIndex].(string)
		var storeId string
		for _, pod := range tikvList.PodList.Items {
			if podName == pod.Name {
				storeId = pod.Labels[label.StoreIDLabelKey]
				break
			}
		}
		viewStoreId := unset
		viewStoreState := unset
		// set storeId and tikv Store State if existed
		if len(storeId) > 0 {
			viewStoreId = storeId
			if tikvList.TikvStatus != nil && tikvList.TikvStatus.Stores != nil {
				if _, ok := tikvList.TikvStatus.Stores[storeId]; ok {
					viewStoreState = tikvList.TikvStatus.Stores[storeId].State
				}
			}
		}
		row.Cells = append(row.Cells, viewStoreId, viewStoreState)
		metaTableRows[id] = row
	}
	return metaTableRows, nil
}

func printVolumeList(volumeList *v1.PersistentVolumeList, options printers.GenerateOptions) ([]metav1beta1.TableRow, error) {
	rows := make([]metav1beta1.TableRow, 0, len(volumeList.Items))
	for i := range volumeList.Items {
		r, err := printVolume(&volumeList.Items[i], options)
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}

func printVolume(volume *v1.PersistentVolume, options printers.GenerateOptions) ([]metav1beta1.TableRow, error) {
	row := metav1beta1.TableRow{
		Object: runtime.RawExtension{Object: volume},
	}

	claim := unset
	if volume.Spec.ClaimRef != nil {
		claim = fmt.Sprintf("%s/%s", volume.Spec.ClaimRef.Namespace, volume.Spec.ClaimRef.Name)
	}
	local := unset
	if volume.Spec.Local != nil {
		local = volume.Spec.Local.Path
	}
	host := unset
	if volume.Spec.NodeAffinity != nil && volume.Spec.NodeAffinity.Required != nil {
	OUT:
		for _, selector := range volume.Spec.NodeAffinity.Required.NodeSelectorTerms {
			for _, expr := range selector.MatchExpressions {
				if expr.Key == "kubernetes.io/hostname" {
					// TODO: handle two or more nodes case
					if len(expr.Values) > 0 {
						host = expr.Values[0]
					}
					break OUT
				}
			}
		}
	}

	capacity := unset
	if volume.Spec.Capacity != nil {
		if val, ok := volume.Spec.Capacity["storage"]; ok {
			capacity = val.String()
		}
	}
	row.Cells = append(row.Cells, volume.Name, claim, volume.Status.Phase, capacity, volume.Spec.StorageClassName)

	if options.Wide {
		row.Cells = append(row.Cells, host, local)
	}

	return []metav1beta1.TableRow{row}, nil
}

// basicPodColumns calculates common columns for PD/TiKV/TiDB pods
func basicPodColumns(pod *v1.Pod) *podBasicColumns {
	restarts := 0
	totalContainers := len(pod.Spec.Containers)
	readyContainers := 0

	reason := string(pod.Status.Phase)
	if pod.Status.Reason != "" {
		reason = pod.Status.Reason
	}

	initializing := false
	for i := range pod.Status.InitContainerStatuses {
		container := pod.Status.InitContainerStatuses[i]
		restarts += int(container.RestartCount)
		switch {
		case container.State.Terminated != nil && container.State.Terminated.ExitCode == 0:
			continue
		case container.State.Terminated != nil:
			// initialization is failed
			if len(container.State.Terminated.Reason) == 0 {
				if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Init:Signal:%d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("Init:ExitCode:%d", container.State.Terminated.ExitCode)
				}
			} else {
				reason = "Init:" + container.State.Terminated.Reason
			}
			initializing = true
		case container.State.Waiting != nil && len(container.State.Waiting.Reason) > 0 && container.State.Waiting.Reason != "PodInitializing":
			reason = "Init:" + container.State.Waiting.Reason
			initializing = true
		default:
			reason = fmt.Sprintf("Init:%d/%d", i, len(pod.Spec.InitContainers))
			initializing = true
		}
		break
	}
	if !initializing {
		restarts = 0
		hasRunning := false
		for i := len(pod.Status.ContainerStatuses) - 1; i >= 0; i-- {
			container := pod.Status.ContainerStatuses[i]

			restarts += int(container.RestartCount)
			if container.State.Waiting != nil && container.State.Waiting.Reason != "" {
				reason = container.State.Waiting.Reason
			} else if container.State.Terminated != nil && container.State.Terminated.Reason != "" {
				reason = container.State.Terminated.Reason
			} else if container.State.Terminated != nil && container.State.Terminated.Reason == "" {
				if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Signal:%d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("ExitCode:%d", container.State.Terminated.ExitCode)
				}
			} else if container.Ready && container.State.Running != nil {
				hasRunning = true
				readyContainers++
			}
		}

		// change pod status back to "Running" if there is at least one container still reporting as "Running" status
		if reason == "Completed" && hasRunning {
			reason = "Running"
		}
	}

	if pod.DeletionTimestamp != nil && pod.Status.Reason == node.NodeUnreachablePodReason {
		reason = "Unknown"
	} else if pod.DeletionTimestamp != nil {
		reason = "Terminating"
	}

	nodeName := pod.Spec.NodeName
	podIP := pod.Status.PodIP
	if nodeName == "" {
		nodeName = unset
	}
	if podIP == "" {
		podIP = unset
	}

	cpuRequest := resource.Quantity{}
	cpuLimit := resource.Quantity{}
	memoryRequest := resource.Quantity{}
	memoryLimit := resource.Quantity{}
	for _, container := range pod.Spec.Containers {
		if container.Resources.Requests.Cpu() != nil {
			cpuRequest.Add(*container.Resources.Requests.Cpu())
		}
		if container.Resources.Requests.Memory() != nil {
			memoryRequest.Add(*container.Resources.Requests.Memory())
		}
		if container.Resources.Limits.Cpu() != nil {
			cpuLimit.Add(*container.Resources.Limits.Cpu())
		}
		if container.Resources.Limits.Memory() != nil {
			memoryLimit.Add(*container.Resources.Limits.Memory())
		}
	}
	cpuRequestStr := cpuRequest.String()
	if cpuRequest.IsZero() {
		cpuRequestStr = unset
	}
	memRequestStr := memoryRequest.String()
	if memoryRequest.IsZero() {
		memRequestStr = unset
	}
	cpuLimitStr := cpuLimit.String()
	if cpuLimit.IsZero() {
		cpuLimitStr = unset
	}
	memLimitStr := memoryLimit.String()
	if memoryLimit.IsZero() {
		memLimitStr = unset
	}
	memInfo := fmt.Sprintf("%s/%s", memRequestStr, memLimitStr)
	cpuInfo := fmt.Sprintf("%s/%s", cpuRequestStr, cpuLimitStr)

	return &podBasicColumns{
		Name:     pod.Name,
		Ready:    fmt.Sprintf("%d/%d", readyContainers, totalContainers),
		Reason:   reason,
		Restarts: int64(restarts),
		Age:      translateTimestampSince(pod.CreationTimestamp),
		NodeName: nodeName,
		PodIP:    podIP,
		MemInfo:  memInfo,
		CPUInfo:  cpuInfo,
	}
}

func translateTimestampSince(timestamp metav1.Time) string {
	if timestamp.IsZero() {
		return "<unknown>"
	}

	return duration.HumanDuration(time.Since(timestamp.Time))
}

// localPrinter prints object with local handlers.
type localPrinter struct {
	printer        kubeprinters.ResourcePrinter
	tableGenerator *kubeprinters.HumanReadableGenerator
	options        kubeprinters.GenerateOptions
}

func (l *localPrinter) PrintObj(obj runtime.Object, w io.Writer) error {
	table, err := l.tableGenerator.GenerateTable(obj, l.options)
	if table == nil {
		return fmt.Errorf("unknown type: %v", obj)
	}
	if err != nil {
		return err
	}
	return l.printer.PrintObj(table, w)
}

// NewLocalPrinter creates a new local printer.
func NewLocalPrinter(printer kubeprinters.ResourcePrinter, tableGenerator *kubeprinters.HumanReadableGenerator, options kubeprinters.GenerateOptions) kubeprinters.ResourcePrinter {
	return &localPrinter{
		printer:        printer,
		tableGenerator: tableGenerator,
		options:        options,
	}
}
