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
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apis/util/config"
	e2eframework "github.com/pingcap/tidb-operator/tests/e2e/br/framework"
	brutil "github.com/pingcap/tidb-operator/tests/e2e/br/framework/br"
	"github.com/pingcap/tidb-operator/tests/e2e/br/utils/portforward"
	e2etc "github.com/pingcap/tidb-operator/tests/e2e/tidbcluster"
	"github.com/pingcap/tidb-operator/tests/e2e/util/db/blockwriter"
	utilginkgo "github.com/pingcap/tidb-operator/tests/e2e/util/ginkgo"
	utilimage "github.com/pingcap/tidb-operator/tests/e2e/util/image"
	nsutil "github.com/pingcap/tidb-operator/tests/e2e/util/ns"
	utiltidbcluster "github.com/pingcap/tidb-operator/tests/e2e/util/tidbcluster"
	"github.com/pingcap/tidb-operator/tests/pkg/fixture"

	"github.com/onsi/ginkgo"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/kubernetes/test/e2e/framework"
)

var (
	tidbReadyTimeout        = time.Minute * 15
	backupCompleteTimeout   = time.Minute * 15
	restoreCompleteTimeout  = time.Minute * 15
	logbackupCatchUpTimeout = time.Minute * 25
)

const (
	typeBR     string = "BR"
	typeDumper string = "Dumper"
)

type option func(t *testcase)

var (
	enableTLS         option = enableTLSWithCA(false)
	enableTLSInsecure option = enableTLSWithCA(true)
)

func enableTLSWithCA(skipCA bool) option {
	return func(t *testcase) {
		t.enableTLS = true
		t.skipCA = skipCA
	}
}

func enableXK8sMode(t *testcase) {
	t.enableXK8sMode = true
}

type testcase struct {
	backupVersion  string
	restoreVersion string
	typ            string
	enableTLS      bool
	skipCA         bool
	enableXK8sMode bool

	// hooks
	configureBackup func(backup *v1alpha1.Backup)
	postBackup      func(backup *v1alpha1.Backup)
}

func newTestCase(backupVersion, restoreVersion string, typ string, opts ...option) *testcase {
	tc := &testcase{
		backupVersion:  backupVersion,
		restoreVersion: restoreVersion,
		typ:            typ,
	}

	for _, opt := range opts {
		opt(tc)
	}

	return tc
}

func (t *testcase) description() string {
	builder := &strings.Builder{}

	builder.WriteString(fmt.Sprintf("[%s][%s To %s]", t.typ, t.backupVersion, t.restoreVersion))

	if t.enableTLS {
		builder.WriteString("[TLS]")
	}
	if t.enableXK8sMode {
		builder.WriteString("[X-K8s TidbCluster]")
	}

	return builder.String()
}

