// Copyright 2018 PingCAP, Inc.
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
// limitations under the License.package spec

package e2e

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql" // init mysql driver
	. "github.com/onsi/ginkgo"         // revive:disable:dot-imports
	. "github.com/onsi/gomega"         // revive:disable:dot-imports
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	username = "admin"
	password = "admin"
)

type Result struct {
	Metric struct {
		Job string `json:"job"`
	} `json:"metric"`
	Values []interface{} `json:"values"`
}

type Response struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string   `json:"resultType"`
		Result     []Result `json:"result"`
	}
}

func testCreate(ns, clusterName string) {
	By(fmt.Sprintf("When create the TiDB cluster: %s/%s", ns, clusterName))
	instanceName := getInstanceName(ns, clusterName)

	cmdStr := fmt.Sprintf("helm install /charts/tidb-cluster -f /tidb-cluster-values.yaml"+
		" -n %s --namespace=%s --set clusterName=%s,tidb.passwordSecretName=%s",
		instanceName, ns, clusterName, ns+"-"+clusterName)
	_, err := execCmd(cmdStr)
	Expect(err).NotTo(HaveOccurred())

	err = createSecret(ns, clusterName)
	Expect(err).NotTo(HaveOccurred())

	By("Then all members should running")
	err = wait.Poll(5*time.Second, 5*time.Minute, func() (bool, error) {
		return allMembersRunning(ns, clusterName)
	})
	Expect(err).NotTo(HaveOccurred())

	By("And password is set correctly")
	err = wait.Poll(5*time.Second, 5*time.Minute, func() (bool, error) {
		return passwordIsSet(ns, clusterName)
	})
	Expect(err).NotTo(HaveOccurred())

	By("And scheduling policy is correct")
	nodeMap, err := getNodeMap(ns, clusterName, label.PDLabelVal)
	Expect(err).NotTo(HaveOccurred())
	for _, podNamesArr := range nodeMap {
		Expect(len(podNamesArr)).To(Equal(1))
	}
	nodeMap, err = getNodeMap(ns, clusterName, label.TiKVLabelVal)
	Expect(err).NotTo(HaveOccurred())
	for _, podNamesArr := range nodeMap {
		Expect(len(podNamesArr)).To(Equal(1))
	}

	By("When create a table and add some data to this table")
	err = wait.Poll(5*time.Second, 5*time.Minute, func() (bool, error) {
		return addDataToCluster(ns, clusterName)
	})
	Expect(err).NotTo(HaveOccurred())

	By("Then the data is correct")
	err = wait.Poll(5*time.Second, 5*time.Minute, func() (bool, error) {
		return dataIsCorrect(ns, clusterName)
	})
	Expect(err).NotTo(HaveOccurred())
}

func allMembersRunning(ns, clusterName string) (bool, error) {
	tc, err := cli.PingcapV1alpha1().TidbClusters(ns).Get(clusterName, metav1.GetOptions{})
	if err != nil {
		return false, nil
	}

	running, err := pdMemberRunning(tc)
	if err != nil || !running {
		return false, nil
	}

	running, err = tikvMemberRunning(tc)
	if err != nil || !running {
		return false, nil
	}

	running, err = tidbMemberRunning(tc)
	if err != nil || !running {
		return false, nil
	}

	synced, err := reclaimPolicySynced(tc)
	if err != nil || !synced {
		return false, nil
	}

	synced, err = metaSynced(tc)
	if err != nil || !synced {
		return false, nil
	}

	running, err = monitorMemberRunning(tc)
	if err != nil || !running {
		return false, nil
	}

	return true, nil
}

