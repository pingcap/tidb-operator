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

package backup

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	e2eframework "github.com/pingcap/tidb-operator/tests/e2e/br/framework"
	brutil "github.com/pingcap/tidb-operator/tests/e2e/br/framework/br"
	"github.com/pingcap/tidb-operator/tests/e2e/br/utils/blockwriter"
	"github.com/pingcap/tidb-operator/tests/e2e/br/utils/feature"
	"github.com/pingcap/tidb-operator/tests/e2e/br/utils/portforward"
	utilimage "github.com/pingcap/tidb-operator/tests/e2e/util/image"
	utiltidbcluster "github.com/pingcap/tidb-operator/tests/e2e/util/tidbcluster"
	"github.com/pingcap/tidb-operator/tests/pkg/fixture"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/test/e2e/framework"
)

var (
	tidbReadyTimeout       = time.Minute * 5
	backupCompleteTimeout  = time.Minute * 3
	restoreCompleteTimeout = time.Minute * 3
)

const (
	featKeyVersion   = "Version"
	versionV4        = "V4"
	versionV5        = "V5"
	versionV42V5     = "V4->V5"
	featVersionV4    = featKeyVersion + ":" + versionV4
	featVersionV5    = featKeyVersion + ":" + versionV5
	featVersionV42V5 = featKeyVersion + ":" + versionV42V5

	featKeyType    = "Type"
	typeBR         = "BR"
	typeDumper     = "Dumper"
	featTypeBR     = featKeyType + ":" + typeBR
	featTypeDumper = featKeyType + ":" + typeDumper

	featTLS = "TLS"
)

var _ = ginkgo.Describe("Backup and Restore", func() {
	f := e2eframework.NewFramework("br")

	ginkgo.BeforeEach(func() {
		accessKey := "12345678"
		secretKey := "12345678"
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		err := f.Storage.Init(ctx, f.Namespace.Name, accessKey, secretKey)
		framework.ExpectNoError(err)
	})

	ginkgo.JustAfterEach(func() {
		err := f.Storage.Clean(context.Background(), f.Namespace.Name)
		framework.ExpectNoError(err)
	})

	feats := []feature.Features{
		feature.New(featVersionV4, featTypeBR).Build(),
		feature.New(featVersionV4, featTypeDumper).Build(),
		// feature.New(featVersionV4, featTypeBR, featTLS).Build(),
		// feature.New(featVersionV4, featTypeDumper, featTLS).Build(),

		feature.New(featVersionV5, featTypeBR).Build(),
		feature.New(featVersionV5, featTypeDumper).Build(),
		feature.New(featVersionV5, featTypeBR, featTLS).Build(),
		feature.New(featVersionV5, featTypeDumper, featTLS).Build(),

		feature.New(featVersionV42V5, featTypeBR).Build(),
		feature.New(featVersionV42V5, featTypeDumper).Build(),
	}

	for i := range feats {
		feat := feats[i]
		desc := feat.String() + "Backup and Restore"

		enableTLS := feat.Has(featTLS)
		typ := strings.ToLower(feat.Value(featKeyType))
		// NOTE: mysql and test will be filtered by default
		dbName := "e2etest"
		backupClusterName := fmt.Sprintf("backup-with-%s", typ)
		restoreClusterName := fmt.Sprintf("restore-with-%s", typ)
		backupName := fmt.Sprintf("%s-backup", backupClusterName)
		restoreName := fmt.Sprintf("%s-restore", restoreClusterName)

		var backupVersion, restoreVersion string
		switch feat.Value(featKeyVersion) {
		case versionV4:
			backupVersion = utilimage.TiDBV4
			restoreVersion = utilimage.TiDBV4
		case versionV5:
			backupVersion = utilimage.TiDBV5
			restoreVersion = utilimage.TiDBV5
		case versionV42V5:
			backupVersion = utilimage.TiDBV4
			restoreVersion = utilimage.TiDBV5
		}

		ginkgo.It(desc, func() {
			ns := f.Namespace.Name
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ginkgo.By("Create TiDB cluster for backup")
			err := createTidbCluster(f, backupClusterName, backupVersion, enableTLS)
			framework.ExpectNoError(err)

			ginkgo.By("Create TiDB cluster for restore")
			err = createTidbCluster(f, restoreClusterName, restoreVersion, enableTLS)
			framework.ExpectNoError(err)

			ginkgo.By("Wait for backup TiDB cluster ready")
			err = utiltidbcluster.WaitForTidbClusterConditionReady(f.ExtClient, ns, backupClusterName, tidbReadyTimeout, 0)
			framework.ExpectNoError(err)

			ginkgo.By("Wait for restore TiDB cluster ready")
			err = utiltidbcluster.WaitForTidbClusterConditionReady(f.ExtClient, ns, restoreClusterName, tidbReadyTimeout, 0)
			framework.ExpectNoError(err)

			ginkgo.By("Forward backup TiDB cluster service")
			backupHost, err := portforward.ForwardOnePort(ctx, f.PortForwarder, ns, getTiDBServiceResourceName(backupClusterName), 4000)
			framework.ExpectNoError(err)
			err = initDatabase(backupHost, dbName)
			framework.ExpectNoError(err)

			ginkgo.By("Write data into backup TiDB cluster")
			backupDSN := getDefaultDSN(backupHost, dbName)
			err = blockwriter.NewDefault().Write(context.Background(), backupDSN)
			framework.ExpectNoError(err)

			ginkgo.By("Create RBAC for backup and restore")
			err = createRBAC(f)
			framework.ExpectNoError(err)

			ginkgo.By("Create backup")
			err = createBackupAndWaitForComplete(f, backupName, backupClusterName, typ)
			framework.ExpectNoError(err)

			ginkgo.By("Create restore")
			err = createRestoreAndWaitForComplete(f, restoreName, restoreClusterName, typ, backupName)
			framework.ExpectNoError(err)

			ginkgo.By("Forward restore TiDB cluster service")
			restoreHost, err := portforward.ForwardOnePort(ctx, f.PortForwarder, ns, getTiDBServiceResourceName(restoreClusterName), 4000)
			framework.ExpectNoError(err)

			ginkgo.By("Validate restore result")
			restoreDSN := getDefaultDSN(restoreHost, dbName)
			err = checkDataIsSame(backupDSN, restoreDSN)
			framework.ExpectNoError(err)
		})
	}
})

