// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tidb

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/k8s"
)

var dummyCancel = func() {}

// PortForwardAndGetTiDBDSN create a port forward for TiDB and return its DSN.
func PortForwardAndGetTiDBDSN(fw k8s.PortForwarder, ns, svcName,
	user, password, database, params string) (string, context.CancelFunc, error) {
	localHost, localPort, cancel, err := k8s.ForwardOnePort(fw, ns, fmt.Sprintf("svc/%s", svcName), uint16(v1alpha1.DefaultTiDBPortClient))
	if err != nil {
		return "", dummyCancel, err
	}
	return fmt.Sprintf("%s:%s@(%s:%d)/%s?%s", user, password, localHost, localPort, database, params), cancel, nil
}

// IsTiDBConnectable checks whether the tidb cluster is connectable.
func IsTiDBConnectable(ctx context.Context, cli client.Client, fw k8s.PortForwarder,
	ns, tcName, dbgName, user, password, tlsSecretName string,
) error {
	var svcList corev1.ServiceList
	if err := cli.List(ctx, &svcList, client.InNamespace(ns), client.MatchingLabels{
		v1alpha1.LabelKeyCluster: tcName, v1alpha1.LabelKeyGroup: dbgName,
	}); err != nil {
		return fmt.Errorf("failed to list tidb service %s for tidb cluster %s/%s: %w", dbgName, ns, tcName, err)
	}
	var svc *corev1.Service
	for i := range svcList.Items {
		item := &svcList.Items[i]
		if item.Spec.ClusterIP != corev1.ClusterIPNone {
			svc = item
			break
		}
	}
	if svc == nil {
		return fmt.Errorf("tidb service %s for tidb cluster %s/%s not found", dbgName, ns, tcName)
	}

	// enable "cleartext client side plugin" for `tidb_auth_token`.
	// ref: https://github.com/go-sql-driver/mysql?tab=readme-ov-file#allowcleartextpasswords
	parms := []string{"charset=utf8", "allowCleartextPasswords=true"}
	if tlsSecretName != "" {
		var secret corev1.Secret
		if err := cli.Get(ctx, client.ObjectKey{Namespace: ns, Name: tlsSecretName}, &secret); err != nil {
			return fmt.Errorf("failed to get TLS secret %s/%s: %w", ns, tlsSecretName, err)
		}

		rootCAs := x509.NewCertPool()
		rootCAs.AppendCertsFromPEM(secret.Data[corev1.ServiceAccountRootCAKey])
		clientCert, certExists := secret.Data[corev1.TLSCertKey]
		clientKey, keyExists := secret.Data[corev1.TLSPrivateKeyKey]
		if !certExists || !keyExists {
			return fmt.Errorf("cert or key does not exist in secret %s/%s", ns, tlsSecretName)
		}

		tlsCert, err := tls.X509KeyPair(clientCert, clientKey)
		if err != nil {
			return fmt.Errorf("unable to load certificates from secret %s/%s: %w", ns, tlsSecretName, err)
		}
		tlsKey := fmt.Sprintf("%s-%s", ns, tlsSecretName)
		err = mysql.RegisterTLSConfig(tlsKey, &tls.Config{
			RootCAs:            rootCAs,
			Certificates:       []tls.Certificate{tlsCert},
			InsecureSkipVerify: true, //nolint:gosec // skip verify because we may use self-signed certificate
		})
		if err != nil {
			return fmt.Errorf("unable to register TLS config %s: %w", tlsKey, err)
		}

		parms = append(parms, fmt.Sprintf("tls=%s", tlsKey))
	}

	var db *sql.DB
	dsn, cancel, err := PortForwardAndGetTiDBDSN(fw, ns, svc.Name, user, password, "test", strings.Join(parms, "&"))
	if err != nil {
		return fmt.Errorf("failed to port forward for tidb service %s/%s: %w", ns, svc.Name, err)
	}
	defer cancel()
	if db, err = sql.Open("mysql", dsn); err != nil {
		return fmt.Errorf("failed to open MySQL connection to tidb service %s/%s: %w", ns, svc.Name, err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping TiDB server with tidb service %s/%s: %w", ns, svc.Name, err)
	}
	return nil
}

// IsTiDBInserted checks whether the tidb cluster has insert some data.
func IsTiDBInserted(fw k8s.PortForwarder, ns, tc, user, password, dbName, tableName string) wait.ConditionFunc {
	return func() (bool, error) {
		parms := []string{"charset=utf8"}
		var db *sql.DB
		dsn, cancel, err := PortForwardAndGetTiDBDSN(fw, ns, tc, user, password, dbName, strings.Join(parms, "&"))
		if err != nil {
			return false, err
		}

		defer cancel()
		if db, err = sql.Open("mysql", dsn); err != nil {
			return false, err
		}

		defer db.Close()
		//nolint:gocritic // use := will shadow err
		if err = db.Ping(); err != nil {
			return false, err
		}

		getCntFn := func(db *sql.DB, tableName string) (int, error) {
			var cnt int
			row := db.QueryRow(fmt.Sprintf("SELECT count(*) FROM %s", tableName))

			err = row.Scan(&cnt)
			if err != nil {
				return cnt, fmt.Errorf("failed to scan count from %s, %w", tableName, err)
			}
			return cnt, nil
		}

		cnt, err := getCntFn(db, tableName)
		if err != nil {
			return false, err
		}
		if cnt == 0 {
			return false, nil
		}

		return true, nil
	}
}

func IsClusterReady(cli client.Client, name, ns string) (*v1alpha1.Cluster, bool) {
	var tcGet v1alpha1.Cluster
	if err := cli.Get(context.TODO(), client.ObjectKey{Namespace: ns, Name: name}, &tcGet); err == nil {
		availCond := meta.FindStatusCondition(tcGet.Status.Conditions, v1alpha1.ClusterCondAvailable)
		if availCond != nil && availCond.Status == metav1.ConditionTrue {
			return &tcGet, true
		}
	}

	return nil, false
}

func AreAllInstancesReady(cli client.Client, pdg *v1alpha1.PDGroup, kvg []*v1alpha1.TiKVGroup,
	dbg []*v1alpha1.TiDBGroup, flashg []*v1alpha1.TiFlashGroup,
) error {
	if err := AreAllPDHealthy(cli, pdg); err != nil {
		return err
	}
	for _, kv := range kvg {
		if err := AreAllTiKVHealthy(cli, kv); err != nil {
			return err
		}
	}
	for _, db := range dbg {
		if err := AreAllTiDBHealthy(cli, db); err != nil {
			return err
		}
	}
	for _, flash := range flashg {
		if err := AreAllTiFlashHealthy(cli, flash); err != nil {
			return err
		}
	}
	return nil
}

func AreAllPDHealthy(cli client.Client, pdg *v1alpha1.PDGroup) error {
	var pdList v1alpha1.PDList
	if err := cli.List(context.TODO(), &pdList, client.InNamespace(pdg.Namespace), client.MatchingLabels{
		v1alpha1.LabelKeyGroup:     pdg.Name,
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
	}); err != nil {
		return fmt.Errorf("failed to list pd %s/%s: %w", pdg.Namespace, pdg.Name, err)
	}
	if len(pdList.Items) != int(*pdg.Spec.Replicas) {
		return fmt.Errorf("pd %s/%s replicas %d not equal to %d", pdg.Namespace, pdg.Name, len(pdList.Items), *pdg.Spec.Replicas)
	}
	for i := range pdList.Items {
		pd := &pdList.Items[i]
		if !meta.IsStatusConditionPresentAndEqual(pd.Status.Conditions, v1alpha1.PDCondInitialized, metav1.ConditionTrue) {
			return fmt.Errorf("pd %s/%s is not initialized", pd.Namespace, pd.Name)
		}
		if !meta.IsStatusConditionPresentAndEqual(pd.Status.Conditions, v1alpha1.PDCondHealth, metav1.ConditionTrue) {
			return fmt.Errorf("pd %s/%s is not healthy", pd.Namespace, pd.Name)
		}
	}

	var podList corev1.PodList
	if err := cli.List(context.TODO(), &podList, client.InNamespace(pdg.Namespace), client.MatchingLabels{
		v1alpha1.LabelKeyCluster:   pdg.Spec.Cluster.Name,
		v1alpha1.LabelKeyGroup:     pdg.Name,
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
	}); err != nil {
		return fmt.Errorf("failed to list pd pod %s/%s: %w", pdg.Namespace, pdg.Name, err)
	}
	if len(podList.Items) != int(*pdg.Spec.Replicas) {
		return fmt.Errorf("pd %s/%s pod replicas %d not equal to %d", pdg.Namespace, pdg.Name, len(podList.Items), *pdg.Spec.Replicas)
	}
	for i := range podList.Items {
		pod := &podList.Items[i]
		if pod.Status.Phase != corev1.PodRunning {
			return fmt.Errorf("pd %s/%s pod %s is not running, current phase: %s", pdg.Namespace, pdg.Name, pod.Name, pod.Status.Phase)
		}
	}

	return nil
}

func AreAllTiKVHealthy(cli client.Client, kvg *v1alpha1.TiKVGroup) error {
	var tikvList v1alpha1.TiKVList
	if err := cli.List(context.TODO(), &tikvList, client.InNamespace(kvg.Namespace), client.MatchingLabels{
		v1alpha1.LabelKeyGroup:     kvg.Name,
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiKV,
	}); err != nil {
		return fmt.Errorf("failed to list tikv %s/%s: %w", kvg.Namespace, kvg.Name, err)
	}
	if len(tikvList.Items) != int(*kvg.Spec.Replicas) {
		return fmt.Errorf("tikv %s/%s replicas %d not equal to %d", kvg.Namespace, kvg.Name, len(tikvList.Items), *kvg.Spec.Replicas)
	}
	for i := range tikvList.Items {
		tikv := &tikvList.Items[i]
		if tikv.Status.State != v1alpha1.StoreStateServing {
			return fmt.Errorf("tikv %s/%s is not Serving, current state: %s", tikv.Namespace, tikv.Name, tikv.Status.State)
		}
	}

	var podList corev1.PodList
	if err := cli.List(context.TODO(), &podList, client.InNamespace(kvg.Namespace), client.MatchingLabels{
		v1alpha1.LabelKeyCluster:   kvg.Spec.Cluster.Name,
		v1alpha1.LabelKeyGroup:     kvg.Name,
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiKV,
	}); err != nil {
		return fmt.Errorf("failed to list tikv pod %s/%s: %w", kvg.Namespace, kvg.Name, err)
	}
	if len(podList.Items) != int(*kvg.Spec.Replicas) {
		return fmt.Errorf("tikv %s/%s pod replicas %d not equal to %d", kvg.Namespace, kvg.Name, len(podList.Items), *kvg.Spec.Replicas)
	}
	for i := range podList.Items {
		pod := &podList.Items[i]
		if pod.Status.Phase != corev1.PodRunning {
			return fmt.Errorf("tikv %s/%s pod %s is not running, current phase: %s", kvg.Namespace, kvg.Name, pod.Name, pod.Status.Phase)
		}
	}

	return nil
}

func AreAllTiDBHealthy(cli client.Client, dbg *v1alpha1.TiDBGroup) error {
	var tidbList v1alpha1.TiDBList
	if err := cli.List(context.TODO(), &tidbList, client.InNamespace(dbg.Namespace), client.MatchingLabels{
		v1alpha1.LabelKeyGroup:     dbg.Name,
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiDB,
	}); err != nil {
		return fmt.Errorf("failed to list tidb %s/%s: %w", dbg.Namespace, dbg.Name, err)
	}
	if len(tidbList.Items) != int(*dbg.Spec.Replicas) {
		return fmt.Errorf("tidb %s/%s replicas %d not equal to %d", dbg.Namespace, dbg.Name, len(tidbList.Items), *dbg.Spec.Replicas)
	}
	for i := range tidbList.Items {
		tidb := &tidbList.Items[i]
		if !meta.IsStatusConditionPresentAndEqual(tidb.Status.Conditions, v1alpha1.TiDBCondHealth, metav1.ConditionTrue) {
			return fmt.Errorf("tidb %s/%s is not healthy", tidb.Namespace, tidb.Name)
		}
	}

	var podList corev1.PodList
	if err := cli.List(context.TODO(), &podList, client.InNamespace(dbg.Namespace), client.MatchingLabels{
		v1alpha1.LabelKeyCluster:   dbg.Spec.Cluster.Name,
		v1alpha1.LabelKeyGroup:     dbg.Name,
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiDB,
	}); err != nil {
		return fmt.Errorf("failed to list tidb pod %s/%s: %w", dbg.Namespace, dbg.Name, err)
	}
	if len(podList.Items) != int(*dbg.Spec.Replicas) {
		return fmt.Errorf("tidb %s/%s pod replicas %d not equal to %d", dbg.Namespace, dbg.Name, len(podList.Items), *dbg.Spec.Replicas)
	}
	for i := range podList.Items {
		pod := &podList.Items[i]
		if pod.Status.Phase != corev1.PodRunning {
			return fmt.Errorf("tidb %s/%s pod %s is not running, current phase: %s", dbg.Namespace, dbg.Name, pod.Name, pod.Status.Phase)
		}
	}

	return nil
}

func AreAllTiFlashHealthy(cli client.Client, flashg *v1alpha1.TiFlashGroup) error {
	var tiflashList v1alpha1.TiFlashList
	if err := cli.List(context.TODO(), &tiflashList, client.InNamespace(flashg.Namespace), client.MatchingLabels{
		v1alpha1.LabelKeyGroup:     flashg.Name,
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiFlash,
	}); err != nil {
		return fmt.Errorf("failed to list tiflash %s/%s: %w", flashg.Namespace, flashg.Name, err)
	}
	if len(tiflashList.Items) != int(*flashg.Spec.Replicas) {
		return fmt.Errorf("tiflash %s/%s replicas %d not equal to %d",
			flashg.Namespace, flashg.Name, len(tiflashList.Items), *flashg.Spec.Replicas)
	}
	for i := range tiflashList.Items {
		tiflash := &tiflashList.Items[i]
		if tiflash.Status.State != v1alpha1.StoreStateServing {
			return fmt.Errorf("tiflash %s/%s is not Serving, current state: %s", tiflash.Namespace, tiflash.Name, tiflash.Status.State)
		}
	}

	var podList corev1.PodList
	if err := cli.List(context.TODO(), &podList, client.InNamespace(flashg.Namespace), client.MatchingLabels{
		v1alpha1.LabelKeyCluster:   flashg.Spec.Cluster.Name,
		v1alpha1.LabelKeyGroup:     flashg.Name,
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiFlash,
	}); err != nil {
		return fmt.Errorf("failed to list tiflash pod %s/%s: %w", flashg.Namespace, flashg.Name, err)
	}
	if len(podList.Items) != int(*flashg.Spec.Replicas) {
		return fmt.Errorf("tiflash %s/%s pod replicas %d not equal to %d",
			flashg.Namespace, flashg.Name, len(podList.Items), *flashg.Spec.Replicas)
	}
	for i := range podList.Items {
		pod := &podList.Items[i]
		if pod.Status.Phase != corev1.PodRunning {
			return fmt.Errorf("tiflash %s/%s pod %s is not running, current phase: %s", flashg.Namespace, flashg.Name, pod.Name, pod.Status.Phase)
		}
	}

	return nil
}

// ExecuteSimpleTransaction performs a transaction to insert or update the given id in the specified table.
func ExecuteSimpleTransaction(db *sql.DB, id int, table string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin txn: %w", err)
	}

	// Prepare SQL statement to replace or insert a record
	//nolint:gosec // only replace table name in test
	str := fmt.Sprintf("replace into %s(id, v) values(?, ?);", table)
	if _, err = tx.Exec(str, id, id); err != nil {
		return fmt.Errorf("failed to exec statement: %w", err)
	}

	// Simulate a different operation by updating the value
	if _, err = tx.Exec(fmt.Sprintf("update %s set v = ? where id = ?;", table), id*2, id); err != nil {
		return fmt.Errorf("failed to exec update statement: %w", err)
	}

	// Simulate a long transaction by sleeping for 5 seconds for even ids
	if id%3 == 0 {
		//nolint:mnd // just for testing
		time.Sleep(10 * time.Second)
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit txn: %w", err)
	}
	return nil
}
