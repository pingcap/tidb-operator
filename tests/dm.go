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

package tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	// To register MySQL driver
	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/sync/errgroup"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework/log"
	"k8s.io/utils/pointer"

	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/label"
	httputil "github.com/pingcap/tidb-operator/pkg/util/http"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
)

const (
	// DMLabelName is the label value for `app.kubernetes.io/name`.
	DMLabelName = "dm-e2e"
	// DMMySQLNamespace is the namespace used to install upstream MySQL instances for DM E2E tests.
	DMMySQLNamespace = "dm-mysql"
	// DMMySQLSvcStsName is the upstream MySQL svc/sts name for DM E2E tests.
	DMMySQLSvcStsName = "dm-mysql"
	// DMMySQLImage is the upstream MySQL container image for DM E2E tests.
	DMMySQLImage = "mysql:5.7"
	// DMMySQLReplicas is the upstream MySQL instance number for DM E2E	tests.
	// We use replicas as different MySQL instances.
	DMMySQLReplicas int32 = 2
	// DMMySQLPort is the upstream MySQL instance's service port.
	DMMySQLPort int32 = 3306
	// DMMySQLStorage is the request storage used by one MySQL instance.
	DMMySQLStorage = "2Gi"

	// DMTiDBNamespace is the namespace used to install the downstream TiDB cluster for DM E2E tests.
	DMTiDBNamespace = "dm-tidb"
	// DMTiDBName is the name of the TiDB cluster for DM E2E tests.
	DMTiDBName = "dm-tidb"

	// the service port for client requests.
	dmMasterSvcPort = uint16(8261)

	// PK steps for generated data between MySQL replicas.
	dmPKStepForReplica = 1000000
)

var (
	dmMySQLLabels = map[string]string{
		label.NameLabelKey:      DMLabelName,
		label.ComponentLabelKey: "mysql",
	}

	dmMySQLSvc = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DMMySQLSvcStsName,
			Namespace: DMMySQLNamespace,
			Labels:    dmMySQLLabels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       DMMySQLSvcStsName,
					Port:       DMMySQLPort,
					TargetPort: intstr.FromInt(int(DMMySQLPort)),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: dmMySQLLabels,
			Type:     corev1.ServiceTypeClusterIP,
		},
	}

	dmMySQLSts = &appv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DMMySQLSvcStsName,
			Namespace: DMMySQLNamespace,
			Labels:    dmMySQLLabels,
		},
		Spec: appv1.StatefulSetSpec{
			Replicas:            pointer.Int32Ptr(DMMySQLReplicas),
			Selector:            &metav1.LabelSelector{MatchLabels: dmMySQLLabels},
			ServiceName:         DMMySQLSvcStsName,
			PodManagementPolicy: appv1.ParallelPodManagement,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: dmMySQLLabels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            DMMySQLSvcStsName,
							Image:           DMMySQLImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      DMMySQLSvcStsName,
									MountPath: "/var/lib/mysql",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "MYSQL_ALLOW_EMPTY_PASSWORD",
									Value: "true",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          DMMySQLSvcStsName,
									ContainerPort: DMMySQLPort,
								},
							},
							Args: []string{
								"--server-id=1",
								"--log-bin=/var/lib/mysql/mysql-bin",
								"--enforce-gtid-consistency=ON",
								"--gtid-mode=ON",
								"--binlog-format=ROW",
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: DMMySQLSvcStsName,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse(DMMySQLStorage),
							},
						},
					},
				},
			},
		},
	}

	dmSources = []string{`
source-id: "replica-01"
enable-gtid: true
enable-relay: true

from:
  host: "dm-mysql-0.dm-mysql"
  user: "root"
  password: ""
  port: 3306
`, `
source-id: "replica-02"
enable-gtid: true
enable-relay: true

from:
host: "dm-mysql-1.dm-mysql"
user: "root"
password: ""
port: 3306
`}

	dmSingleTask = `
name: "task_single"
task-mode: all
timezone: "UTC"

target-database:
  host: "dm-tidb-tidb-0.dm-tidb"
  port: 4000
  user: "root"
  password: ""

mysql-instances:
  -
    source-id: "replica-01"
    block-allow-list: "instance"

block-allow-list:
  instance:
    do-dbs: ["t1", "t2"] # ref "dmTableNames" below.
`

	// table names in every database.
	dmTableNames = []string{"t1", "t2"}

	// the table schema for every table.
	// NOTE: we use the same schema for all tables now because we are focusing tests on TiDB Operator.
	dmTableSchema = "(c1 INT PRIMARY KEY, c2 TEXT)"

	// the format string for `INSERT INTO` statement.
	dmInsertStmtFormat = "INSERT INTO `%s`.`%s` (c1, c2) VALUES (%d, %s)"

	// generated rows for every table in full stage.
	dmFullRowsInTable = 10
)