func addDataToCluster(ns, clusterName string) (bool, error) {
	db, err := sql.Open("mysql", getDSN(ns, clusterName))
	if err != nil {
		logf("can't open connection to mysql: %v", err)
		return false, nil
	}
	defer db.Close()

	_, err = db.Exec(fmt.Sprintf("CREATE TABLE %s (clusterName VARCHAR(64))", testTableName))
	if err != nil {
		logf("can't create table to mysql: %v", err)
		return false, nil
	}

	_, err = db.Exec(fmt.Sprintf("INSERT INTO %s VALUES (?)", testTableName), testTableVal)
	if err != nil {
		logf("can't insert data to mysql: %v", err)
		return false, nil
	}

	return true, nil
}

func dataIsCorrect(ns, clusterName string) (bool, error) {
	db, err := sql.Open("mysql", getDSN(ns, clusterName))
	if err != nil {
		return false, nil
	}

	rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s", testTableName))
	if err != nil {
		logf(err.Error())
		return false, nil
	}

	for rows.Next() {
		var v string
		err := rows.Scan(&v)
		if err != nil {
			logf(err.Error())
		}

		if v == testTableVal {
			return true, nil
		}

		return true, fmt.Errorf("val should equal: %s", testTableVal)
	}

	return false, nil
}

func pdMemberRunning(tc *v1alpha1.TidbCluster) (bool, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	pdSetName := controller.PDMemberName(tcName)
	pdSet, err := kubeCli.AppsV1beta1().StatefulSets(ns).Get(pdSetName, metav1.GetOptions{})
	if err != nil {
		logf(err.Error())
		return false, nil
	}

	logf("pdSet.Status: %+v", pdSet.Status)

	if tc.Status.PD.StatefulSet == nil {
		logf("tc.Status.PD.StatefulSet is nil")
		return false, nil
	}

	if *pdSet.Spec.Replicas != tc.Spec.PD.Replicas {
		logf("pdSet.Spec.Replicas(%d) != tc.Spec.PD.Replicas(%d)",
			*pdSet.Spec.Replicas, tc.Spec.PD.Replicas)
		return false, nil
	}

	if pdSet.Status.ReadyReplicas != tc.Spec.PD.Replicas {
		logf("pdSet.Status.ReadyReplicas(%d) != %d",
			pdSet.Status.ReadyReplicas, tc.Spec.PD.Replicas)
		return false, nil
	}

	if len(tc.Status.PD.Members) != int(tc.Spec.PD.Replicas) {
		logf("tc.Status.PD.Members count(%d) != %d",
			len(tc.Status.PD.Members), tc.Spec.PD.Replicas)
		return false, nil
	}

	if pdSet.Status.ReadyReplicas != pdSet.Status.Replicas {
		logf("pdSet.Status.ReadyReplicas(%d) != pdSet.Status.Replicas(%d)",
			pdSet.Status.ReadyReplicas, pdSet.Status.Replicas)
		return false, nil
	}

	for _, member := range tc.Status.PD.Members {
		if !member.Health {
			logf("pd member(%s) is not health", member.ID)
			return false, nil
		}
	}

	if tc.Status.ClusterID == "" {
		logf("tc.Status.ClusterID is nil")
		return false, nil
	}

	_, err = kubeCli.CoreV1().Services(ns).Get(controller.PDMemberName(tcName), metav1.GetOptions{})
	if err != nil {
		logf(err.Error())
		return false, nil
	}
	_, err = kubeCli.CoreV1().Services(ns).Get(controller.PDPeerMemberName(tcName), metav1.GetOptions{})
	if err != nil {
		logf(err.Error())
		return false, nil
	}

	return true, nil
}

