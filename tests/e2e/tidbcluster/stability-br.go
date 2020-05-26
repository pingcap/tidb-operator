// Copyright 2020 PingCAP, Inc.
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

package tidbcluster

import (
	"context"
	nerrors "errors"
	"fmt"
	_ "net/http/pprof"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/onsi/ginkgo"
	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	asclientset "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/scheme"
	"github.com/pingcap/tidb-operator/tests"
	e2econfig "github.com/pingcap/tidb-operator/tests/e2e/config"
	e2eframework "github.com/pingcap/tidb-operator/tests/e2e/framework"
	utilimage "github.com/pingcap/tidb-operator/tests/e2e/util/image"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	teststorage "github.com/pingcap/tidb-operator/tests/e2e/util/storage"
	utiltidb "github.com/pingcap/tidb-operator/tests/e2e/util/tidb"
	"github.com/pingcap/tidb-operator/tests/pkg/fixture"
	corev1 "k8s.io/api/core/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	restclient "k8s.io/client-go/rest"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	"k8s.io/kubernetes/test/e2e/framework"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = ginkgo.Describe("[tidb-operator][Stability]", func() {
	f := e2eframework.NewDefaultFramework("br")

	var ns string
	var c clientset.Interface
	var cli versioned.Interface
	var asCli asclientset.Interface
	var aggrCli aggregatorclient.Interface
	var apiExtCli apiextensionsclientset.Interface
	var oa tests.OperatorActions
	var cfg *tests.Config
	var config *restclient.Config
	var ocfg *tests.OperatorConfig
	var genericCli client.Client
	var fwCancel context.CancelFunc
	var fw portforward.PortForward
	/**
	 * StatefulSet or AdvancedStatefulSet interface.
	 */
	var stsGetter func(namespace string) typedappsv1.StatefulSetInterface

	ginkgo.BeforeEach(func() {
		ns = f.Namespace.Name
		c = f.ClientSet
		var err error
		config, err = framework.LoadConfig()
		framework.ExpectNoError(err, "failed to load config")
		cli, err = versioned.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset")
		asCli, err = asclientset.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset")
		genericCli, err = client.New(config, client.Options{Scheme: scheme.Scheme})
		framework.ExpectNoError(err, "failed to create clientset")
		aggrCli, err = aggregatorclient.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset")
		apiExtCli, err = apiextensionsclientset.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset")
		clientRawConfig, err := e2econfig.LoadClientRawConfig()
		framework.ExpectNoError(err, "failed to load raw config")
		ctx, cancel := context.WithCancel(context.Background())
		fw, err = portforward.NewPortForwarder(ctx, e2econfig.NewSimpleRESTClientGetter(clientRawConfig))
		framework.ExpectNoError(err, "failed to create port forwarder")
		fwCancel = cancel
		cfg = e2econfig.TestConfig
		ocfg = e2econfig.NewDefaultOperatorConfig(cfg)
		if ocfg.Enabled(features.AdvancedStatefulSet) {
			stsGetter = helper.NewHijackClient(c, asCli).AppsV1().StatefulSets
		} else {
			stsGetter = c.AppsV1().StatefulSets
		}
		oa = tests.NewOperatorActions(cli, c, asCli, aggrCli, apiExtCli, tests.DefaultPollInterval, ocfg, e2econfig.TestConfig, nil, fw, f)
	})

	ginkgo.AfterEach(func() {
		if fwCancel != nil {
			fwCancel()
		}
	})

	ginkgo.It("Backup and restore with BR", func() {
		provider := framework.TestContext.Provider
		if provider != "aws" && provider != "kind" {
			framework.Skipf("provider is not aws or kind, skipping")
		}

		testBR(provider, ns, fw, c, genericCli, oa, cli, false)
	})

	ginkgo.Context("[Feature: TLS]", func() {
		ginkgo.It("Backup and restore with BR when TLS enabled", func() {
			provider := framework.TestContext.Provider
			if provider != "aws" && provider != "kind" {
				framework.Skipf("provider is not aws or kind, skipping")
			}

			ginkgo.By("Installing cert-manager")
			err := installCertManager(f.ClientSet)
			framework.ExpectNoError(err, "failed to install cert-manager")
			
			testBR(provider, ns, fw, c, genericCli, oa, cli, true)
			ginkgo.By("Deleting cert-manager")
			err = deleteCertManager(f.ClientSet)
			framework.ExpectNoError(err, "failed to delete cert-manager")
		})
	})

})