// GetDMMySQLAddress gets the upstream MySQL address for a replica.
func GetDMMySQLAddress(ordinal int32) string {
	return fmt.Sprintf("%s-%d.%s.%s", DMMySQLSvcStsName, ordinal, DMMySQLSvcStsName, DMMySQLNamespace)
}

// DeployDMMySQL deploy upstream MySQL instances for DM E2E tests.
func DeployDMMySQL(kubeCli kubernetes.Interface) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: DMMySQLNamespace,
		},
	}
	_, err := kubeCli.CoreV1().Namespaces().Create(ns)
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create namespace[%s]: %v", DMMySQLNamespace, err)
	}

	_, err = kubeCli.CoreV1().Services(DMMySQLNamespace).Create(dmMySQLSvc)
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create service[%s]: %v", dmMySQLSvc.Name, err)
	}

	_, err = kubeCli.AppsV1().StatefulSets(DMMySQLNamespace).Create(dmMySQLSts)
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create statefulset[%s]: %v", dmMySQLSts.Name, err)
	}
	return nil
}

// CheckDMMySQLReady checks whether all upstream MySQL instances are ready.
func CheckDMMySQLReady(fw portforward.PortForward) error {
	var eg errgroup.Group
	for i := int32(0); i < DMMySQLReplicas; i++ {
		ordinal := i
		eg.Go(func() error {
			return wait.Poll(10*time.Second, 10*time.Minute, func() (done bool, err error) {
				localHost, localPort, cancel, err := portforward.ForwardOnePort(
					fw, DMMySQLNamespace, fmt.Sprintf("pod/%s-%d", DMMySQLSvcStsName, ordinal), uint16(DMMySQLPort))
				if err != nil {
					log.Logf("failed to forward MySQL[%d] pod: %v", ordinal, err)
					return false, nil
				}
				defer cancel()

				_, err = openDB(localHost, localPort)
				if err != nil {
					log.Logf("failed to open database connection: %v", err)
					return false, nil
				}
				return true, nil
			})
		})
	}

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("failed to check MySQL ready: %v", err)
	}
	return nil
}

// CleanDMMySQL cleans upstream MySQL instances for DM E2E tests.
func CleanDMMySQL(kubeCli kubernetes.Interface) error {
	err := kubeCli.AppsV1().StatefulSets(DMMySQLNamespace).Delete(dmMySQLSts.Name, nil)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete statefulset[%s]: %v", dmMySQLSts.Name, err)
	}

	err = kubeCli.CoreV1().Services(DMMySQLNamespace).Delete(dmMySQLSvc.Name, nil)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete service[%s]: %v", dmMySQLSvc.Name, err)
	}

	err = kubeCli.CoreV1().Namespaces().Delete(DMMySQLNamespace, nil)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete namespace[%s]: %v", DMMySQLNamespace, err)
	}
	return nil
}

// CleanDMTiDB cleans the downstream TiDB cluster for DM E2E tests.
func CleanDMTiDB(cli *versioned.Clientset, kubeCli kubernetes.Interface) error {
	if err := cli.PingcapV1alpha1().TidbClusters(DMTiDBNamespace).Delete(DMTiDBName, nil); err != nil && !errors.IsNotFound(err) {
		return err
	}

	if err := kubeCli.CoreV1().Namespaces().Delete(DMTiDBNamespace, nil); err != nil && !errors.IsNotFound(err) {
		return err
	}
	return nil
}