func tikvMemberRunning(tc *v1alpha1.TidbCluster) (bool, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	tikvSetName := controller.TiKVMemberName(tcName)

	tikvSet, err := kubeCli.AppsV1beta1().StatefulSets(ns).Get(tikvSetName, metav1.GetOptions{})
	if err != nil {
		logf(err.Error())
		return false, nil
	}

	logf("tikvSet.Status: %+v", tikvSet.Status)

	if tc.Status.TiKV.StatefulSet == nil {
		logf("tc.Status.TiKV.StatefulSet is nil")
		return false, nil
	}

	if *tikvSet.Spec.Replicas != tc.Spec.TiKV.Replicas {
		logf("tikvSet.Spec.Replicas(%d) != tc.Spec.TiKV.Replicas(%d)",
			*tikvSet.Spec.Replicas, tc.Spec.TiKV.Replicas)
		return false, nil
	}

	if tikvSet.Status.ReadyReplicas != tc.Spec.TiKV.Replicas {
		logf("tikvSet.Status.ReadyReplicas(%d) != %d",
			tikvSet.Status.ReadyReplicas, tc.Spec.TiKV.Replicas)
		return false, nil
	}

	if len(tc.Status.TiKV.Stores) != int(tc.Spec.TiKV.Replicas) {
		logf("tc.Status.TiKV.Stores.count(%d) != %d",
			len(tc.Status.TiKV.Stores), tc.Spec.TiKV.Replicas)
		return false, nil
	}

	if tikvSet.Status.ReadyReplicas != tikvSet.Status.Replicas {
		logf("tikvSet.Status.ReadyReplicas(%d) != tikvSet.Status.Replicas(%d)",
			tikvSet.Status.ReadyReplicas, tikvSet.Status.Replicas)
		return false, nil
	}

	for _, store := range tc.Status.TiKV.Stores {
		if store.State != v1alpha1.TiKVStateUp {
			logf("store(%s) state != %s", store.ID, v1alpha1.TiKVStateUp)
			return false, nil
		}
	}

	_, err = kubeCli.CoreV1().Services(ns).Get(controller.TiKVPeerMemberName(tcName), metav1.GetOptions{})
	if err != nil {
		logf(err.Error())
		return false, nil
	}

	return true, nil
}

func tidbMemberRunning(tc *v1alpha1.TidbCluster) (bool, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	tidbSetName := controller.TiDBMemberName(tcName)
	tidbSet, err := kubeCli.AppsV1beta1().StatefulSets(ns).Get(tidbSetName, metav1.GetOptions{})
	if err != nil {
		logf(err.Error())
		return false, nil
	}

	logf("tidbSet.Status: %+v", tidbSet.Status)

	if tc.Status.TiDB.StatefulSet == nil {
		logf("tc.Status.TiDB.StatefulSet is nil")
		return false, nil
	}

	if *tidbSet.Spec.Replicas != tc.Spec.TiDB.Replicas {
		logf("tidbSet.Spec.Replicas(%d) != tc.Spec.TiDB.Replicas(%d)",
			*tidbSet.Spec.Replicas, tc.Spec.TiDB.Replicas)
		return false, nil
	}

	if tidbSet.Status.ReadyReplicas != tc.Spec.TiDB.Replicas {
		logf("tidbSet.Status.ReadyReplicas(%d) != %d",
			tidbSet.Status.ReadyReplicas, tc.Spec.TiDB.Replicas)
		return false, nil
	}

	if tidbSet.Status.ReadyReplicas != tidbSet.Status.Replicas {
		logf("tidbSet.Status.ReadyReplicas(%d) != tidbSet.Status.Replicas(%d)",
			tidbSet.Status.ReadyReplicas, tidbSet.Status.Replicas)
		return false, nil
	}

	_, err = kubeCli.CoreV1().Services(ns).Get(controller.TiDBMemberName(tcName), metav1.GetOptions{})
	if err != nil {
		logf(err.Error())
		return false, nil
	}

	return true, nil
}

func monitorMemberRunning(tc *v1alpha1.TidbCluster) (bool, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	deployName := fmt.Sprintf("%s-monitor", tcName)
	deploy, err := kubeCli.AppsV1beta1().Deployments(ns).Get(deployName, metav1.GetOptions{})
	if err != nil {
		logf(err.Error())
		return false, nil
	}
	if deploy.Status.ReadyReplicas < 1 {
		logf("monitor ready replicas %d < 1", deploy.Status.ReadyReplicas)
		return false, nil
	}
	if err := checkGrafanaData(tc); err != nil {
		logf("can't get grafana data: %v", err)
		return false, nil
	}
	return true, nil
}