func getTiDBServiceResourceName(tcName string) string {
	// TODO: use common util to get tidb service name
	return "svc/" + tcName + "-tidb"
}

func createTidbCluster(f *e2eframework.Framework, name string, version string, enableTLS bool) error {
	ns := f.Namespace.Name
	// TODO: change to use tidbclusterutil like brutil
	tc := fixture.GetTidbCluster(ns, name, version)
	tc.Spec.PD.Replicas = 1
	tc.Spec.TiKV.Replicas = 1
	tc.Spec.TiDB.Replicas = 1
	if enableTLS {
		tc.Spec.TiDB.TLSClient = &v1alpha1.TiDBTLSClient{Enabled: true}
		tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}

		if err := f.TLSManager.CreateTLSForTidbCluster(tc); err != nil {
			return err
		}
	}

	if _, err := f.ExtClient.PingcapV1alpha1().TidbClusters(ns).Create(tc); err != nil {
		return err
	}

	return nil
}

func createRBAC(f *e2eframework.Framework) error {
	ns := f.Namespace.Name
	sa := brutil.GetServiceAccount(ns)
	if _, err := f.ClientSet.CoreV1().ServiceAccounts(ns).Create(sa); err != nil {
		return err
	}

	role := brutil.GetRole(ns)
	if _, err := f.ClientSet.RbacV1().Roles(ns).Create(role); err != nil {
		return err
	}
	rb := brutil.GetRoleBinding(ns)
	if _, err := f.ClientSet.RbacV1().RoleBindings(ns).Create(rb); err != nil {
		return err
	}
	return nil
}

func createBackupAndWaitForComplete(f *e2eframework.Framework, name, tcName, typ string) error {
	ns := f.Namespace.Name
	// secret to visit tidb cluster
	s := brutil.GetSecret(ns, name, "")
	if _, err := f.ClientSet.CoreV1().Secrets(ns).Create(s); err != nil {
		return err
	}

	backupFolder := time.Now().Format(time.RFC3339)
	cfg := f.Storage.Config(ns, backupFolder)
	backup := brutil.GetBackup(ns, name, tcName, typ, cfg)

	if _, err := f.ExtClient.PingcapV1alpha1().Backups(ns).Create(backup); err != nil {
		return err
	}

	if err := brutil.WaitForBackupComplete(f.ExtClient, ns, name, backupCompleteTimeout); err != nil {
		return err
	}
	return nil
}

// nolint
// NOTE: it is not used now
func deleteBackup(f *e2eframework.Framework, name string) error {
	ns := f.Namespace.Name

	if err := f.ExtClient.PingcapV1alpha1().Backups(ns).Delete(name, nil); err != nil {
		return err
	}

	if err := brutil.WaitForBackupDeleted(f.ExtClient, ns, name, time.Second*30); err != nil {
		return err
	}
	return nil
}