// CreateDMSources creates upstream MySQL sources into DM cluster.
func CreateDMSources(fw portforward.PortForward, ns, masterSvcName string) error {
	apiPath := "/apis/v1alpha1/sources"

	type Req struct {
		Op     int      `json:"op"`
		Config []string `json:"config"`
	}
	var req = Req{
		Op:     1,
		Config: dmSources,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal source create request, %v, %v", req, err)
	}

	return wait.Poll(5*time.Second, 3*time.Minute, func() (bool, error) {
		localHost, localPort, cancel, err := portforward.ForwardOnePort(
			fw, ns, fmt.Sprintf("svc/%s", masterSvcName), dmMasterSvcPort)
		if err != nil {
			log.Logf("failed to forward dm-master svc: %v", err)
			return false, nil
		}
		defer cancel()

		body, err := httputil.DoBodyOK(
			&http.Client{Transport: &http.Transport{}},
			fmt.Sprintf("http://%s:%d%s", localHost, localPort, apiPath),
			"PUT",
			bytes.NewReader(data))
		if err != nil && !bytes.Contains(body, []byte("already exists")) {
			log.Logf("failed to create DM sources: %v", err)
			return false, nil
		}
		return true, nil
	})
}

// GenDMFullData generates full stage data for upstream MySQL.
func GenDMFullData(fw portforward.PortForward, ns string) error {
	var eg errgroup.Group
	for i := int32(0); i < DMMySQLReplicas; i++ {
		ordinal := i
		eg.Go(func() error {
			localHost, localPort, cancel, err := portforward.ForwardOnePort(
				fw, DMMySQLNamespace, fmt.Sprintf("pod/%s-%d", DMMySQLSvcStsName, ordinal), uint16(DMMySQLPort))
			if err != nil {
				return fmt.Errorf("failed to forward MySQL[%d] pod: %v", ordinal, err)
			}
			defer cancel()

			db, err := openDB(localHost, localPort)
			if err != nil {
				return err
			}

			// use ns as the database name.
			// NOTE: we don't handle `already exists` or `duplicate entry` error now because we think ns is unqiue and the database instances are re-setup for each running.
			_, err = db.Exec(fmt.Sprintf("CREATE DATABASE `%s`", ns))
			if err != nil {
				return fmt.Errorf("failed to create database `%s`: %v", ns, err)
			}
			for j, tbl := range dmTableNames {
				_, err = db.Exec(fmt.Sprintf("CREATE TABLE `%s`.`%s` %s", ns, tbl, dmTableSchema))
				if err != nil {
					return fmt.Errorf("failed to create table `%s`.`%s`: %v", ns, tbl, err)
				}
				for k := 1; k <= dmFullRowsInTable; k++ {
					val := j*dmPKStepForReplica + k
					_, err = db.Exec(fmt.Sprintf(dmInsertStmtFormat, ns, tbl, val, strconv.Itoa(val))) // must match `dmTableSchema`
					if err != nil {
						return fmt.Errorf("failed to insert data into table `%s`.`%s`: %v", ns, tbl, err)
					}
				}
			}
			return nil
		})
	}

	return eg.Wait()
}

// StartDMSingleSourceTask starts a DM migration task with a single source.
func StartDMSingleSourceTask(fw portforward.PortForward, ns, masterSvcName string) error {
	apiPath := "/apis/v1alpha1/tasks"

	type Req struct {
		Task string `json:"task"`
	}
	var req = Req{
		Task: dmSingleTask,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal task start request, %v, %v", req, err)
	}

	return wait.Poll(5*time.Second, time.Minute, func() (bool, error) {
		localHost, localPort, cancel, err := portforward.ForwardOnePort(
			fw, ns, fmt.Sprintf("svc/%s", masterSvcName), dmMasterSvcPort)
		if err != nil {
			log.Logf("failed to forward dm-master svc: %v", err)
			return false, nil
		}
		defer cancel()

		body, err := httputil.DoBodyOK(
			&http.Client{Transport: &http.Transport{}},
			fmt.Sprintf("http://%s:%d%s", localHost, localPort, apiPath),
			"POST",
			bytes.NewReader(data))
		if err != nil && !bytes.Contains(body, []byte("already exists")) {
			log.Logf("failed to start DM single source task: %v", err)
			return false, nil
		}
		return true, nil
	})
}

func openDB(host string, port uint16) (*sql.DB, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	dsn := fmt.Sprintf("root:@(%s)/?charset=utf8", addr)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %v", addr, err)
	}
	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %v", addr, err)
	}
	return db, nil
}