func checkGrafanaData(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	svcName := fmt.Sprintf("%s-grafana", tcName)
	end := time.Now()
	start := end.Add(-time.Minute)
	values := url.Values{}
	values.Set("query", `sum(tikv_pd_heartbeat_tick_total{type="leader"}) by (job)`)
	values.Set("start", fmt.Sprintf("%d", start.Unix()))
	values.Set("end", fmt.Sprintf("%d", end.Unix()))
	values.Set("step", "30")
	u := fmt.Sprintf("http://%s.%s.svc.cluster.local:3000/api/datasources/proxy/1/api/v1/query_range?%s", svcName, ns, values.Encode())
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(username, password)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	data := &Response{}
	if err := json.Unmarshal(buf, data); err != nil {
		return err
	}
	if data.Status != "success" || len(data.Data.Result) < 1 {
		return fmt.Errorf("invalid response: status: %s, result: %v", data.Status, data.Data.Result)
	}
	return nil
}

func reclaimPolicySynced(tc *v1alpha1.TidbCluster) (bool, error) {
	ns := tc.GetNamespace()
	instanceName := tc.GetLabels()[label.InstanceLabelKey]
	labelSelector := label.New().Instance(instanceName)
	pvcList, err := kubeCli.CoreV1().PersistentVolumeClaims(ns).List(
		metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(
				labelSelector.Labels(),
			).String(),
		},
	)
	if err != nil {
		logf(err.Error())
		return false, nil
	}

	for _, pvc := range pvcList.Items {
		pv, err := kubeCli.CoreV1().PersistentVolumes().Get(pvc.Spec.VolumeName, metav1.GetOptions{})
		if err != nil {
			logf(err.Error())
			return false, nil
		}

		logf("pv: %s's persistentVolumeReclaimPolicy is %s", pv.GetName(), pv.Spec.PersistentVolumeReclaimPolicy)
		if pv.Spec.PersistentVolumeReclaimPolicy != tc.Spec.PVReclaimPolicy {
			return false, nil
		}
	}

	return true, nil
}

func passwordIsSet(ns, clusterName string) (bool, error) {
	jobName := clusterName + "-tidb-initializer"
	job, err := kubeCli.BatchV1().Jobs(ns).Get(jobName, metav1.GetOptions{})
	if err != nil {
		return false, nil
	}
	if job.Status.Succeeded < 1 {
		logf("password setter job not finished")
		return false, nil
	}

	db, err := sql.Open("mysql", getDSN(ns, clusterName))
	if err != nil {
		logf("can't open connection to mysql: %v", err)
		return false, nil
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		logf("can't connect to tidb: %s/%s-tidb with password %s", ns, clusterName, password)
		return false, nil
	}
	return true, nil
}

func getDSN(ns, clusterName string) string {
	return fmt.Sprintf("root:%s@(%s-tidb.%s:4000)/test?charset=utf8", password, clusterName, ns)
}

func metaSynced(tc *v1alpha1.TidbCluster) (bool, error) {
	ns := tc.GetNamespace()
	instanceName := tc.GetLabels()[label.InstanceLabelKey]

	pdControl := controller.NewDefaultPDControl()
	pdCli := pdControl.GetPDClient(tc)
	cluster, err := pdCli.GetCluster()
	if err != nil {
		logf(err.Error())
		return false, nil
	}
	clusterID := strconv.FormatUint(cluster.Id, 10)

	labelSelector := label.New().Instance(instanceName)
	podList, err := kubeCli.CoreV1().Pods(ns).List(
		metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(
				labelSelector.Labels(),
			).String(),
		},
	)
	if err != nil {
		logf(err.Error())
		return false, nil
	}

