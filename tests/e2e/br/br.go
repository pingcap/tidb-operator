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

package br

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	corev1alpha1 "github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/pdapi/v1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	brframework "github.com/pingcap/tidb-operator/tests/e2e/br/framework"
	"github.com/pingcap/tidb-operator/tests/e2e/cluster"
	"github.com/pingcap/tidb-operator/tests/e2e/data"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/db/blockwriter"
	utilimage "github.com/pingcap/tidb-operator/tests/e2e/utils/image"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/k8s"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

var (
	tidbReadyTimeout             = time.Minute * 15
	backupCompleteTimeout        = time.Minute * 15
	restoreCompleteTimeout       = time.Minute * 15
	logbackupCatchUpTimeout      = time.Minute * 25
	logbackupCommandWaitTimeout  = time.Second * 30
	logbackupCommandPollInterval = time.Second * 3
)

const (
	typeBR     string = "BR"
	typeDumper string = "Dumper"
	// TODO use https://github.com/pingcap/failpoint instead e2e test env
	e2eBackupEnv        string = "E2E_TEST_ENV"
	e2eExtendBackupTime string = "E2E_TEST_FLAG_EXTEND_BACKUP_TIME"
	e2eTestFlagPanic    string = "E2E_TEST_FLAG_PANIC"
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

type testcase struct {
	backupVersion  string
	restoreVersion string
	enableTLS      bool
	skipCA         bool

	// hooks
	configureBackup func(backup *v1alpha1.Backup)
	postBackup      func(backup *v1alpha1.Backup)
}

func newTestCase(backupVersion, restoreVersion string, opts ...option) *testcase {
	tc := &testcase{
		backupVersion:  backupVersion,
		restoreVersion: restoreVersion,
	}

	for _, opt := range opts {
		opt(tc)
	}

	return tc
}

func (t *testcase) description() string {
	builder := &strings.Builder{}

	builder.WriteString(fmt.Sprintf("[%s To %s]", t.backupVersion, t.restoreVersion))

	if t.enableTLS {
		builder.WriteString("[TLS]")
	}
	return builder.String()
}

var _ = ginkgo.Describe("Backup and Restore", func() {
	f := brframework.NewFramework("br")
	f.SetupBootstrapSQL("SET PASSWORD FOR 'root'@'%' = 'pingcap';")

	ginkgo.BeforeEach(func() {
		accessKey := "12345678"
		secretKey := "12345678"
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		err := f.Storage.Init(ctx, f.Namespace.Name, accessKey, secretKey)
		f.Must(err)
	})

	ginkgo.JustAfterEach(func() {
		if ginkgo.CurrentSpecReport().Failed() && os.Getenv("PAUSE_ON_FAILURE") == "true" {
			fmt.Println(ginkgo.CurrentSpecReport().FailureMessage())
			fmt.Println("\n\n🛑 TEST FAILED - EXECUTION PAUSED 🛑")
			fmt.Println("Press Ctrl+C to exit")

			// Wait for SIGINT (Ctrl+C)
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt, syscall.SIGTERM)
			<-c
		}
		err := f.Storage.Clean(context.Background(), f.Namespace.Name)
		f.Must(err)
	})

	cases := []*testcase{
		// latest version BR and enable TLS
		newTestCase(utilimage.TiDBLatest, utilimage.TiDBLatest, enableTLS),

		// latest version BR
		newTestCase(utilimage.TiDBLatest, utilimage.TiDBLatest),
	}
	for _, prevVersion := range utilimage.TiDBPreviousVersions {
		cases = append(cases,
			// previous version BR
			newTestCase(prevVersion, prevVersion),
			// previous version -> latest version BR
			newTestCase(prevVersion, utilimage.TiDBLatest),
		)
	}

	brTest := func(tcase *testcase) {
		enableTLS := tcase.enableTLS
		skipCA := tcase.skipCA
		backupVersion := tcase.backupVersion
		restoreVersion := tcase.restoreVersion

		// NOTE: mysql and test will be filtered by default
		dbName := "e2etest"
		backupClusterName := fmt.Sprintf("backup-with-%s", strings.ReplaceAll(backupVersion, ".", "x"))
		restoreClusterName := fmt.Sprintf("restore-with-%s", strings.ReplaceAll(restoreVersion, ".", "x"))
		backupName := backupClusterName
		restoreName := restoreClusterName

		ns := f.Namespace.Name
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		{
			ginkgo.By("Create TiDB cluster for backup")
			err := createTidbCluster(f, backupClusterName, backupVersion, enableTLS, skipCA)
			f.Must(err)

			ginkgo.By("Create TiDB cluster for restore")
			err = createTidbCluster(f, restoreClusterName, restoreVersion, enableTLS, skipCA)
			f.Must(err)

			ginkgo.By("Wait for backup TiDB cluster ready")
			err = waiter.WaitForClusterReady(ctx, f.Client, ns, backupClusterName, tidbReadyTimeout)
			f.Must(err)

			ginkgo.By("Wait for restore TiDB cluster ready")
			err = waiter.WaitForClusterReady(ctx, f.Client, ns, restoreClusterName, tidbReadyTimeout)
			f.Must(err)
		}

		ginkgo.By("Forward backup TiDB cluster service")
		backupHost, backupPort, cancel, err := k8s.ForwardOnePort(f.PortForwarder, ns, getTiDBServiceResourceName(backupClusterName), corev1alpha1.DefaultTiDBPortClient)
		f.Must(err)
		defer cancel()
		backupDomain := fmt.Sprintf("%s:%d", backupHost, backupPort)
		err = initDatabase(backupDomain, dbName)
		f.Must(err)

		ginkgo.By("Write data into backup TiDB cluster")
		backupDSN := getDefaultDSN(backupDomain, dbName)
		err = blockwriter.New().Write(context.Background(), backupDSN)
		f.Must(err)

		ginkgo.By("Create RBAC for backup and restore")
		err = createRBAC(f)
		f.Must(err)

		ginkgo.By("Create backup")
		backup, err := createBackupAndWaitForComplete(f, backupName, backupClusterName, tcase.configureBackup)
		f.Must(err)

		if tcase.postBackup != nil {
			tcase.postBackup(backup)
		}

		ginkgo.By("Create restore")
		err = createRestoreAndWaitForComplete(f, restoreName, restoreClusterName, backupName, nil)
		f.Must(err)

		ginkgo.By("Forward restore TiDB cluster service")
		restoreHost, restorePort, cancel, err := k8s.ForwardOnePort(f.PortForwarder, ns, getTiDBServiceResourceName(restoreClusterName), corev1alpha1.DefaultTiDBPortClient)
		f.Must(err)
		defer cancel()
		restoreDomain := fmt.Sprintf("%s:%d", restoreHost, restorePort)
		ginkgo.By("Validate restore result")
		restoreDSN := getDefaultDSN(restoreDomain, dbName)
		err = checkDataIsSame(backupDSN, restoreDSN)
		f.Must(err)
	}

	for i := range cases {
		tcase := cases[i]
		ginkgo.It("[Matrix] "+tcase.description(), func() {
			brTest(tcase)
		})
	}

	ginkgo.Context("Specific Version", func() {
		cases := []*testcase{
			// newTestCase(utilimage.TiDBV7x5x3, utilimage.TiDBLatest),
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

		tcase := newTestCase(utilimage.TiDBLatest, utilimage.TiDBLatest)
		tcase.configureBackup = func(backup *v1alpha1.Backup) {
			backup.Spec.StorageProvider.S3.Bucket = path.Join(backup.Spec.StorageProvider.S3.Bucket, middlePath) // bucket add suffix
		}
		tcase.postBackup = func(backup *v1alpha1.Backup) {
			ginkgo.By("Check whether prefix of backup files in storage is right")
			expectedPrefix := path.Join(middlePath, backup.Spec.StorageProvider.S3.Prefix)
			cleaned, err := f.Storage.IsDataCleaned(ctx, ns, expectedPrefix)
			f.Must(err)
			ExpectEqual(cleaned, false, "storage should have data")
		}
	})

	ginkgo.Context("Backup Clean", func() {
		ginkgo.It("clean bakcup files with policy Delete", func() {
			backupClusterName := "backup-clean"
			backupVersion := utilimage.TiDBLatest
			enableTLS := false
			skipCA := false
			dbName := "e2etest"
			backupName := backupClusterName

			ns := f.Namespace.Name
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ginkgo.By("Create TiDB cluster for backup")
			err := createTidbCluster(f, backupClusterName, backupVersion, enableTLS, skipCA)
			f.Must(err)

			ginkgo.By("Wait for backup TiDB cluster ready")
			err = waiter.WaitForClusterReady(ctx, f.Client, ns, backupClusterName, tidbReadyTimeout)
			f.Must(err)

			ginkgo.By("Forward backup TiDB cluster service")
			backupHost, backupPort, cancel, err := k8s.ForwardOnePort(f.PortForwarder, ns, getTiDBServiceResourceName(backupClusterName), corev1alpha1.DefaultTiDBPortClient)
			f.Must(err)
			defer cancel()
			backupDomain := fmt.Sprintf("%s:%d", backupHost, backupPort)
			err = initDatabase(backupDomain, dbName)
			f.Must(err)

			ginkgo.By("Write data into backup TiDB cluster")
			backupDSN := getDefaultDSN(backupDomain, dbName)
			err = blockwriter.New().Write(context.Background(), backupDSN)
			f.Must(err)

			ginkgo.By("Create RBAC for backup and restore")
			err = createRBAC(f)
			f.Must(err)

			ginkgo.By("Create backup with clean policy Delete")
			backup, err := createBackupAndWaitForComplete(f, backupName, backupClusterName, func(backup *v1alpha1.Backup) {
				backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
			})
			f.Must(err)

			ginkgo.By("Delete backup")
			err = deleteBackup(f, backupName)
			f.Must(err)

			ginkgo.By("Check if all backup files in storage is deleted")
			cleaned, err := f.Storage.IsDataCleaned(ctx, ns, backup.Spec.S3.Prefix) // now we only use s3
			f.Must(err)
			ExpectEqual(cleaned, true, "storage should be cleaned")
		})

	})

	ginkgo.Context("Log Backup Test", func() {
		// in other worlds, don't use logSubcommand for stop, truncate, start, etc
		ginkgo.It("start,truncate,stop log backup using old interface", func() {
			backupClusterName := "log-backup"
			backupVersion := utilimage.TiDBLatest
			enableTLS := false
			skipCA := false
			backupName := backupClusterName
			ns := f.Namespace.Name
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ginkgo.By("Create log-backup.enable TiDB cluster for log backup")
			err := createLogBackupEnabledTidbCluster(f, backupClusterName, backupVersion, enableTLS, skipCA)
			f.Must(err)

			ginkgo.By("Wait for backup TiDB cluster ready")
			err = waiter.WaitForClusterReady(ctx, f.Client, ns, backupClusterName, tidbReadyTimeout)
			f.Must(err)
			ginkgo.By("Wait for PD ready")
			err = WaitForPDReady(f, ns, backupClusterName, 2*time.Minute)
			f.Must(err)

			ginkgo.By("Create RBAC for log backup")
			err = createRBAC(f)
			f.Must(err)

			ginkgo.By("Start log backup")
			backup, err := createBackupAndWaitForComplete(f, backupName, backupClusterName, func(backup *v1alpha1.Backup) {
				backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
				backup.Spec.Mode = v1alpha1.BackupModeLog
			})
			f.Must(err)
			ExpectNotEqual(backup.Status.CommitTs, "")

			ginkgo.By("Truncate log backup")
			backup, err = continueLogBackupAndWaitForComplete(f, backup, func(backup *v1alpha1.Backup) {
				backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
				backup.Spec.Mode = v1alpha1.BackupModeLog
				commitTS, _ := v1alpha1.ParseTSString(backup.Status.CommitTs)
				backup.Spec.LogTruncateUntil = strconv.FormatUint(commitTS+1000, 10)
			})
			f.Must(err)
			ExpectEqual(backup.Status.LogSuccessTruncateUntil, backup.Spec.LogTruncateUntil)

			ginkgo.By("Truncate log backup again")
			backup, err = continueLogBackupAndWaitForComplete(f, backup, func(backup *v1alpha1.Backup) {
				backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
				backup.Spec.Mode = v1alpha1.BackupModeLog
				lastTruncateUntil, _ := v1alpha1.ParseTSString(backup.Status.LogSuccessTruncateUntil)
				backup.Spec.LogTruncateUntil = strconv.FormatUint(lastTruncateUntil+1000, 10)
			})
			f.Must(err)
			ExpectEqual(backup.Status.LogSuccessTruncateUntil, backup.Spec.LogTruncateUntil)

			ginkgo.By("Stop log backup")
			backup, err = continueLogBackupAndWaitForComplete(f, backup, func(backup *v1alpha1.Backup) {
				backup.Spec.LogStop = true
				backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
				backup.Spec.Mode = v1alpha1.BackupModeLog
			})
			f.Must(err)
			ExpectEqual(backup.Status.Phase, v1alpha1.BackupStopped)

			ginkgo.By("Truncate log backup after stop")
			backup, err = continueLogBackupAndWaitForComplete(f, backup, func(backup *v1alpha1.Backup) {
				backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
				backup.Spec.Mode = v1alpha1.BackupModeLog
				backup.Spec.LogTruncateUntil = time.Now().Format(time.RFC3339)
			})
			f.Must(err)
			ExpectEqual(backup.Status.LogSuccessTruncateUntil, backup.Spec.LogTruncateUntil)

			ginkgo.By("Delete backup")
			err = deleteBackup(f, backupName)
			f.Must(err)

			ginkgo.By("Check if all backup files in storage is deleted")
			cleaned, err := f.Storage.IsDataCleaned(ctx, ns, backup.Spec.S3.Prefix) // now we only use s3
			f.Must(err)
			ExpectEqual(cleaned, true, "storage should be cleaned")
		})

		ginkgo.It("start -> pause -> resume -> pause -> resume -> stop log backup", func() {
			backupClusterName := "log-backup"
			backupVersion := utilimage.TiDBLatest
			enableTLS := false
			skipCA := false
			backupName := backupClusterName
			ns := f.Namespace.Name
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ginkgo.By("Create log-backup.enable TiDB cluster for log backup")
			err := createLogBackupEnabledTidbCluster(f, backupClusterName, backupVersion, enableTLS, skipCA)
			f.Must(err)

			ginkgo.By("Wait for backup TiDB cluster ready")
			err = waiter.WaitForClusterReady(ctx, f.Client, ns, backupClusterName, tidbReadyTimeout)
			f.Must(err)
			ginkgo.By("Wait for PD ready")
			err = WaitForPDReady(f, ns, backupClusterName, 2*time.Minute)
			f.Must(err)

			ginkgo.By("Create RBAC for log backup")
			err = createRBAC(f)
			f.Must(err)

			ginkgo.By("Start log backup")
			backup, err := createBackupAndWaitForComplete(f, backupName, backupClusterName, func(backup *v1alpha1.Backup) {
				backup.Spec.LogSubcommand = v1alpha1.LogStartCommand
				backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
				backup.Spec.Mode = v1alpha1.BackupModeLog
			})
			f.Must(err)
			ExpectNotEqual(backup.Status.CommitTs, "")

			ginkgo.By("Truncate log backup")
			backup, err = continueLogBackupAndWaitForComplete(f, backup, func(backup *v1alpha1.Backup) {
				backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
				backup.Spec.Mode = v1alpha1.BackupModeLog
				backup.Spec.LogTruncateUntil = time.Now().Format(time.RFC3339)
			})
			f.Must(err)
			ExpectEqual(backup.Status.LogSuccessTruncateUntil, backup.Spec.LogTruncateUntil)

			ginkgo.By("Pause log backup")
			backup, err = continueLogBackupAndWaitForComplete(f, backup, func(backup *v1alpha1.Backup) {
				backup.Spec.LogSubcommand = v1alpha1.LogPauseCommand
				backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
				backup.Spec.Mode = v1alpha1.BackupModeLog
			})
			f.Must(err)
			ExpectEqual(backup.Status.Phase, v1alpha1.BackupPaused)

			ginkgo.By("Truncate log backup again")
			backup, err = continueLogBackupAndWaitForComplete(f, backup, func(backup *v1alpha1.Backup) {
				backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
				backup.Spec.Mode = v1alpha1.BackupModeLog
				backup.Spec.LogTruncateUntil = time.Now().Format(time.RFC3339)
			})
			f.Must(err)
			ExpectEqual(backup.Status.LogSuccessTruncateUntil, backup.Spec.LogTruncateUntil)

			ginkgo.By("resume log backup")
			backup, err = continueLogBackupAndWaitForComplete(f, backup, func(backup *v1alpha1.Backup) {
				backup.Spec.LogSubcommand = v1alpha1.LogStartCommand
				backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
				backup.Spec.Mode = v1alpha1.BackupModeLog
			})
			f.Must(err)
			ExpectEqual(backup.Status.Phase, v1alpha1.BackupRunning)

			ginkgo.By("Pause log backup again")
			backup, err = continueLogBackupAndWaitForComplete(f, backup, func(backup *v1alpha1.Backup) {
				backup.Spec.LogSubcommand = v1alpha1.LogPauseCommand
				backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
				backup.Spec.Mode = v1alpha1.BackupModeLog
			})
			f.Must(err)
			ExpectEqual(backup.Status.Phase, v1alpha1.BackupPaused)

			ginkgo.By("resume log backup again")
			backup, err = continueLogBackupAndWaitForComplete(f, backup, func(backup *v1alpha1.Backup) {
				backup.Spec.LogSubcommand = v1alpha1.LogStartCommand
				backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
				backup.Spec.Mode = v1alpha1.BackupModeLog
			})
			f.Must(err)
			ExpectEqual(backup.Status.Phase, v1alpha1.BackupRunning)

			ginkgo.By("Stop log backup")
			backup, err = continueLogBackupAndWaitForComplete(f, backup, func(backup *v1alpha1.Backup) {
				backup.Spec.LogSubcommand = v1alpha1.LogStopCommand
				backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
				backup.Spec.Mode = v1alpha1.BackupModeLog
			})
			f.Must(err)
			ExpectEqual(backup.Status.Phase, v1alpha1.BackupStopped)

			ginkgo.By("Truncate log backup after stop")
			backup, err = continueLogBackupAndWaitForComplete(f, backup, func(backup *v1alpha1.Backup) {
				backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
				backup.Spec.Mode = v1alpha1.BackupModeLog
				backup.Spec.LogTruncateUntil = time.Now().Format(time.RFC3339)
			})
			f.Must(err)
			ExpectEqual(backup.Status.LogSuccessTruncateUntil, backup.Spec.LogTruncateUntil)

			ginkgo.By("Delete backup")
			err = deleteBackup(f, backupName)
			f.Must(err)

			ginkgo.By("Check if all backup files in storage is deleted")
			cleaned, err := f.Storage.IsDataCleaned(ctx, ns, backup.Spec.S3.Prefix) // now we only use s3
			f.Must(err)
			ExpectEqual(cleaned, true, "storage should be cleaned")
		})

		ginkgo.It("delete log backup CR could stop on-going task", func() {
			backupClusterName := "log-backup"
			backupVersion := utilimage.TiDBLatest
			enableTLS := false
			skipCA := false
			backupName := backupClusterName

			ns := f.Namespace.Name
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ginkgo.By("Create log-backup.enable TiDB cluster for log backup")
			err := createLogBackupEnabledTidbCluster(f, backupClusterName, backupVersion, enableTLS, skipCA)
			f.Must(err)

			ginkgo.By("Wait for backup TiDB cluster ready")
			err = waiter.WaitForClusterReady(ctx, f.Client, ns, backupClusterName, tidbReadyTimeout)
			f.Must(err)
			ginkgo.By("Wait for PD ready")
			err = WaitForPDReady(f, ns, backupClusterName, 2*time.Minute)
			f.Must(err)

			ginkgo.By("Create RBAC for log backup")
			err = createRBAC(f)
			f.Must(err)

			// first round //
			ginkgo.By("Start log backup")
			backup, err := createBackupAndWaitForComplete(f, backupName, backupClusterName, func(backup *v1alpha1.Backup) {
				backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
				backup.Spec.Mode = v1alpha1.BackupModeLog
			})
			f.Must(err)
			ExpectNotEqual(backup.Status.CommitTs, "")
			ExpectEqual(backup.Status.Phase, v1alpha1.BackupRunning)

			ginkgo.By("Delete backup")
			err = deleteBackup(f, backupName)
			f.Must(err)

			ginkgo.By("Check if all backup files in storage is deleted")
			cleaned, err := f.Storage.IsDataCleaned(ctx, ns, backup.Spec.S3.Prefix) // now we only use s3
			f.Must(err)
			ExpectEqual(cleaned, true, "storage should be cleaned")

			// second round: start -> pause -> delete //
			ginkgo.By("Start log backup the second time")
			backup, err = createBackupAndWaitForComplete(f, backupName, backupClusterName, func(backup *v1alpha1.Backup) {
				backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
				backup.Spec.Mode = v1alpha1.BackupModeLog
			})
			f.Must(err)
			ExpectNotEqual(backup.Status.CommitTs, "")

			ginkgo.By("Pause log backup")
			backup, err = continueLogBackupAndWaitForComplete(f, backup, func(backup *v1alpha1.Backup) {
				backup.Spec.LogSubcommand = v1alpha1.LogPauseCommand
				backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
				backup.Spec.Mode = v1alpha1.BackupModeLog
			})
			f.Must(err)
			ExpectEqual(backup.Status.Phase, v1alpha1.BackupPaused)

			ginkgo.By("Delete backup")
			err = deleteBackup(f, backupName)
			f.Must(err)

			ginkgo.By("Check if all backup files in storage is deleted")
			cleaned, err = f.Storage.IsDataCleaned(ctx, ns, backup.Spec.S3.Prefix) // now we only use s3
			f.Must(err)
			ExpectEqual(cleaned, true, "storage should be cleaned")

			// third round: start -> stop -> delete//
			ginkgo.By("Start log backup the third time")
			backup, err = createBackupAndWaitForComplete(f, backupName, backupClusterName, func(backup *v1alpha1.Backup) {
				backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
				backup.Spec.Mode = v1alpha1.BackupModeLog
			})
			f.Must(err)
			ExpectNotEqual(backup.Status.CommitTs, "")

			ginkgo.By("Stop log backup")
			backup, err = continueLogBackupAndWaitForComplete(f, backup, func(backup *v1alpha1.Backup) {
				backup.Spec.LogSubcommand = v1alpha1.LogStopCommand
				backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
				backup.Spec.Mode = v1alpha1.BackupModeLog
			})
			f.Must(err)
			ExpectEqual(backup.Status.Phase, v1alpha1.BackupStopped)

			ginkgo.By("Delete backup")
			err = deleteBackup(f, backupName)
			f.Must(err)

			ginkgo.By("Check if all backup files in storage is deleted")
			cleaned, err = f.Storage.IsDataCleaned(ctx, ns, backup.Spec.S3.Prefix) // now we only use s3
			f.Must(err)
			ExpectEqual(cleaned, true, "storage should be cleaned")
		})

		ginkgo.It("test stop the log backup on schedule phase", func() {
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
			err := createLogBackupEnabledTidbCluster(f, backupClusterName, backupVersion, enableTLS, skipCA)
			f.Must(err)

			ginkgo.By("Wait for backup TiDB cluster ready")
			err = waiter.WaitForClusterReady(ctx, f.Client, ns, backupClusterName, tidbReadyTimeout)
			f.Must(err)
			ginkgo.By("Wait for PD ready")
			err = WaitForPDReady(f, ns, backupClusterName, 2*time.Minute)
			f.Must(err)

			ginkgo.By("Create RBAC for log backup")
			err = createRBAC(f)
			f.Must(err)

			ginkgo.By("Start log backup")
			backup, err := createBackupAndWaitForSchedule(f, backupName, backupClusterName, typ, func(backup *v1alpha1.Backup) {
				// A tricky way to keep the log backup from starting.
				backup.Spec.ToolImage = "null:invalid"
				backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeRetain
				backup.Spec.Mode = v1alpha1.BackupModeLog
			})
			f.Must(err)
			ExpectEqual(backup.Status.CommitTs, "")

			ginkgo.By("Delete backup")
			err = deleteBackup(f, backupName)
			f.Must(err)

			// To make sure the task is deleted.
			ginkgo.By("Start log backup the second time")
			backup, err = createBackupAndWaitForComplete(f, backupName, backupClusterName, func(backup *v1alpha1.Backup) {
				backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
				backup.Spec.Mode = v1alpha1.BackupModeLog
			})
			f.Must(err)
			ExpectNotEqual(backup.Status.CommitTs, "")

			ginkgo.By("Delete backup")
			err = deleteBackup(f, backupName)
			f.Must(err)

			ginkgo.By("Check if all backup files in storage is deleted")
			cleaned, err := f.Storage.IsDataCleaned(ctx, ns, backup.Spec.S3.Prefix) // now we only use s3
			f.Must(err)
			ExpectEqual(cleaned, true, "storage should be cleaned")
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
		// 	k8se2e.ExpectNoError(err)
		// 	ginkgo.By("Wait for tls-master TiDB cluster ready")
		// 	err = waiter.WaitForClusterReady(ctx, f.Client, ns, masterClusterName, tidbReadyTimeout)
		// 	k8se2e.ExpectNoError(err)

		// 	ginkgo.By("Create RBAC for backup")
		// 	err = createRBAC(f)
		// 	k8se2e.ExpectNoError(err)

		// 	logBackupName := "log-backup"
		// 	typ := strings.ToLower(typeBR)
		// 	ginkgo.By("Start log backup")
		// 	logBackup, err := createBackupAndWaitForComplete(f, logBackupName, masterClusterName, typ, func(backup *v1alpha1.Backup) {
		// 		backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
		// 		backup.Spec.Mode = v1alpha1.BackupModeLog
		// 	})
		// 	k8se2e.ExpectNoError(err)
		// 	k8se2e.ExpectNotEqual(logBackup.Status.CommitTs, "")

		// 	ginkgo.By("wait log backup progress reach current ts")
		// 	currentTS := strconv.FormatUint(config.GoTimeToTS(time.Now()), 10)
		// 	err = brframework.WaitForLogBackupProgressReachTS(f.ExtClient, ns, logBackupName, currentTS, logbackupCatchUpTimeout)
		// 	k8se2e.ExpectNoError(err)

		// 	ginkgo.By("Delete log backup")
		// 	err = deleteBackup(f, logBackupName)
		// 	k8se2e.ExpectNoError(err)

		// 	ginkgo.By("Check if all log backup files in storage is deleted")
		// 	cleaned, err := f.Storage.IsDataCleaned(ctx, ns, logBackup.Spec.S3.Prefix) // now we only use s3
		// 	k8se2e.ExpectNoError(err)
		// 	k8se2e.ExpectEqual(cleaned, true, "storage should be cleaned")
		// })
	})

	// the following cases may encounter errors after restarting the backup pod:
	// "there may be some backup files in the path already, please specify a correct backup directory"
	ginkgo.Context("Restart Backup by k8s Test", func() {
		ginkgo.It("delete backup pod and restart by k8s test", func() {
			ginkgo.Skip("unstable case, after restart: there may be some backup files in the path already, please specify a correct backup directory")

			backupClusterName := "delete-backup-pod-test"
			backupVersion := utilimage.TiDBLatest
			enableTLS := false
			skipCA := false
			backupName := backupClusterName

			ns := f.Namespace.Name
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ginkgo.By("Create TiDB cluster")
			err := createTidbCluster(f, backupClusterName, backupVersion, enableTLS, skipCA)
			f.Must(err)

			ginkgo.By("Wait for TiDB cluster ready")
			err = waiter.WaitForClusterReady(ctx, f.Client, ns, backupClusterName, tidbReadyTimeout)
			f.Must(err)

			ginkgo.By("Create RBAC for backup")
			err = createRBAC(f)
			f.Must(err)

			ginkgo.By("Start backup and wait to running")
			backup, err := createBackupAndWaitForRunning(f, backupName, backupClusterName, func(backup *v1alpha1.Backup) {
				backup.Spec.Env = []v1.EnvVar{v1.EnvVar{Name: e2eBackupEnv, Value: e2eExtendBackupTime}}
			})
			f.Must(err)

			ginkgo.By("delete backup pod")
			err = brframework.WaitAndDeleteRunningBackupPod(f, backup, backupCompleteTimeout)
			f.Must(err)

			ginkgo.By("wait auto restart backup pod until backup complete")
			err = brframework.WaitForBackupComplete(f.Client, ns, backupName, backupCompleteTimeout)
			f.Must(err)

			// the behavior of recreate a pod when the old one is deleted for a Job has been changed since k8s 1.25.
			// ref:
			// - https://github.com/kubernetes/kubernetes/pull/110948
			// - https://github.com/kubernetes/kubernetes/blob/v1.25.0/pkg/controller/job/job_controller.go#L720-L727
			// - https://github.com/kubernetes/kubernetes/blob/v1.25.0/pkg/controller/job/job_controller.go#L781-L785
			ginkgo.By("make sure it's not restarted by backoff retry policy")
			num, err := getBackoffRetryNum(f, backup)
			f.Must(err)
			ExpectEqual(num <= 1, true)

			ginkgo.By("Delete backup")
			err = deleteBackup(f, backupName)
			f.Must(err)

			ginkgo.By("Check if all backup files in storage is deleted")
			cleaned, err := f.Storage.IsDataCleaned(ctx, ns, backup.Spec.S3.Prefix) // now we only use s3
			f.Must(err)
			ExpectEqual(cleaned, true, "storage should be cleaned")
		})
	})

	ginkgo.Context("Backoff retry policy Test", func() {
		ginkgo.It("kill backup pod and restart by backoff retry policy", func() {
			// ginkgo.Skip("unstable case, after restart: there may be some backup files in the path already, please specify a correct backup directory")

			backupClusterName := "kill-backup-pod-test"
			backupVersion := utilimage.TiDBLatest
			enableTLS := false
			skipCA := false
			backupName := backupClusterName

			ns := f.Namespace.Name
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ginkgo.By("Create TiDB cluster")
			err := createTidbCluster(f, backupClusterName, backupVersion, enableTLS, skipCA)
			f.Must(err)

			ginkgo.By("Wait for TiDB cluster ready")
			err = waiter.WaitForClusterReady(ctx, f.Client, ns, backupClusterName, tidbReadyTimeout)
			f.Must(err)

			ginkgo.By("Create RBAC for backup")
			err = createRBAC(f)
			f.Must(err)

			ginkgo.By("Start backup and wait to running")
			backup, err := createBackupAndWaitForRunning(f, backupName, backupClusterName, func(backup *v1alpha1.Backup) {
				backup.Spec.BackoffRetryPolicy.MinRetryDuration = "2s" // retry after 2s

				backup.Spec.Env = []v1.EnvVar{v1.EnvVar{Name: e2eBackupEnv, Value: e2eExtendBackupTime + "," + e2eTestFlagPanic}}
			})
			f.Must(err)

			ginkgo.By("wait backup pod failed by simulate panic")
			err = brframework.WaitBackupPodOnPhase(f, backup, v1.PodFailed, backupCompleteTimeout)
			f.Must(err)

			ginkgo.By("update backup evn, remove simulate panic")
			backup, err = updateBackup(f, backup.Name, func(backup *v1alpha1.Backup) {
				backup.Spec.Env = []v1.EnvVar{v1.EnvVar{Name: "a", Value: "b"}}
			})
			f.Must(err)

			ginkgo.By("wait auto restart backup pod until running again")
			err = brframework.WaitBackupPodOnPhase(f, backup, v1.PodRunning, backupCompleteTimeout)
			f.Must(err)

			ginkgo.By("make sure it's restarted by backoff retry policy")
			num, err := getBackoffRetryNum(f, backup)
			f.Must(err)
			ExpectEqual(num, 1)

			ginkgo.By("wait auto restart backup pod until backup complete")
			err = brframework.WaitForBackupComplete(f.Client, ns, backupName, backupCompleteTimeout)
			f.Must(err)

			ginkgo.By("Delete backup")
			err = deleteBackup(f, backupName)
			f.Must(err)

			ginkgo.By("Check if all backup files in storage is deleted")
			cleaned, err := f.Storage.IsDataCleaned(ctx, ns, backup.Spec.S3.Prefix) // now we only use s3
			f.Must(err)
			ExpectEqual(cleaned, true, "storage should be cleaned")
		})

		ginkgo.It("kill backup pod and exceed maxRetryTimes", func() {
			// ginkgo.Skip("unstable case, after restart: there may be some backup files in the path already, please specify a correct backup directory")

			backupClusterName := "kill-backup-pod-exceed-times-test"
			backupVersion := utilimage.TiDBLatest
			enableTLS := false
			skipCA := false
			backupName := backupClusterName

			ns := f.Namespace.Name
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ginkgo.By("Create TiDB cluster")
			err := createTidbCluster(f, backupClusterName, backupVersion, enableTLS, skipCA)
			f.Must(err)

			ginkgo.By("Wait for TiDB cluster ready")
			err = waiter.WaitForClusterReady(ctx, f.Client, ns, backupClusterName, tidbReadyTimeout)
			f.Must(err)

			ginkgo.By("Create RBAC for backup")
			err = createRBAC(f)
			f.Must(err)

			ginkgo.By("Start backup and wait to running")
			backup, err := createBackupAndWaitForRunning(f, backupName, backupClusterName, func(backup *v1alpha1.Backup) {
				backup.Spec.BackoffRetryPolicy = v1alpha1.BackoffRetryPolicy{
					MinRetryDuration: "2s",
					MaxRetryTimes:    2,
					RetryTimeout:     "30m",
				}
				backup.Spec.Env = []v1.EnvVar{v1.EnvVar{Name: e2eBackupEnv, Value: e2eExtendBackupTime + "," + e2eTestFlagPanic}}
			})
			f.Must(err)

			ginkgo.By("wait backup pod failed by simulate panic")
			err = brframework.WaitBackupPodOnPhase(f, backup, v1.PodFailed, backupCompleteTimeout)
			f.Must(err)

			ginkgo.By("wait auto restart backup pod until running again")
			err = brframework.WaitBackupPodOnPhase(f, backup, v1.PodRunning, backupCompleteTimeout)
			f.Must(err)

			ginkgo.By("make sure it's restarted by backoff retry policy")
			f.Must(brframework.GetAndCheckBackup(f.Client, ns, backupName, func(backup *v1alpha1.Backup) bool {
				num, err := getBackoffRetryNum(f, backup)
				f.Must(err)
				return num == 1
			}))

			ginkgo.By("wait backup pod failed by simulate panic")
			err = brframework.WaitBackupPodOnPhase(f, backup, v1.PodFailed, backupCompleteTimeout)
			f.Must(err)

			ginkgo.By("wait auto restart backup pod until running again")
			err = brframework.WaitBackupPodOnPhase(f, backup, v1.PodRunning, backupCompleteTimeout)
			f.Must(err)

			ginkgo.By("make sure it's restarted by backoff retry policy")
			f.Must(brframework.GetAndCheckBackup(f.Client, ns, backupName, func(backup *v1alpha1.Backup) bool {
				num, err := getBackoffRetryNum(f, backup)
				f.Must(err)
				return num == 2
			}))

			ginkgo.By("wait backup pod failed by simulate panic")
			err = brframework.WaitBackupPodOnPhase(f, backup, v1.PodFailed, backupCompleteTimeout)
			f.Must(err)

			ginkgo.By("wait auto restart backup pod until backup failed")
			err = brframework.WaitForBackupFailed(f.Client, ns, backupName, backupCompleteTimeout)
			f.Must(err)

			ginkgo.By("Delete backup")
			err = deleteBackup(f, backupName)
			f.Must(err)

			ginkgo.By("Check if all backup files in storage is deleted")
			cleaned, err := f.Storage.IsDataCleaned(ctx, ns, backup.Spec.S3.Prefix) // now we only use s3
			f.Must(err)
			ExpectEqual(cleaned, true, "storage should be cleaned")
		})

		ginkgo.It("kill backup pod and exceed retryTimeout", func() {
			// ginkgo.Skip("unstable case, after restart: there may be some backup files in the path already, please specify a correct backup directory")

			backupClusterName := "kill-backup-pod-exceed-timeout-test"
			backupVersion := utilimage.TiDBLatest
			enableTLS := false
			skipCA := false
			backupName := backupClusterName

			ns := f.Namespace.Name
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ginkgo.By("Create TiDB cluster")
			err := createTidbCluster(f, backupClusterName, backupVersion, enableTLS, skipCA)
			f.Must(err)

			ginkgo.By("Wait for TiDB cluster ready")
			err = waiter.WaitForClusterReady(ctx, f.Client, ns, backupClusterName, tidbReadyTimeout)
			f.Must(err)

			ginkgo.By("Create RBAC for backup")
			err = createRBAC(f)
			f.Must(err)

			ginkgo.By("Start backup and wait to running")
			backup, err := createBackupAndWaitForRunning(f, backupName, backupClusterName, func(backup *v1alpha1.Backup) {
				backup.Spec.BackoffRetryPolicy = v1alpha1.BackoffRetryPolicy{
					MinRetryDuration: "10s",
					MaxRetryTimes:    2,
					RetryTimeout:     "1m",
				}
				// panic every 30s
				backup.Spec.Env = []v1.EnvVar{v1.EnvVar{Name: e2eBackupEnv, Value: e2eExtendBackupTime + "," + e2eTestFlagPanic}}
			})
			f.Must(err)

			ginkgo.By("wait backup pod failed by simulate panic")
			err = brframework.WaitBackupPodOnPhase(f, backup, v1.PodFailed, backupCompleteTimeout)
			f.Must(err)

			ginkgo.By("wait auto restart backup pod until running again")
			err = brframework.WaitBackupPodOnPhase(f, backup, v1.PodRunning, backupCompleteTimeout)
			f.Must(err)

			ginkgo.By("make sure it's restarted by backoff retry policy")
			f.Must(brframework.GetAndCheckBackup(f.Client, ns, backupName, func(backup *v1alpha1.Backup) bool {
				num, err := getBackoffRetryNum(f, backup)
				f.Must(err)
				return num == 1
			}))
			// 30s(panic wait time) + 10s(backup pod restart time) pass

			ginkgo.By("wait backup pod failed by simulate panic")
			err = brframework.WaitBackupPodOnPhase(f, backup, v1.PodFailed, backupCompleteTimeout)
			f.Must(err)
			// 30s pass. 70s in total, exceed the RetryTimeout(1m)

			ginkgo.By("wait auto restart backup pod until backup failed")
			err = brframework.WaitForBackupFailed(f.Client, ns, backupName, backupCompleteTimeout)
			f.Must(err)

			ginkgo.By("Delete backup")
			err = deleteBackup(f, backupName)
			f.Must(err)

			ginkgo.By("Check if all backup files in storage is deleted")
			cleaned, err := f.Storage.IsDataCleaned(ctx, ns, backup.Spec.S3.Prefix) // now we only use s3
			f.Must(err)
			ExpectEqual(cleaned, true, "storage should be cleaned")
		})
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
			err := createLogBackupEnabledTidbCluster(f, masterClusterName, backupVersion, enableTLS, skipCA)
			f.Must(err)
			ginkgo.By("Wait for pitr-master TiDB cluster ready")
			err = waiter.WaitForClusterReady(ctx, f.Client, ns, masterClusterName, tidbReadyTimeout)
			f.Must(err)
			time.Sleep(10 * time.Second) // wait for tidb cluster to be more stable

			ginkgo.By("Create RBAC for backup")
			err = createRBAC(f)
			f.Must(err)

			logBackupName := "log-backup"
			ginkgo.By("Start log backup")
			logBackup, err := createBackupAndWaitForComplete(f, logBackupName, masterClusterName, func(backup *v1alpha1.Backup) {
				backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
				backup.Spec.Mode = v1alpha1.BackupModeLog
			})
			f.Must(err)
			ExpectNotEqual(logBackup.Status.CommitTs, "")

			fullBackupName := "full-backup"
			ginkgo.By("Start full backup")
			fullBackup, err := createBackupAndWaitForComplete(f, fullBackupName, masterClusterName, func(backup *v1alpha1.Backup) {
				backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
			})
			f.Must(err)
			ExpectNotEqual(fullBackup.Status.CommitTs, "")

			ginkgo.By("Forward master TiDB cluster service")
			masterHost, masterPort, cancel, err := k8s.ForwardOnePort(f.PortForwarder, ns, getTiDBServiceResourceName(masterClusterName), corev1alpha1.DefaultTiDBPortClient)
			f.Must(err)
			defer cancel()
			masterDomain := fmt.Sprintf("%s:%d", masterHost, masterPort)
			err = initDatabase(masterDomain, dbName)
			f.Must(err)

			ginkgo.By("Write data into master TiDB cluster")
			masterDSN := getDefaultDSN(masterDomain, dbName)
			err = blockwriter.New().Write(context.Background(), masterDSN)
			f.Must(err)

			ginkgo.By("Forward master PD service")
			masterPDHost, masterPDPort, cancel, err := k8s.ForwardOnePort(f.PortForwarder, ns, getPDServiceResourceName(masterClusterName), corev1alpha1.DefaultPDPortClient)
			f.Must(err)
			defer cancel()
			masterPDDomain := fmt.Sprintf("%s:%d", masterPDHost, masterPDPort)
			ginkgo.By("Wait log backup reach current ts")
			currentTS := strconv.FormatUint(v1alpha1.GoTimeToTS(time.Now()), 10)
			err = brframework.WaitForLogBackupReachTS(logBackupName, masterPDDomain, currentTS, logbackupCatchUpTimeout)
			f.Must(err)

			ginkgo.By("wait log backup progress reach current ts")
			err = brframework.WaitForLogBackupProgressReachTS(f.Client, ns, logBackupName, currentTS, logbackupCatchUpTimeout)
			f.Must(err)

			ginkgo.By("Create log-backup.enable TiDB cluster for pitr-backup")
			backupClusterName := "pitr-backup"
			err = createLogBackupEnabledTidbCluster(f, backupClusterName, backupVersion, enableTLS, skipCA)
			f.Must(err)
			ginkgo.By("Wait for pitr-backup TiDB cluster ready")
			err = waiter.WaitForClusterReady(ctx, f.Client, ns, backupClusterName, tidbReadyTimeout)
			f.Must(err)

			time.Sleep(20 * time.Second)
			ginkgo.By("Create pitr restore")
			restoreName := "pitr-restore"
			err = createRestoreAndWaitForComplete(f, restoreName, backupClusterName, logBackupName, func(restore *v1alpha1.Restore) {
				restore.Spec.Mode = v1alpha1.RestoreModePiTR
				restore.Spec.PitrFullBackupStorageProvider.S3 = fullBackup.Spec.S3
				restore.Spec.PitrRestoredTs = currentTS
			})
			f.Must(err)

			ginkgo.By("wait pitr restore progress done")
			err = brframework.WaitForRestoreProgressDone(f.Client, ns, restoreName, restoreCompleteTimeout)
			f.Must(err)

			ginkgo.By("Forward restore TiDB cluster service")
			backupHost, backupPort, cancel, err := k8s.ForwardOnePort(f.PortForwarder, ns, getTiDBServiceResourceName(backupClusterName), corev1alpha1.DefaultTiDBPortClient)
			f.Must(err)
			defer cancel()
			backupDomain := fmt.Sprintf("%s:%d", backupHost, backupPort)

			ginkgo.By("Validate pitr restore result")
			backupDSN := getDefaultDSN(backupDomain, dbName)
			err = checkDataIsSame(masterDSN, backupDSN)
			f.Must(err)

			ginkgo.By("Delete pitr log backup")
			err = deleteBackup(f, logBackupName)
			f.Must(err)

			ginkgo.By("Check if all pitr log backup files in storage is deleted")
			cleaned, err := f.Storage.IsDataCleaned(ctx, ns, logBackup.Spec.S3.Prefix) // now we only use s3
			f.Must(err)
			ExpectEqual(cleaned, true, "storage should be cleaned")

			ginkgo.By("Delete pitr full backup")
			err = deleteBackup(f, fullBackupName)
			f.Must(err)

			ginkgo.By("Check if all pitr full backup files in storage is deleted")
			cleaned, err = f.Storage.IsDataCleaned(ctx, ns, fullBackup.Spec.S3.Prefix) // now we only use s3
			f.Must(err)
			ExpectEqual(cleaned, true, "storage should be cleaned")
		})
	})

	/*
		ginkgo.Context("Compact backup Test", func() {
			ginkgo.It("test normal function", func() {
				backupVersion := utilimage.TiDBLatest
				enableTLS := false
				skipCA := false
				ns := f.Namespace.Name
				dbName := "e2etest"
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				ginkgo.By("Create log-backup.enable TiDB cluster for pitr-master")
				masterClusterName := "pitr-master"
				err := createLogBackupEnabledTidbCluster(f, masterClusterName, backupVersion, enableTLS, skipCA)
				f.Must(err)
				ginkgo.By("Wait for pitr-master TiDB cluster ready")
				err = waiter.WaitForClusterReady(ctx, f.Client, ns, masterClusterName, tidbReadyTimeout)
				f.Must(err)

				ginkgo.By("Create RBAC for backup")
				err = createRBAC(f)
				f.Must(err)

				logBackupName := "log-backup"
				typ := strings.ToLower(typeBR)
				ginkgo.By("Start log backup")
				logBackup, err := createBackupAndWaitForComplete(f, logBackupName, masterClusterName, typ, func(backup *v1alpha1.Backup) {
					backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
					backup.Spec.Mode = v1alpha1.BackupModeLog
				})
				f.Must(err)
				ExpectNotEqual(logBackup.Status.CommitTs, "")

				fullBackupName := "full-backup"
				ginkgo.By("Start full backup")
				fullBackup, err := createBackupAndWaitForComplete(f, fullBackupName, masterClusterName, typ, func(backup *v1alpha1.Backup) {
					backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete
				})
				f.Must(err)
				ExpectNotEqual(fullBackup.Status.CommitTs, "")

				ginkgo.By("Forward master TiDB cluster service")
				masterHost, err := portforward.ForwardOnePort(ctx, f.PortForwarder, ns, getTiDBServiceResourceName(masterClusterName), int(corev1alpha1.DefaultTiDBPortClient))
				f.Must(err)
				err = initDatabase(masterHost, dbName)
				f.Must(err)

				ginkgo.By("Write data into master TiDB cluster")
				masterDSN := getDefaultDSN(masterHost, dbName)
				err = blockwriter.New().Write(context.Background(), masterDSN)
				f.Must(err)

				ginkgo.By("Forward master PD service")
				masterPDHost, err := portforward.ForwardOnePort(ctx, f.PortForwarder, ns, getPDServiceResourceName(masterClusterName), int(corev1alpha1.DefaultPDPortClient))
				f.Must(err)
				ginkgo.By("Wait log backup reach current ts")
				currentTS := strconv.FormatUint(v1alpha1.GoTimeToTS(time.Now()), 10)
				err = brframework.WaitForLogBackupReachTS(logBackupName, masterPDHost, currentTS, logbackupCatchUpTimeout)
				f.Must(err)

				ginkgo.By("wait log backup progress reach current ts")
				err = brframework.WaitForLogBackupProgressReachTS(f.Client, ns, logBackupName, currentTS, logbackupCatchUpTimeout)
				f.Must(err)

				compactName := "compact-backup"
				ginkgo.By("Start a compact backup")
				_, err = createCompactBackupAndWaitForComplete(f, compactName, masterClusterName, func(compact *v1alpha1.CompactBackup) {
					compact.Spec.StartTs = fullBackup.Status.CommitTs
					compact.Spec.EndTs = currentTS
					compact.Spec.S3 = logBackup.Spec.S3
					compact.Spec.BR = logBackup.Spec.BR
					compact.Spec.MaxRetryTimes = 2
				})
				f.Must(err)
			})
			ginkgo.It("test backoff when create job failed", func() {
				_, cancel := context.WithCancel(context.Background())
				defer cancel()

				ginkgo.By("Create RBAC for backup")
				err := createRBAC(f)
				f.Must(err)

				compactName := "compact-backup"
				ginkgo.By("Start a compact backup")
				_, err = createCompactBackupAndWaitForComplete(f, compactName, "No_Such_Cluster", func(compact *v1alpha1.CompactBackup) {
					compact.Spec.StartTs = "1"
					compact.Spec.EndTs = "1"
					compact.Spec.S3 = nil
					compact.Spec.MaxRetryTimes = 2
				})
				framework.ExpectError(err, "create job failed, reached max retry times")
			})
		})
	*/
})

func getTiDBServiceResourceName(clusterName string) string {
	dbGroupName := getGroupName(clusterName, "dbg")
	// TODO: use common util to get tidb service name
	return "svc/" + dbGroupName + "-tidb"
}

// getGroupName returns the group name
// groupSuffix is the suffix of the group name, for example, "dbg", "kvg", "pdg" for tidb group
func getGroupName(clusterName string, groupSuffix string) string {
	return clusterName + "-" + groupSuffix
}

func getPDServiceResourceName(tcName string) string {
	// TODO: use common util to get tidb service name
	return "svc/" + getGroupName(tcName, "pdg") + "-pd"
}

func createTidbCluster(f *brframework.Framework, name string, version string, enableTLS bool, skipCA bool) error {
	ctx := context.TODO()

	clusterPatches := []data.ClusterPatch{
		data.WithClusterName(name),
		data.WithBootstrapSQL(),
	}
	if enableTLS {
		gomega.Expect(cluster.InstallTiDBIssuer(ctx, f.YamlApplier, f.Namespace.Name, name)).To(gomega.Succeed())
		gomega.Expect(cluster.InstallTiDBCertificates(ctx, f.YamlApplier, f.Namespace.Name, name, "dbg")).To(gomega.Succeed())
		gomega.Expect(cluster.InstallTiDBComponentsCertificates(ctx,
			f.YamlApplier,
			f.Namespace.Name,
			name,
			getGroupName(name, "pdg"),
			getGroupName(name, "kvg"),
			getGroupName(name, "dbg"),
			getGroupName(name, "flashg"),
			getGroupName(name, "cdcg"),
		)).To(gomega.Succeed())

		clusterPatches = append(clusterPatches, data.WithClusterTLS())
	}
	cluster := f.MustCreateCluster(ctx, clusterPatches...)
	_ = f.MustCreatePD(ctx,
		data.WithGroupName[*runtime.PDGroup](getGroupName(name, "pdg")),
		data.WithGroupVersion[*runtime.PDGroup](version),
		data.WithGroupCluster[*runtime.PDGroup](name),
	)
	_ = f.MustCreateTiKV(ctx,
		data.WithGroupName[*runtime.TiKVGroup](getGroupName(name, "kvg")),
		data.WithGroupVersion[*runtime.TiKVGroup](version),
		data.WithGroupCluster[*runtime.TiKVGroup](name),
	)
	_ = f.MustCreateTiDB(ctx,
		data.WithGroupName[*runtime.TiDBGroup](getGroupName(name, "dbg")),
		data.WithGroupVersion[*runtime.TiDBGroup](version),
		data.WithGroupCluster[*runtime.TiDBGroup](name),
	)

	ginkgo.DeferCleanup(func(ctx context.Context) {
		ginkgo.By(fmt.Sprintf("Delete the cluster: %s", cluster.Name))
		f.Must(f.Client.Delete(ctx, cluster))
	})
	return nil
}

// createLogBackupEnabledTidbCluster create tidb cluster and set "log-backup.enable = true" in tikv to enable log backup.
func createLogBackupEnabledTidbCluster(f *brframework.Framework, name string, version string, enableTLS bool, skipCA bool) error {
	ctx := context.TODO()

	clusterPatches := []data.ClusterPatch{
		data.WithClusterName(name),
		data.WithBootstrapSQL(),
	}
	if enableTLS {
		gomega.Expect(cluster.InstallTiDBIssuer(ctx, f.YamlApplier, f.Namespace.Name, name)).To(gomega.Succeed())
		gomega.Expect(cluster.InstallTiDBCertificates(ctx, f.YamlApplier, f.Namespace.Name, name, "dbg")).To(gomega.Succeed())
		gomega.Expect(cluster.InstallTiDBComponentsCertificates(ctx,
			f.YamlApplier,
			f.Namespace.Name,
			name,
			getGroupName(name, "pdg"),
			getGroupName(name, "kvg"),
			getGroupName(name, "dbg"),
			getGroupName(name, "flashg"),
			getGroupName(name, "cdcg"),
		)).To(gomega.Succeed())

		clusterPatches = append(clusterPatches, data.WithClusterTLS())
	}
	cluster := f.MustCreateCluster(ctx, clusterPatches...)
	_ = f.MustCreatePD(ctx,
		data.WithGroupName[*runtime.PDGroup](getGroupName(name, "pdg")),
		data.WithGroupVersion[*runtime.PDGroup](version),
		data.WithGroupCluster[*runtime.PDGroup](name),
	)

	// create tikv group
	kvg := data.NewTiKVGroup(f.Namespace.Name,
		data.WithGroupName[*runtime.TiKVGroup](getGroupName(name, "kvg")),
		data.WithGroupVersion[*runtime.TiKVGroup](version),
		data.WithGroupCluster[*runtime.TiKVGroup](name),
	)
	kvg.Spec.Template.Spec.Config = "log-backup.enable = true"
	ginkgo.By("Creating a tikv group")
	f.Must(f.Client.Create(ctx, kvg))

	_ = f.MustCreateTiDB(ctx,
		data.WithGroupName[*runtime.TiDBGroup](getGroupName(name, "dbg")),
		data.WithGroupVersion[*runtime.TiDBGroup](version),
		data.WithGroupCluster[*runtime.TiDBGroup](name),
	)

	ginkgo.DeferCleanup(func(ctx context.Context) {
		ginkgo.By(fmt.Sprintf("Delete the cluster: %s", cluster.Name))
		f.Must(f.Client.Delete(ctx, cluster))
	})
	return nil
}

func createRBAC(f *brframework.Framework) error {
	ctx := context.TODO()
	ns := f.Namespace.Name
	sa := brframework.GetServiceAccount(ns)
	if err := f.Client.Create(ctx, sa); err != nil {
		return err
	}

	role := brframework.GetRole(ns)
	if err := f.Client.Create(ctx, role); err != nil {
		return err
	}
	rb := brframework.GetRoleBinding(ns)
	if err := f.Client.Create(ctx, rb); err != nil {
		return err
	}
	return nil
}

func createBackupAndWaitForComplete(f *brframework.Framework, name, tcName string, configure func(*v1alpha1.Backup)) (*v1alpha1.Backup, error) {
	ctx := context.TODO()
	ns := f.Namespace.Name
	// secret to visit tidb cluster
	s := brframework.GetSecret(ns, name, "")
	// Check if the secret already exists
	if err := f.Client.Get(ctx, client.ObjectKeyFromObject(s), &corev1.Secret{}); err != nil {
		if apierrors.IsNotFound(err) {
			if err := f.Client.Create(ctx, s); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	backupFolder := time.Now().Format(time.RFC3339)
	cfg := f.Storage.Config(ns, backupFolder)
	backup := brframework.GetBackup(ns, name, tcName, cfg)

	if configure != nil {
		configure(backup)
	}

	if err := f.Client.Create(ctx, backup); err != nil {
		return nil, err
	}

	if err := brframework.WaitForBackupComplete(f.Client, ns, name, backupCompleteTimeout); err != nil {
		return backup, err
	}
	err := f.Client.Get(ctx, client.ObjectKeyFromObject(backup), backup)
	return backup, err
}

func createBackupAndWaitForRunning(f *brframework.Framework, name, tcName string, configure func(*v1alpha1.Backup)) (*v1alpha1.Backup, error) {
	ctx := context.TODO()
	ns := f.Namespace.Name
	// secret to visit tidb cluster
	s := brframework.GetSecret(ns, name, "")
	if err := f.Client.Create(ctx, s); err != nil {
		return nil, err
	}

	backupFolder := time.Now().Format(time.RFC3339)
	cfg := f.Storage.Config(ns, backupFolder)
	backup := brframework.GetBackup(ns, name, tcName, cfg)

	if configure != nil {
		configure(backup)
	}

	if err := f.Client.Create(ctx, backup); err != nil {
		return nil, err
	}

	if err := brframework.WaitForBackupOnRunning(f.Client, ns, name, backupCompleteTimeout); err != nil {
		return backup, err
	}
	err := f.Client.Get(ctx, client.ObjectKeyFromObject(backup), backup)
	return backup, err
}

func createBackupAndWaitForSchedule(f *brframework.Framework, name, tcName, typ string, configure func(*v1alpha1.Backup)) (*v1alpha1.Backup, error) {
	ctx := context.TODO()
	ns := f.Namespace.Name
	// secret to visit tidb cluster
	s := brframework.GetSecret(ns, name, "")
	if err := f.Client.Create(ctx, s); err != nil {
		return nil, err
	}

	backupFolder := time.Now().Format(time.RFC3339)
	cfg := f.Storage.Config(ns, backupFolder)
	backup := brframework.GetBackup(ns, name, tcName, cfg)

	if configure != nil {
		configure(backup)
	}

	if err := f.Client.Create(ctx, backup); err != nil {
		return nil, err
	}

	if err := brframework.WaitForBackupOnScheduled(f.Client, ns, name, backupCompleteTimeout); err != nil {
		return backup, err
	}
	err := f.Client.Get(ctx, client.ObjectKeyFromObject(backup), backup)
	return backup, err
}

func getBackoffRetryNum(f *brframework.Framework, backup *v1alpha1.Backup) (int, error) {
	ctx := context.TODO()
	ns := f.Namespace.Name
	name := backup.Name

	newBackup := v1alpha1.Backup{}
	err := f.Client.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, &newBackup)
	if err != nil {
		return 0, err
	}

	if len(newBackup.Status.BackoffRetryStatus) == 0 {
		return 0, nil
	}
	return newBackup.Status.BackoffRetryStatus[len(newBackup.Status.BackoffRetryStatus)-1].RetryNum, nil
}

// continueLogBackupAndWaitForComplete update backup cr to continue run log backup subcommand
func continueLogBackupAndWaitForComplete(f *brframework.Framework, backup *v1alpha1.Backup, configure func(*v1alpha1.Backup)) (*v1alpha1.Backup, error) {
	ctx := context.TODO()
	ns := f.Namespace.Name
	name := backup.Name

	if configure != nil {
		configure(backup)
	}

	if err := f.Client.Update(ctx, backup); err != nil {
		return nil, err
	}

	if err := brframework.WaitForBackupComplete(f.Client, ns, name, backupCompleteTimeout); err != nil {
		return backup, err
	}

	err := f.Client.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, backup)
	return backup, err
}

// updateBackup update backup cr
func updateBackup(f *brframework.Framework, backupName string, configure func(*v1alpha1.Backup)) (*v1alpha1.Backup, error) {
	ns := f.Namespace.Name
	ctx := context.TODO()
	backup := &v1alpha1.Backup{}
	err := f.Client.Get(ctx, client.ObjectKey{Namespace: ns, Name: backupName}, backup)
	if err != nil {
		return nil, err
	}

	if configure != nil {
		configure(backup)
	}

	if err := f.Client.Update(ctx, backup); err != nil {
		return nil, err
	}

	return backup, nil
}

func deleteBackup(f *brframework.Framework, name string) error {
	ctx := context.TODO()
	ns := f.Namespace.Name
	backup := &v1alpha1.Backup{}
	backup.Namespace = ns
	backup.Name = name
	if err := f.Client.Delete(ctx, backup); err != nil {
		return err
	}

	if err := brframework.WaitForBackupDeleted(f.Client, ns, name, 15*time.Minute); err != nil {
		return err
	}
	return nil
}

// nolint
// NOTE: it is not used
func cleanBackup(f *brframework.Framework) error {
	ns := f.Namespace.Name
	ctx := context.TODO()
	bl := &v1alpha1.BackupList{}
	err := f.Client.List(ctx, bl, client.InNamespace(ns))
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

func createRestoreAndWaitForComplete(f *brframework.Framework, name, tcName, backupName string, configure func(*v1alpha1.Restore)) error {
	ctx := context.TODO()
	ns := f.Namespace.Name
	// secret to visit tidb cluster
	s := brframework.GetSecret(ns, name, "")
	if err := f.Client.Create(ctx, s); err != nil {
		return err
	}

	backup := v1alpha1.Backup{}
	err := f.Client.Get(ctx, client.ObjectKey{Namespace: ns, Name: backupName}, &backup)
	if err != nil {
		return err
	}

	cfg := backup.Spec.S3

	restore := brframework.GetRestore(ns, name, tcName, cfg)

	if configure != nil {
		configure(restore)
	}

	if err := f.Client.Create(ctx, restore); err != nil {
		return err
	}

	if err := brframework.WaitForRestoreComplete(f.Client, ns, name, restoreCompleteTimeout); err != nil {
		return err
	}

	return nil
}

func getDefaultDSN(host, dbName string) string {
	user := "root"
	password := "pingcap"
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

func WaitForPDReady(f *brframework.Framework, ns, name string, timeout time.Duration) error {
	time.Sleep(10 * time.Second)
	ctx := context.TODO()
	// get scheme
	cluster := &corev1alpha1.Cluster{}
	err := f.Client.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, cluster)
	if err != nil {
		return err
	}
	scheme := "https://"
	if !coreutil.IsTLSClusterEnabled(cluster) {
		scheme = "http://"
	}
	// get host
	masterPDHost, masterPDPort, cancel, err := k8s.ForwardOnePort(f.PortForwarder, ns, getPDServiceResourceName(name), corev1alpha1.DefaultPDPortClient)
	if err != nil {
		return err
	}
	defer cancel()
	masterPDDomain := fmt.Sprintf("%s:%d", masterPDHost, masterPDPort)
	pdUrl := fmt.Sprintf("%s%s", scheme, masterPDDomain)
	pdcli := pdapi.NewPDClient(pdUrl, 30*time.Second, nil)
	return wait.PollUntilContextTimeout(ctx, 2*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		health, err := pdcli.GetHealth(ctx)
		if err != nil {
			return false, err
		}
		for _, h := range health.Healths {
			if !h.Health {
				return false, nil
			}
		}
		return true, nil
	})
}

// func createCompactBackupAndWaitForComplete(f *e2eframework.Framework, name, tcName string, configure func(*v1alpha1.CompactBackup)) (*v1alpha1.CompactBackup, error) {
// 	ctx := context.TODO()
// 	ns := f.Namespace.Name
// 	// secret to visit tidb cluster
// 	s := brframework.GetSecret(ns, name, "")
// 	// Check if the secret already exists
// 	if err := f.Client.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, &corev1.Secret{}); err != nil {
// 		if apierrors.IsNotFound(err) {
// 			if err := f.Client.Create(ctx, s); err != nil {
// 				return nil, err
// 			}
// 		} else {
// 			return nil, err
// 		}
// 	}

// 	backupFolder := time.Now().Format(time.RFC3339)
// 	cfg := f.Storage.Config(ns, backupFolder)
// 	compact := brframework.GetCompactBackup(ns, name, tcName, cfg)

// 	if configure != nil {
// 		configure(compact)
// 	}

// 	if err := f.Client.Create(ctx, compact); err != nil {
// 		return nil, err
// 	}

// 	if err := brframework.WaitForCompactComplete(f, ns, name, backupCompleteTimeout); err != nil {
// 		return compact, err
// 	}
// 	compact = &v1alpha1.CompactBackup{}
// 	err := f.Client.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, &compact)
// 	return compact, err
// }

// ExpectEqual expects the specified two are the same, otherwise an exception raises
func ExpectEqual(actual interface{}, extra interface{}, explain ...interface{}) {
	gomega.ExpectWithOffset(1, actual).To(gomega.Equal(extra), explain...)
}

// ExpectNotEqual expects the specified two are not the same, otherwise an exception raises
func ExpectNotEqual(actual interface{}, extra interface{}, explain ...interface{}) {
	gomega.ExpectWithOffset(1, actual).NotTo(gomega.Equal(extra), explain...)
}

// ExpectError expects an error happens, otherwise an exception raises
func ExpectError(err error, explain ...interface{}) {
	gomega.ExpectWithOffset(1, err).To(gomega.HaveOccurred(), explain...)
}

// ExpectNoError checks if "err" is set, and if so, fails assertion while logging the error.
func ExpectNoError(err error, explain ...interface{}) {
	ExpectNoErrorWithOffset(1, err, explain...)
}

// ExpectNoErrorWithOffset checks if "err" is set, and if so, fails assertion while logging the error at "offset" levels above its caller
// (for example, for call chain f -> g -> ExpectNoErrorWithOffset(1, ...) error would be logged for "f").
func ExpectNoErrorWithOffset(offset int, err error, explain ...interface{}) {
	gomega.ExpectWithOffset(1+offset, err).NotTo(gomega.HaveOccurred(), explain...)
}