var _ = ginkgo.Describe("Backup and Restore", func() {
	f := e2eframework.NewFramework("br")

	var namespaces []string

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

	cases := []*testcase{
		// latest version BR
		newTestCase(utilimage.TiDBLatest, utilimage.TiDBLatest, typeBR),
		// latest version Dumper
		newTestCase(utilimage.TiDBLatest, utilimage.TiDBLatest, typeDumper),
		// latest version BR and enable TLS
		newTestCase(utilimage.TiDBLatest, utilimage.TiDBLatest, typeBR, enableTLS),
		newTestCase(utilimage.TiDBLatest, utilimage.TiDBLatest, typeBR, enableTLSInsecure),
		// latest version Dumper and enable TLS
		newTestCase(utilimage.TiDBLatest, utilimage.TiDBLatest, typeDumper, enableTLS),
		newTestCase(utilimage.TiDBLatest, utilimage.TiDBLatest, typeDumper, enableTLSInsecure),
	}
	for _, prevVersion := range utilimage.TiDBPreviousVersions {
		cases = append(cases,
			// previous version BR
			newTestCase(prevVersion, prevVersion, typeBR),
			// previous version Dumper
			newTestCase(prevVersion, prevVersion, typeDumper),
			// previous version -> latest version BR
			newTestCase(prevVersion, utilimage.TiDBLatest, typeBR),
			// previous version -> latest version Dumper
			newTestCase(prevVersion, utilimage.TiDBLatest, typeDumper),
		)
	}

	brTest := func(tcase *testcase) {
		enableTLS := tcase.enableTLS
		skipCA := tcase.skipCA
		typ := strings.ToLower(tcase.typ)
		backupVersion := tcase.backupVersion
		restoreVersion := tcase.restoreVersion

		// NOTE: mysql and test will be filtered by default
		dbName := "e2etest"
		backupClusterName := fmt.Sprintf("backup-with-%s-%s", typ, strings.ReplaceAll(backupVersion, ".", "x"))
		restoreClusterName := fmt.Sprintf("restore-with-%s-%s", typ, strings.ReplaceAll(restoreVersion, ".", "x"))
		backupName := backupClusterName
		restoreName := restoreClusterName

		ns := f.Namespace.Name
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		if tcase.enableXK8sMode {
			ginkgo.By("Create TiDB cluster for backup and wait ready")
			err := createXK8sTidbClusterWithComponentsReady(f, namespaces, backupClusterName, backupVersion, enableTLS)
			framework.ExpectNoError(err, "creating TiDB clsuter for backup")

			ginkgo.By("Create TiDB cluster for restore and wait ready")
			err = createXK8sTidbClusterWithComponentsReady(f, namespaces, restoreClusterName, restoreVersion, enableTLS)
			framework.ExpectNoError(err, "creating TiDB clsuter for restore")
		} else {
			ginkgo.By("Create TiDB cluster for backup")
			err := createTidbCluster(f, backupClusterName, backupVersion, enableTLS, skipCA)
			framework.ExpectNoError(err)

			ginkgo.By("Create TiDB cluster for restore")
			err = createTidbCluster(f, restoreClusterName, restoreVersion, enableTLS, skipCA)
			framework.ExpectNoError(err)

			ginkgo.By("Wait for backup TiDB cluster ready")
			err = utiltidbcluster.WaitForTCConditionReady(f.ExtClient, ns, backupClusterName, tidbReadyTimeout, 0)
			framework.ExpectNoError(err)

			ginkgo.By("Wait for restore TiDB cluster ready")
			err = utiltidbcluster.WaitForTCConditionReady(f.ExtClient, ns, restoreClusterName, tidbReadyTimeout, 0)
			framework.ExpectNoError(err)
		}

		ginkgo.By("Forward backup TiDB cluster service")
		backupHost, err := portforward.ForwardOnePort(ctx, f.PortForwarder, ns, getTiDBServiceResourceName(backupClusterName), int(brutil.TiDBServicePort))
		framework.ExpectNoError(err)
		err = initDatabase(backupHost, dbName)
		framework.ExpectNoError(err)

		ginkgo.By("Write data into backup TiDB cluster")
		backupDSN := getDefaultDSN(backupHost, dbName)
		err = blockwriter.New().Write(context.Background(), backupDSN)
		framework.ExpectNoError(err)

		ginkgo.By("Create RBAC for backup and restore")
		err = createRBAC(f)
		framework.ExpectNoError(err)

		ginkgo.By("Create backup")
		backup, err := createBackupAndWaitForComplete(f, backupName, backupClusterName, typ, tcase.configureBackup)
		framework.ExpectNoError(err)

		if tcase.postBackup != nil {
			tcase.postBackup(backup)
		}

		ginkgo.By("Create restore")
		err = createRestoreAndWaitForComplete(f, restoreName, restoreClusterName, typ, backupName, nil)
		framework.ExpectNoError(err)

		ginkgo.By("Forward restore TiDB cluster service")
		restoreHost, err := portforward.ForwardOnePort(ctx, f.PortForwarder, ns, getTiDBServiceResourceName(restoreClusterName), int(brutil.TiDBServicePort))
		framework.ExpectNoError(err)

		ginkgo.By("Validate restore result")
		restoreDSN := getDefaultDSN(restoreHost, dbName)
		err = checkDataIsSame(backupDSN, restoreDSN)
		framework.ExpectNoError(err)
	}

	for i := range cases {
		tcase := cases[i]
		ginkgo.It(tcase.description(), func() {
			brTest(tcase)
		})
	}

	ginkgo.Context("By X-k8s TidbCluster", func() {
		cases := []*testcase{
			// BR with x-k8s tidbcluster
			newTestCase(utilimage.TiDBLatest, utilimage.TiDBLatest, typeBR, enableXK8sMode),
			newTestCase(utilimage.TiDBLatest, utilimage.TiDBLatest, typeBR, enableXK8sMode, enableTLS),
		}

		ginkgo.JustBeforeEach(func() {
			namespaces = []string{f.Namespace.Name, f.Namespace.Name + "-1", f.Namespace.Name + "-2"}
			// create all namespaces except framework's namespace
			for _, ns := range namespaces[1:] {
				ginkgo.By(fmt.Sprintf("Building namespace %s", ns))
				_, existed, err := nsutil.CreateNamespaceIfNeeded(ns, f.ClientSet, nil)
				framework.ExpectEqual(existed, false, fmt.Sprintf("namespace %s is existed", ns))
				framework.ExpectNoError(err, fmt.Sprintf("failed to create namespace %s", ns))
			}
		})

		ginkgo.AfterEach(func() {
			// delete all namespaces if needed except framework's namespace
			if framework.TestContext.DeleteNamespace && (framework.TestContext.DeleteNamespaceOnFailure || !ginkgo.CurrentGinkgoTestDescription().Failed) {
				for _, ns := range namespaces[1:] {
					ginkgo.By(fmt.Sprintf("Destroying namespace %s", ns))
					_, err := nsutil.DeleteNamespace(ns, f.ClientSet)
					framework.ExpectNoError(err, fmt.Sprintf("failed to create namespace %s", ns))
				}
			}
		})

		for i := range cases {
			tcase := cases[i]
			ginkgo.It(tcase.description(), func() {
				brTest(tcase)
			})
		}

	})

	utilginkgo.ContextWhenFocus("Specific Version", func() {
		cases := []*testcase{
			newTestCase(utilimage.TiDBV5x0x0, utilimage.TiDBLatest, typeBR),
			newTestCase(utilimage.TiDBV5x0x2, utilimage.TiDBLatest, typeBR),
		}
		for i := range cases {
			tcase := cases[i]
			ginkgo.It(tcase.description(), func() {
				brTest(tcase)
			})
		}
	})

	ginkgo.It("backup and restore with mixed bucket and prefix", func() {
		middlePath := "mid"

		ns := f.Namespace.Name
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		tcase := newTestCase(utilimage.TiDBLatest, utilimage.TiDBLatest, typeBR)
		tcase.configureBackup = func(backup *v1alpha1.Backup) {
			backup.Spec.StorageProvider.S3.Bucket = path.Join(backup.Spec.StorageProvider.S3.Bucket, middlePath) // bucket add suffix
		}
		tcase.postBackup = func(backup *v1alpha1.Backup) {
			ginkgo.By("Check whether prefix of backup files in storage is right")
			expectedPrefix := path.Join(middlePath, backup.Spec.StorageProvider.S3.Prefix)
			cleaned, err := f.Storage.IsDataCleaned(ctx, ns, expectedPrefix)
			framework.ExpectNoError(err)
			framework.ExpectEqual(cleaned, false, "storage should have data")
		}
	})

	ginkgo.Context("[Backup Clean]", func() {
		ginkgo.It("clean bakcup files with policy Delete", func() {
			backupClusterName := "backup-clean"
			backupVersion := utilimage.TiDBLatest
			enableTLS := false
			skipCA := false
			dbName := "e2etest"
			backupName := backupClusterName
			typ := strings.ToLower(typeBR)

			ns := f.Namespace.Name
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ginkgo.By("Create TiDB cluster for backup")
			err := createTidbCluster(f, backupClusterName, backupVersion, enableTLS, skipCA)
			framework.ExpectNoError(err)

			ginkgo.By("Wait for backup TiDB cluster ready")
			err = utiltidbcluster.WaitForTCConditionReady(f.ExtClient, ns, backupClusterName, tidbReadyTimeout, 0)
			framework.ExpectNoError(err)

			ginkgo.By("Forward backup TiDB cluster service")
			backupHost, err := portforward.ForwardOnePort(ctx, f.PortForwarder, ns, getTiDBServiceResourceName(backupClusterName), int(brutil.TiDBServicePort))
			framework.ExpectNoError(err)
			err = initDatabase(backupHost, dbName)
			framework.ExpectNoError(err)

			ginkgo.By("Write data into backup TiDB cluster")
			backupDSN := getDefaultDSN(backupHost, dbName)
			err = blockwriter.New().Write(context.Background(), backupDSN)
			framework.ExpectNoError(err)

			ginkgo.By("Create RBAC for backup and restore")
			err = createRBAC(f)
			framework.ExpectNoError(err)

			ginkgo.By("Create backup with clean policy Delete")
			backup, err := createBackupAndWaitForComplete(f, backupName, backupClusterName, typ, func(backup *v1alpha1.Backup) {
				backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
			})
			framework.ExpectNoError(err)

			ginkgo.By("Delete backup")
			err = deleteBackup(f, backupName)
			framework.ExpectNoError(err)

			ginkgo.By("Check if all backup files in storage is deleted")
			cleaned, err := f.Storage.IsDataCleaned(ctx, ns, backup.Spec.S3.Prefix) // now we only use s3
			framework.ExpectNoError(err)
			framework.ExpectEqual(cleaned, true, "storage should be cleaned")
		})

	})

	ginkgo.Context("Log Backup Test", func() {
		ginkgo.It("start,truncate,stop log backup", func() {
			backupClusterName := "log-backup"
			backupVersion := utilimage.TiDBLatest
			enableTLS := false
			skipCA := false
			backupName := backupClusterName
			typ := strings.ToLower(typeBR)

			ns := f.Namespace.Name
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ginkgo.By("Create log-backup.enable TiDB cluster for log backup")
			err := createLogBackupEnableTidbCluster(f, backupClusterName, backupVersion, enableTLS, skipCA)
			framework.ExpectNoError(err)

			ginkgo.By("Wait for backup TiDB cluster ready")
			err = utiltidbcluster.WaitForTCConditionReady(f.ExtClient, ns, backupClusterName, tidbReadyTimeout, 0)
			framework.ExpectNoError(err)

			ginkgo.By("Create RBAC for log backup")
			err = createRBAC(f)
			framework.ExpectNoError(err)

			ginkgo.By("Start log backup")
			backup, err := createBackupAndWaitForComplete(f, backupName, backupClusterName, typ, func(backup *v1alpha1.Backup) {
				backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
				backup.Spec.Mode = v1alpha1.BackupModeLog
			})
			framework.ExpectNoError(err)
			framework.ExpectNotEqual(backup.Status.CommitTs, "")

			ginkgo.By("Truncate log backup")
			backup, err = continueLogBackupAndWaitForComplete(f, backup, func(backup *v1alpha1.Backup) {
				backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
				backup.Spec.Mode = v1alpha1.BackupModeLog
				backup.Spec.LogTruncateUntil = time.Now().Format(time.RFC3339)
			})
			framework.ExpectNoError(err)
			framework.ExpectEqual(backup.Status.LogSuccessTruncateUntil, backup.Spec.LogTruncateUntil)

			ginkgo.By("Truncate log backup again")
			backup, err = continueLogBackupAndWaitForComplete(f, backup, func(backup *v1alpha1.Backup) {
				backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
				backup.Spec.Mode = v1alpha1.BackupModeLog
				backup.Spec.LogTruncateUntil = time.Now().Format(time.RFC3339)
			})
			framework.ExpectNoError(err)
			framework.ExpectEqual(backup.Status.LogSuccessTruncateUntil, backup.Spec.LogTruncateUntil)

			ginkgo.By("Stop log backup")
			backup, err = continueLogBackupAndWaitForComplete(f, backup, func(backup *v1alpha1.Backup) {
				backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
				backup.Spec.Mode = v1alpha1.BackupModeLog
				backup.Spec.LogStop = true
			})
			framework.ExpectNoError(err)
			framework.ExpectEqual(backup.Status.Phase, v1alpha1.BackupComplete)

			ginkgo.By("Truncate log backup after stop")
			backup, err = continueLogBackupAndWaitForComplete(f, backup, func(backup *v1alpha1.Backup) {
				backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
				backup.Spec.Mode = v1alpha1.BackupModeLog
				backup.Spec.LogTruncateUntil = time.Now().Format(time.RFC3339)
				backup.Spec.LogStop = false
			})
			framework.ExpectNoError(err)
			framework.ExpectEqual(backup.Status.LogSuccessTruncateUntil, backup.Spec.LogTruncateUntil)

			ginkgo.By("Delete backup")
			err = deleteBackup(f, backupName)
			framework.ExpectNoError(err)

			ginkgo.By("Check if all backup files in storage is deleted")
			cleaned, err := f.Storage.IsDataCleaned(ctx, ns, backup.Spec.S3.Prefix) // now we only use s3
			framework.ExpectNoError(err)
			framework.ExpectEqual(cleaned, true, "storage should be cleaned")
		})

		// TODO: tikv error:[ERROR] [mod.rs:747] ["Status server error: TLS handshake error"], will open this test when this is fixed.
		// ginkgo.It("Log backup progress track with tls cluster", func() {
		// 	backupVersion := utilimage.TiDBLatest
		// 	enableTLS := true
		// 	skipCA := false
		// 	ns := f.Namespace.Name
		// 	ctx, cancel := context.WithCancel(context.Background())
		// 	defer cancel()

		// 	ginkgo.By("Create log-backup.enable TiDB cluster with tls")
		// 	masterClusterName := "tls-master"
		// 	err := createLogBackupEnableTidbCluster(f, masterClusterName, backupVersion, enableTLS, skipCA)
		// 	framework.ExpectNoError(err)
		// 	ginkgo.By("Wait for tls-master TiDB cluster ready")
		// 	err = utiltidbcluster.WaitForTCConditionReady(f.ExtClient, ns, masterClusterName, tidbReadyTimeout, 0)
		// 	framework.ExpectNoError(err)

		// 	ginkgo.By("Create RBAC for backup")
		// 	err = createRBAC(f)
		// 	framework.ExpectNoError(err)

		// 	logBackupName := "log-backup"
		// 	typ := strings.ToLower(typeBR)
		// 	ginkgo.By("Start log backup")
		// 	logBackup, err := createBackupAndWaitForComplete(f, logBackupName, masterClusterName, typ, func(backup *v1alpha1.Backup) {
		// 		backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
		// 		backup.Spec.Mode = v1alpha1.BackupModeLog
		// 	})
		// 	framework.ExpectNoError(err)
		// 	framework.ExpectNotEqual(logBackup.Status.CommitTs, "")

		// 	ginkgo.By("wait log backup progress reach current ts")
		// 	currentTS := strconv.FormatUint(config.GoTimeToTS(time.Now()), 10)
		// 	err = brutil.WaitForLogBackupProgressReachTS(f.ExtClient, ns, logBackupName, currentTS, logbackupCatchUpTimeout)
		// 	framework.ExpectNoError(err)

		// 	ginkgo.By("Delete log backup")
		// 	err = deleteBackup(f, logBackupName)
		// 	framework.ExpectNoError(err)

		// 	ginkgo.By("Check if all log backup files in storage is deleted")
		// 	cleaned, err := f.Storage.IsDataCleaned(ctx, ns, logBackup.Spec.S3.Prefix) // now we only use s3
		// 	framework.ExpectNoError(err)
		// 	framework.ExpectEqual(cleaned, true, "storage should be cleaned")
		// })
	})

	ginkgo.Context("PiTR Restore Test", func() {
		ginkgo.It("Base test of PiRR restore", func() {
			backupVersion := utilimage.TiDBLatest
			enableTLS := false
			skipCA := false
			ns := f.Namespace.Name
			dbName := "e2etest"
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ginkgo.By("Create log-backup.enable TiDB cluster for pitr-master")
			masterClusterName := "pitr-master"
			err := createLogBackupEnableTidbCluster(f, masterClusterName, backupVersion, enableTLS, skipCA)
			framework.ExpectNoError(err)
			ginkgo.By("Wait for pitr-master TiDB cluster ready")
			err = utiltidbcluster.WaitForTCConditionReady(f.ExtClient, ns, masterClusterName, tidbReadyTimeout, 0)
			framework.ExpectNoError(err)

			ginkgo.By("Create RBAC for backup")
			err = createRBAC(f)
			framework.ExpectNoError(err)

			logBackupName := "log-backup"
			typ := strings.ToLower(typeBR)
			ginkgo.By("Start log backup")
			logBackup, err := createBackupAndWaitForComplete(f, logBackupName, masterClusterName, typ, func(backup *v1alpha1.Backup) {
				backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
				backup.Spec.Mode = v1alpha1.BackupModeLog
			})
			framework.ExpectNoError(err)
			framework.ExpectNotEqual(logBackup.Status.CommitTs, "")

			fullBackupName := "full-backup"
			ginkgo.By("Start full backup")
			fullBackup, err := createBackupAndWaitForComplete(f, fullBackupName, masterClusterName, typ, func(backup *v1alpha1.Backup) {
				backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
			})
			framework.ExpectNoError(err)
			framework.ExpectNotEqual(fullBackup.Status.CommitTs, "")

			ginkgo.By("Forward master TiDB cluster service")
			masterHost, err := portforward.ForwardOnePort(ctx, f.PortForwarder, ns, getTiDBServiceResourceName(masterClusterName), int(brutil.TiDBServicePort))
			framework.ExpectNoError(err)
			err = initDatabase(masterHost, dbName)
			framework.ExpectNoError(err)

			ginkgo.By("Write data into master TiDB cluster")
			masterDSN := getDefaultDSN(masterHost, dbName)
			err = blockwriter.New().Write(context.Background(), masterDSN)
			framework.ExpectNoError(err)

			ginkgo.By("Forward master PD service")
			masterPDHost, err := portforward.ForwardOnePort(ctx, f.PortForwarder, ns, getPDServiceResourceName(masterClusterName), int(brutil.PDServicePort))
			framework.ExpectNoError(err)
			ginkgo.By("Wait log backup reach current ts")
			currentTS := strconv.FormatUint(config.GoTimeToTS(time.Now()), 10)
			err = brutil.WaitForLogBackupReachTS(logBackupName, masterPDHost, currentTS, logbackupCatchUpTimeout)
			framework.ExpectNoError(err)

			ginkgo.By("wait log backup progress reach current ts")
			err = brutil.WaitForLogBackupProgressReachTS(f.ExtClient, ns, logBackupName, currentTS, logbackupCatchUpTimeout)
			framework.ExpectNoError(err)

			ginkgo.By("Create log-backup.enable TiDB cluster for pitr-backup")
			backupClusterName := "pitr-backup"
			err = createLogBackupEnableTidbCluster(f, backupClusterName, backupVersion, enableTLS, skipCA)
			framework.ExpectNoError(err)
			ginkgo.By("Wait for pitr-backup TiDB cluster ready")
			err = utiltidbcluster.WaitForTCConditionReady(f.ExtClient, ns, backupClusterName, tidbReadyTimeout, 0)
			framework.ExpectNoError(err)

			ginkgo.By("Create pitr restore")
			restoreName := "pitr-restore"
			err = createRestoreAndWaitForComplete(f, restoreName, backupClusterName, typ, logBackupName, func(restore *v1alpha1.Restore) {
				restore.Spec.Mode = v1alpha1.RestoreModePiTR
				restore.Spec.PitrFullBackupStorageProvider.S3 = fullBackup.Spec.S3
				restore.Spec.PitrRestoredTs = currentTS
			})
			framework.ExpectNoError(err)

			ginkgo.By("wait pitr restore progress done")
			err = brutil.WaitForRestoreProgressDone(f.ExtClient, ns, restoreName, restoreCompleteTimeout)
			framework.ExpectNoError(err)

			ginkgo.By("Forward restore TiDB cluster service")
			backupHost, err := portforward.ForwardOnePort(ctx, f.PortForwarder, ns, getTiDBServiceResourceName(backupClusterName), int(brutil.TiDBServicePort))
			framework.ExpectNoError(err)

			ginkgo.By("Validate pitr restore result")
			backupDSN := getDefaultDSN(backupHost, dbName)
			err = checkDataIsSame(masterDSN, backupDSN)
			framework.ExpectNoError(err)

			ginkgo.By("Delete pitr log backup")
			err = deleteBackup(f, logBackupName)
			framework.ExpectNoError(err)

			ginkgo.By("Check if all pitr log backup files in storage is deleted")
			cleaned, err := f.Storage.IsDataCleaned(ctx, ns, logBackup.Spec.S3.Prefix) // now we only use s3
			framework.ExpectNoError(err)
			framework.ExpectEqual(cleaned, true, "storage should be cleaned")

			ginkgo.By("Delete pitr full backup")
			err = deleteBackup(f, fullBackupName)
			framework.ExpectNoError(err)

			ginkgo.By("Check if all pitr full backup files in storage is deleted")
			cleaned, err = f.Storage.IsDataCleaned(ctx, ns, fullBackup.Spec.S3.Prefix) // now we only use s3
			framework.ExpectNoError(err)
			framework.ExpectEqual(cleaned, true, "storage should be cleaned")
		})
	})
})