outerLoop:
	for _, pod := range podList.Items {
		podName := pod.GetName()
		Expect(pod.Labels[label.ClusterIDLabelKey]).To(Equal(clusterID))

		component := pod.Labels[label.ComponentLabelKey]
		switch component {
		case label.PDLabelVal:
			var memberID string
			members, err := pdCli.GetMembers()
			if err != nil {
				logf(err.Error())
				return false, nil
			}
			for _, member := range members.Members {
				if member.Name == podName {
					memberID = strconv.FormatUint(member.GetMemberId(), 10)
					break
				}
			}
			Expect(memberID).NotTo(BeEmpty())
			Expect(pod.Labels[label.MemberIDLabelKey]).To(Equal(memberID))
		case label.TiKVLabelVal:
			var storeID string
			stores, err := pdCli.GetStores()
			if err != nil {
				logf(err.Error())
				return false, nil
			}
			for _, store := range stores.Stores {
				addr := store.Store.GetAddress()
				if strings.Split(addr, ".")[0] == podName {
					storeID = strconv.FormatUint(store.Store.GetId(), 10)
					break
				}
			}
			Expect(storeID).NotTo(BeEmpty())
			Expect(pod.Labels[label.StoreIDLabelKey]).To(Equal(storeID))
		case label.TiDBLabelVal:
			continue outerLoop
		}

		var pvcName string
		for _, vol := range pod.Spec.Volumes {
			if vol.PersistentVolumeClaim != nil {
				pvcName = vol.PersistentVolumeClaim.ClaimName
				break
			}
		}
		if pvcName == "" {
			logf("pod: %s/%s's pvcName is empty", ns, podName)
			return false, nil
		}

		pvc, err := kubeCli.CoreV1().PersistentVolumeClaims(ns).Get(pvcName, metav1.GetOptions{})
		if err != nil {
			logf(err.Error())
			return false, nil
		}
		Expect(pvc.Labels[label.ClusterIDLabelKey]).To(Equal(clusterID))
		Expect(pvc.Labels[label.MemberIDLabelKey]).To(Equal(pod.Labels[label.MemberIDLabelKey]))
		Expect(pvc.Labels[label.StoreIDLabelKey]).To(Equal(pod.Labels[label.StoreIDLabelKey]))
		Expect(pvc.Annotations[label.AnnPodNameKey]).To(Equal(podName))

		pvName := pvc.Spec.VolumeName
		pv, err := kubeCli.CoreV1().PersistentVolumes().Get(pvName, metav1.GetOptions{})
		if err != nil {
			logf(err.Error())
			return false, nil
		}
		Expect(pv.Labels[label.NamespaceLabelKey]).To(Equal(ns))
		Expect(pv.Labels[label.ComponentLabelKey]).To(Equal(pod.Labels[label.ComponentLabelKey]))
		Expect(pv.Labels[label.NameLabelKey]).To(Equal(pod.Labels[label.NameLabelKey]))
		Expect(pv.Labels[label.ManagedByLabelKey]).To(Equal(pod.Labels[label.ManagedByLabelKey]))
		Expect(pv.Labels[label.InstanceLabelKey]).To(Equal(pod.Labels[label.InstanceLabelKey]))
		Expect(pv.Labels[label.ClusterIDLabelKey]).To(Equal(clusterID))
		Expect(pv.Labels[label.MemberIDLabelKey]).To(Equal(pod.Labels[label.MemberIDLabelKey]))
		Expect(pv.Labels[label.StoreIDLabelKey]).To(Equal(pod.Labels[label.StoreIDLabelKey]))
		Expect(pv.Annotations[label.AnnPodNameKey]).To(Equal(podName))
	}

	return true, nil
}

func createSecret(ns, clusterName string) error {
	secretName := ns + "-" + clusterName
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: ns,
		},
		Data: map[string][]byte{
			"root": []byte(password),
		},
		Type: corev1.SecretTypeOpaque,
	}
	_, err := kubeCli.CoreV1().Secrets(ns).Create(&secret)
	return err
}