func testBR(provider, ns string, fw portforward.PortForward, c clientset.Interface, genericCli client.Client, oa tests.OperatorActions, cli versioned.Interface, tlsEnabled bool) {
	backupFolder := time.Now().Format(time.RFC3339)
	var storage teststorage.TestStorage
	switch provider {
	case "kind":
		s3config := &v1alpha1.S3StorageProvider{
			Provider:   v1alpha1.S3StorageProviderTypeCeph,
			Prefix:     backupFolder,
			SecretName: fixture.S3Secret,
			Bucket:     "local",
			Endpoint:   "http://minio-service:9000",
		}
		key := "12345678"
		minio, cancel, err := teststorage.NewMinioStorage(fw, ns, key, key, c, s3config)
		framework.ExpectNoError(err)
		storage = minio
		defer cancel()
	case "aws":
		cred := credentials.NewSharedCredentials("", "default")
		s3config := &v1alpha1.S3StorageProvider{
			Provider:   v1alpha1.S3StorageProviderTypeAWS,
			Region:     fixture.AWSRegion,
			Bucket:     fixture.Bucket,
			Prefix:     backupFolder,
			SecretName: fixture.S3Secret,
		}
		s3Storage, err := teststorage.NewS3Storage(cred, s3config)
		framework.ExpectNoError(err)
		storage = s3Storage
	default:
		framework.Failf("unknown provider: %s", provider)
	}
	if storage == nil {
		framework.Failf("storage generate failed")
	}

	tcNameFrom := "backup"
	tcNameTo := "restore"
	serviceAccountName := "tidb-backup-manager"

	if tlsEnabled {
		ginkgo.By("Installing tidb issuer")
		err := installTiDBIssuer(ns, tcNameFrom)
		framework.ExpectNoError(err, "failed to generate tidb issuer template")
		err = installTiDBIssuer(ns, tcNameTo)
		framework.ExpectNoError(err, "failed to generate tidb issuer template")

		ginkgo.By("Installing tidb server and client certificate")
		err = installTiDBCertificates(ns, tcNameFrom)
		framework.ExpectNoError(err, "failed to install tidb server and client certificate")
		err = installTiDBCertificates(ns, tcNameTo)
		framework.ExpectNoError(err, "failed to install tidb server and client certificate")

		ginkgo.By("Installing backup and restore separate client certificates")
		err = installBackupCertificates(ns, tcNameFrom)
		framework.ExpectNoError(err, "failed to install backup client certificate")
		err = installRestoreCertificates(ns, tcNameTo)
		framework.ExpectNoError(err, "failed to install restore client certificate")
	}

	// create backup cluster
	tcFrom := fixture.GetTidbCluster(ns, tcNameFrom, utilimage.TiDBV4UpgradeVersion)
	tcFrom.Spec.PD.Replicas = 1
	tcFrom.Spec.TiKV.Replicas = 1
	tcFrom.Spec.TiDB.Replicas = 1
	if tlsEnabled {
		tcFrom.Spec.TiDB.TLSClient = &v1alpha1.TiDBTLSClient{Enabled: true}
	}
	err := genericCli.Create(context.TODO(), tcFrom)
	framework.ExpectNoError(err)
	err = oa.WaitForTidbClusterReady(tcFrom, 30*time.Minute, 15*time.Second)
	framework.ExpectNoError(err)
	clusterFrom := newTidbClusterConfig(e2econfig.TestConfig, ns, tcNameFrom, "", utilimage.TiDBV4Version)

	// create restore cluster
	tcTo := fixture.GetTidbCluster(ns, tcNameTo, utilimage.TiDBV4Version)
	tcTo.Spec.PD.Replicas = 1
	tcTo.Spec.TiKV.Replicas = 1
	tcTo.Spec.TiDB.Replicas = 1
	if tlsEnabled {
		tcTo.Spec.TiDB.TLSClient = &v1alpha1.TiDBTLSClient{Enabled: true}
	}
	err = genericCli.Create(context.TODO(), tcTo)
	framework.ExpectNoError(err)
	err = oa.WaitForTidbClusterReady(tcTo, 30*time.Minute, 15*time.Second)
	framework.ExpectNoError(err)
	clusterTo := newTidbClusterConfig(e2econfig.TestConfig, ns, tcNameTo, "", utilimage.TiDBV4Version)

	// import some data to sql with blockwriter
	ginkgo.By(fmt.Sprintf("Begin inserting data into cluster %q", clusterFrom.ClusterName))
	oa.BeginInsertDataToOrDie(&clusterFrom)
	err = wait.PollImmediate(time.Second*5, time.Minute*5, utiltidb.TiDBIsInserted(fw, tcFrom.GetNamespace(), tcFrom.GetName(), "root", "", "sbtest", "block_writer"))
	framework.ExpectNoError(err)
	ginkgo.By(fmt.Sprintf("Stop inserting data into cluster %q", clusterFrom.ClusterName))
	oa.StopInsertDataTo(&clusterFrom)

	// prepare for create backup/restore CRD
	backupRole := fixture.GetBackupRole(tcFrom, serviceAccountName)
	_, err = c.RbacV1beta1().Roles(ns).Create(backupRole)
	framework.ExpectNoError(err)
	backupServiceAccount := fixture.GetBackupServiceAccount(tcFrom, serviceAccountName)
	_, err = c.CoreV1().ServiceAccounts(ns).Create(backupServiceAccount)
	framework.ExpectNoError(err)
	backupRoleBinding := fixture.GetBackupRoleBinding(tcFrom, serviceAccountName)
	_, err = c.RbacV1beta1().RoleBindings(ns).Create(backupRoleBinding)
	framework.ExpectNoError(err)
	backupSecret := fixture.GetBackupSecret(tcFrom, "")
	_, err = c.CoreV1().Secrets(ns).Create(backupSecret)
	framework.ExpectNoError(err)
	restoreSecret := fixture.GetBackupSecret(tcTo, "")
	_, err = c.CoreV1().Secrets(ns).Create(restoreSecret)
	framework.ExpectNoError(err)
	storageSecret := storage.ProvideCredential(ns)
	_, err = c.CoreV1().Secrets(ns).Create(storageSecret)
	framework.ExpectNoError(err)

	ginkgo.By(fmt.Sprintf("Begion to backup data cluster %q", clusterFrom.ClusterName))
	// create backup CRD to process backup
	backup := storage.ProvideBackup(tcFrom, backupSecret)
	if tlsEnabled {
		backupSecretName := fmt.Sprintf("%s-backup-tls", tcNameFrom)
		backup.Spec.From.TLSClientSecretName = &backupSecretName
	}
	_, err = cli.PingcapV1alpha1().Backups(ns).Create(backup)
	framework.ExpectNoError(err)

	// check backup is successed
	err = wait.PollImmediate(5*time.Second, 10*time.Minute, func() (bool, error) {
		tmpBackup, err := cli.PingcapV1alpha1().Backups(ns).Get(backup.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		// Check the status in conditions one by one,
		// if the status other than complete or failed is running
		for _, condition := range tmpBackup.Status.Conditions {
			if condition.Type == v1alpha1.BackupComplete && condition.Status == corev1.ConditionTrue {
				return true, nil
			} else if condition.Type == v1alpha1.BackupFailed && condition.Status == corev1.ConditionTrue {
				return false, errors.NewInternalError(nerrors.New(condition.Reason))
			}
		}
		return false, nil
	})
	framework.ExpectNoError(err)

	ginkgo.By(fmt.Sprintf("Begion to Restore data cluster %q", clusterTo.ClusterName))
	// create restore CRD to process restore
	restore := storage.ProvideRestore(tcTo, restoreSecret)
	if tlsEnabled {
		restoreSecretName := fmt.Sprintf("%s-restore-tls", tcNameTo)
		restore.Spec.To.TLSClientSecretName = &restoreSecretName
	}
	_, err = cli.PingcapV1alpha1().Restores(ns).Create(restore)
	framework.ExpectNoError(err)

	// check restore is successed
	err = wait.PollImmediate(5*time.Second, 10*time.Minute, func() (bool, error) {
		tmpRestore, err := cli.PingcapV1alpha1().Restores(ns).Get(restore.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		// Check the status in conditions one by one,
		// if the status other than complete or failed is running
		for _, condition := range tmpRestore.Status.Conditions {
			if condition.Type == v1alpha1.RestoreComplete && condition.Status == corev1.ConditionTrue {
				return true, nil
			} else if condition.Type == v1alpha1.RestoreFailed && condition.Status == corev1.ConditionTrue {
				return false, errors.NewInternalError(nerrors.New(condition.Reason))
			}
		}
		return false, nil
	})
	framework.ExpectNoError(err)

	ginkgo.By(fmt.Sprintf("Check the correctness of cluster %q and %q", clusterFrom.ClusterName, clusterTo.ClusterName))
	isSame, err := oa.DataIsTheSameAs(&clusterFrom, &clusterTo)
	framework.ExpectNoError(err)
	if !isSame {
		framework.ExpectNoError(nerrors.New("backup database and restore database is not the same"))
	}

	// delete backup data in S3
	err = cli.PingcapV1alpha1().Backups(ns).Delete(backup.Name, &metav1.DeleteOptions{})
	framework.ExpectNoError(err)

	err = storage.CheckDataCleaned()
	framework.ExpectNoError(err)

	err = wait.PollImmediate(5*time.Second, 5*time.Minute, func() (bool, error) {
		_, err := cli.PingcapV1alpha1().Backups(ns).Get(backup.Name, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			return true, nil
		}
		return false, nil
	})
	framework.ExpectNoError(err, "clean backup failed")
	framework.Logf("clean backup success")
}