func getTiDBServiceResourceName(tcName string) string {
	// TODO: use common util to get tidb service name
	return "svc/" + tcName + "-tidb"
}

func getPDServiceResourceName(tcName string) string {
	// TODO: use common util to get tidb service name
	return "svc/" + tcName + "-pd"
}

func createTidbCluster(f *e2eframework.Framework, name string, version string, enableTLS bool, skipCA bool) error {
	ns := f.Namespace.Name
	// TODO: change to use tidbclusterutil like brutil
	tc := fixture.GetTidbCluster(ns, name, version)
	tc.Spec.PD.Replicas = 1
	tc.Spec.TiKV.Replicas = 1
	tc.Spec.TiDB.Replicas = 1
	if enableTLS {
		tc.Spec.TiDB.TLSClient = &v1alpha1.TiDBTLSClient{Enabled: true}
		tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
		tc.Spec.TiDB.TLSClient.SkipInternalClientCA = skipCA

		if err := f.TLSManager.CreateTLSForTidbCluster(tc); err != nil {
			return err
		}
	}

	if _, err := f.ExtClient.PingcapV1alpha1().TidbClusters(ns).Create(context.TODO(), tc, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}

// createLogBackupEnableTidbCluster create tidb cluster and set "log-backup.enable = true" in tikv to enable log backup.
func createLogBackupEnableTidbCluster(f *e2eframework.Framework, name string, version string, enableTLS bool, skipCA bool) error {
	ns := f.Namespace.Name
	// TODO: change to use tidbclusterutil like brutil
	tc := fixture.GetTidbCluster(ns, name, version)
	tc.Spec.PD.Replicas = 1
	tc.Spec.TiKV.Replicas = 1
	tc.Spec.TiDB.Replicas = 1
	if enableTLS {
		tc.Spec.TiDB.TLSClient = &v1alpha1.TiDBTLSClient{Enabled: true}
		tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
		tc.Spec.TiDB.TLSClient.SkipInternalClientCA = skipCA

		if err := f.TLSManager.CreateTLSForTidbCluster(tc); err != nil {
			return err
		}
	}
	tc.Spec.TiKV.Config.Set("log-backup.enable", true)
	if _, err := f.ExtClient.PingcapV1alpha1().TidbClusters(ns).Create(context.TODO(), tc, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}

func createXK8sTidbClusterWithComponentsReady(f *e2eframework.Framework, namespaces []string, name string, version string, enableTLS bool) error {
	ns1, ns2, ns3 := namespaces[0], namespaces[1], namespaces[2]
	clusterDomain := "cluster.local"
	tc1 := GetTCForXK8s(ns1, name, version, clusterDomain, nil)
	tc2 := GetTCForXK8s(ns2, name, version, clusterDomain, tc1)
	tc3 := GetTCForXK8s(ns3, name, version, clusterDomain, tc1)

	if enableTLS {
		ginkgo.By("Installing initial tidb CA certificate")
		err := e2etc.InstallTiDBIssuer(ns1, name)
		if err != nil {
			return err
		}

		ginkgo.By("Export initial CA secret and install into other tidb clusters")
		var caSecret *v1.Secret
		err = wait.PollImmediate(5*time.Second, 1*time.Minute, func() (bool, error) {
			caSecret, err = f.ClientSet.CoreV1().Secrets(ns1).Get(context.TODO(), fmt.Sprintf("%s-ca-secret", name), metav1.GetOptions{})
			if err != nil {
				return false, nil
			}
			return true, nil
		})
		if err != nil {
			return err
		}
		caSecret.ObjectMeta.ResourceVersion = ""
		caSecret.Namespace = ns2
		err = f.GenericClient.Create(context.TODO(), caSecret)
		if err != nil {
			return err
		}
		caSecret.ObjectMeta.ResourceVersion = ""
		caSecret.Namespace = ns3
		err = f.GenericClient.Create(context.TODO(), caSecret)
		if err != nil {
			return err
		}

		ginkgo.By("Installing tidb cluster issuer with initial ca")
		err = e2etc.InstallXK8sTiDBIssuer(ns2, name, name)
		if err != nil {
			return err
		}
		err = e2etc.InstallXK8sTiDBIssuer(ns3, name, name)
		if err != nil {
			return err
		}

		ginkgo.By("Installing tidb server and client certificate")
		err = e2etc.InstallXK8sTiDBCertificates(ns1, name, clusterDomain)
		if err != nil {
			return err
		}
		err = e2etc.InstallXK8sTiDBCertificates(ns2, name, clusterDomain)
		if err != nil {
			return err
		}
		err = e2etc.InstallXK8sTiDBCertificates(ns3, name, clusterDomain)
		if err != nil {
			return err
		}

		ginkgo.By("Installing tidb components certificates")
		err = e2etc.InstallXK8sTiDBComponentsCertificates(ns1, name, clusterDomain, false)
		if err != nil {
			return err
		}
		err = e2etc.InstallXK8sTiDBComponentsCertificates(ns2, name, clusterDomain, false)
		if err != nil {
			return err
		}
		err = e2etc.InstallXK8sTiDBComponentsCertificates(ns3, name, clusterDomain, false)
		if err != nil {
			return err
		}

		tc1.Spec.TiDB.TLSClient = &v1alpha1.TiDBTLSClient{Enabled: true}
		tc1.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}

		tc2.Spec.TiDB.TLSClient = &v1alpha1.TiDBTLSClient{Enabled: true}
		tc2.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}

		tc3.Spec.TiDB.TLSClient = &v1alpha1.TiDBTLSClient{Enabled: true}
		tc3.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
	}

	ginkgo.By(fmt.Sprintf("Creating tidb cluster in %s", ns1))
	if _, err := f.ExtClient.PingcapV1alpha1().TidbClusters(ns1).Create(context.TODO(), tc1, metav1.CreateOptions{}); err != nil {
		return err
	}
	err := utiltidbcluster.WaitForTCConditionReady(f.ExtClient, ns1, name, tidbReadyTimeout, 0)
	if err != nil {
		return err
	}

	ginkgo.By(fmt.Sprintf("Creating tidb cluster in %s", ns2))
	if _, err := f.ExtClient.PingcapV1alpha1().TidbClusters(ns2).Create(context.TODO(), tc2, metav1.CreateOptions{}); err != nil {
		return err
	}
	err = utiltidbcluster.WaitForTCConditionReady(f.ExtClient, ns2, name, tidbReadyTimeout, 0)
	if err != nil {
		return err
	}

	ginkgo.By(fmt.Sprintf("Creating tidb cluster in %s", ns3))
	if _, err := f.ExtClient.PingcapV1alpha1().TidbClusters(ns3).Create(context.TODO(), tc3, metav1.CreateOptions{}); err != nil {
		return err
	}
	err = utiltidbcluster.WaitForTCConditionReady(f.ExtClient, ns3, name, tidbReadyTimeout, 0)
	if err != nil {
		return err
	}

	ginkgo.By("Check deploy status")
	return e2etc.CheckStatusWhenAcrossK8sWithTimeout(f.ExtClient, []*v1alpha1.TidbCluster{tc1, tc2, tc3}, 5*time.Second, 3*time.Minute)
}