// nolint
// NOTE: it is not used
func cleanBackup(f *e2eframework.Framework) error {
	ns := f.Namespace.Name
	bl, err := f.ExtClient.PingcapV1alpha1().Backups(ns).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for i := range bl.Items {
		if err := deleteBackup(f, bl.Items[i].Name); err != nil {
			return err
		}
	}
	return nil
}

func createRestoreAndWaitForComplete(f *e2eframework.Framework, name, tcName, typ string, backupName string) error {
	ns := f.Namespace.Name

	// secret to visit tidb cluster
	s := brutil.GetSecret(ns, name, "")
	if _, err := f.ClientSet.CoreV1().Secrets(ns).Create(s); err != nil {
		return err
	}

	backup, err := f.ExtClient.PingcapV1alpha1().Backups(ns).Get(backupName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	cfg := backup.Spec.S3
	// TODO: backup path is hard to understand
	// BackupPath is only needed for dumper
	if typ == "dumper" {
		cfg.Path = backup.Status.BackupPath
	}

	restore := brutil.GetRestore(ns, name, tcName, typ, cfg)
	if _, err := f.ExtClient.PingcapV1alpha1().Restores(ns).Create(restore); err != nil {
		return err
	}

	if err := brutil.WaitForRestoreComplete(f.ExtClient, ns, name, restoreCompleteTimeout); err != nil {
		return err
	}

	return nil
}

func getDefaultDSN(host, dbName string) string {
	user := "root"
	password := ""
	dsn := fmt.Sprintf("%s:%s@(%s)/%s?charset=utf8", user, password, host, dbName)
	return dsn
}

func initDatabase(host, dbName string) error {
	dsn := getDefaultDSN(host, "")
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.Exec("CREATE DATABASE IF NOT EXISTS " + dbName + ";"); err != nil {
		return err
	}
	return nil
}

func checkDataIsSame(backupDSN, restoreDSN string) error {
	backup, err := sql.Open("mysql", backupDSN)
	if err != nil {
		return err
	}
	defer backup.Close()
	restore, err := sql.Open("mysql", restoreDSN)
	if err != nil {
		return err
	}
	defer restore.Close()

	backupTables, err := getTableList(backup)
	if err != nil {
		return err
	}
	restoreTables, err := getTableList(restore)
	if err != nil {
		return err
	}
	if len(backupTables) != len(restoreTables) {
		return fmt.Errorf("backup tables(%v) is not equal with restore tables(%v)", len(backupTables), len(restoreTables))
	}

	for i := 0; i < len(backupTables); i++ {
		backupTable := backupTables[i]
		restoreTable := restoreTables[i]
		if backupTable != restoreTable {
			return fmt.Errorf("%vth backup table name(%s) is not equal with restore table name(%s)", i, backupTable, restoreTable)
		}

		backupRecordCount, err := getTableRecordCount(backup, backupTable)
		if err != nil {
			return err
		}
		restoreRecordCount, err := getTableRecordCount(restore, restoreTable)
		if err != nil {
			return err
		}

		if backupRecordCount != restoreRecordCount {
			return fmt.Errorf("table(%s) has %v records in backup but %v in restore", backupTable, backupRecordCount, restoreRecordCount)
		}

		x := rand.Intn(backupRecordCount)
		backupRecord, err := getRecord(backup, backupTable, x)
		if err != nil {
			return err
		}
		restoreRecord, err := getRecord(restore, restoreTable, x)
		if err != nil {
			return err
		}
		if backupRecord != restoreRecord {
			return fmt.Errorf("%vth record in table(%s) is not equal", x, backupTable)
		}
	}

	return nil
}

func getRecord(db *sql.DB, table string, x int) (string, error) {
	var bs string
	row := db.QueryRow(fmt.Sprintf("SELECT 'raw_bytes' FROM %s WHERE id = %d", table, x))
	err := row.Scan(&bs)
	if err != nil {
		return "", err
	}
	return bs, nil
}

func getTableRecordCount(db *sql.DB, table string) (int, error) {
	var cnt int
	row := db.QueryRow(fmt.Sprintf("SELECT count(*) FROM %s", table))
	err := row.Scan(&cnt)
	if err != nil {
		return cnt, fmt.Errorf("failed to scan count from %s, %v", table, err)
	}
	return cnt, nil
}

func getTableList(db *sql.DB) ([]string, error) {
	tables := []string{}
	rows, err := db.Query("SHOW TABLES;")
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	return tables, nil
}