func GetTCForXK8s(ns, name, version, clusterDomain string, joinTC *v1alpha1.TidbCluster) *v1alpha1.TidbCluster {
	tc := fixture.GetTidbCluster(ns, name, version)

	tc.Spec.PD.Replicas = 1
	tc.Spec.TiDB.Replicas = 1
	tc.Spec.TiKV.Replicas = 1

	tc.Spec.ClusterDomain = clusterDomain
	if joinTC != nil {
		tc.Spec.Cluster = &v1alpha1.TidbClusterRef{
			Namespace:     joinTC.Namespace,
			Name:          joinTC.Name,
			ClusterDomain: joinTC.Spec.ClusterDomain,
		}
	}

	return tc
}

func createRBAC(f *e2eframework.Framework) error {
	ns := f.Namespace.Name
	sa := brutil.GetServiceAccount(ns)
	if _, err := f.ClientSet.CoreV1().ServiceAccounts(ns).Create(context.TODO(), sa, metav1.CreateOptions{}); err != nil {
		return err
	}

	role := brutil.GetRole(ns)
	if _, err := f.ClientSet.RbacV1().Roles(ns).Create(context.TODO(), role, metav1.CreateOptions{}); err != nil {
		return err
	}
	rb := brutil.GetRoleBinding(ns)
	if _, err := f.ClientSet.RbacV1().RoleBindings(ns).Create(context.TODO(), rb, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func createBackupAndWaitForComplete(f *e2eframework.Framework, name, tcName, typ string, configure func(*v1alpha1.Backup)) (*v1alpha1.Backup, error) {
	ns := f.Namespace.Name
	// secret to visit tidb cluster
	s := brutil.GetSecret(ns, name, "")
	if _, err := f.ClientSet.CoreV1().Secrets(ns).Create(context.TODO(), s, metav1.CreateOptions{}); err != nil {
		return nil, err
	}

	backupFolder := time.Now().Format(time.RFC3339)
	cfg := f.Storage.Config(ns, backupFolder)
	backup := brutil.GetBackup(ns, name, tcName, typ, cfg)

	if configure != nil {
		configure(backup)
	}

	if _, err := f.ExtClient.PingcapV1alpha1().Backups(ns).Create(context.TODO(), backup, metav1.CreateOptions{}); err != nil {
		return nil, err
	}

	if err := brutil.WaitForBackupComplete(f.ExtClient, ns, name, backupCompleteTimeout); err != nil {
		return backup, err
	}
	return f.ExtClient.PingcapV1alpha1().Backups(ns).Get(context.TODO(), name, metav1.GetOptions{})
}

// continueLogBackupAndWaitForComplete update backup cr to continue run log backup subcommand
func continueLogBackupAndWaitForComplete(f *e2eframework.Framework, backup *v1alpha1.Backup, configure func(*v1alpha1.Backup)) (*v1alpha1.Backup, error) {
	ns := f.Namespace.Name
	name := backup.Name

	if configure != nil {
		configure(backup)
	}

	if _, err := f.ExtClient.PingcapV1alpha1().Backups(ns).Update(context.TODO(), backup, metav1.UpdateOptions{}); err != nil {
		return nil, err
	}

	if err := brutil.WaitForBackupComplete(f.ExtClient, ns, name, backupCompleteTimeout); err != nil {
		return backup, err
	}

	return f.ExtClient.PingcapV1alpha1().Backups(ns).Get(context.TODO(), name, metav1.GetOptions{})
}

func deleteBackup(f *e2eframework.Framework, name string) error {
	ns := f.Namespace.Name

	if err := f.ExtClient.PingcapV1alpha1().Backups(ns).Delete(context.TODO(), name, metav1.DeleteOptions{}); err != nil {
		return err
	}

	if err := brutil.WaitForBackupDeleted(f.ExtClient, ns, name, time.Second*60); err != nil {
		return err
	}
	return nil
}

// nolint
// NOTE: it is not used
func cleanBackup(f *e2eframework.Framework) error {
	ns := f.Namespace.Name
	bl, err := f.ExtClient.PingcapV1alpha1().Backups(ns).List(context.TODO(), metav1.ListOptions{})
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

func createRestoreAndWaitForComplete(f *e2eframework.Framework, name, tcName, typ string, backupName string, configure func(*v1alpha1.Restore)) error {
	ns := f.Namespace.Name

	// secret to visit tidb cluster
	s := brutil.GetSecret(ns, name, "")
	if _, err := f.ClientSet.CoreV1().Secrets(ns).Create(context.TODO(), s, metav1.CreateOptions{}); err != nil {
		return err
	}

	backup, err := f.ExtClient.PingcapV1alpha1().Backups(ns).Get(context.TODO(), backupName, metav1.GetOptions{})
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

	if configure != nil {
		configure(restore)
	}

	if _, err := f.ExtClient.PingcapV1alpha1().Restores(ns).Create(context.TODO(), restore, metav1.CreateOptions{}); err != nil {
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
