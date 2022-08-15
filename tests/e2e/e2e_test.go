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

package e2e

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	e2econfig "github.com/pingcap/tidb-operator/tests/e2e/config"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/e2e/framework/config"
	"k8s.io/kubernetes/test/e2e/framework/log"
	"k8s.io/kubernetes/test/e2e/framework/testfiles"

	// test sources
	_ "github.com/pingcap/tidb-operator/tests/e2e/br"
	_ "github.com/pingcap/tidb-operator/tests/e2e/dmcluster"
	_ "github.com/pingcap/tidb-operator/tests/e2e/tidbcluster"
	_ "github.com/pingcap/tidb-operator/tests/e2e/tidbngmonitoring"
	_ "github.com/pingcap/tidb-operator/tests/e2e/tikv"
)

// handleFlags sets up all flags and parses the command line.
func handleFlags() {
	config.CopyFlags(config.Flags, flag.CommandLine)
	framework.RegisterCommonFlags(flag.CommandLine)
	framework.RegisterClusterFlags(flag.CommandLine)
	e2econfig.RegisterTiDBOperatorFlags(flag.CommandLine)
	flag.Parse()
}

func init() {
	framework.RegisterProvider("kind", func() (framework.ProviderInterface, error) {
		return framework.NullProvider{}, nil
	})
	framework.RegisterProvider("openshift", func() (framework.ProviderInterface, error) {
		return framework.NullProvider{}, nil
	})
}

// See https://github.com/kubernetes/kubernetes/pull/90591
func createTestingNS(baseName string, c clientset.Interface, labels map[string]string) (*v1.Namespace, error) {
	if labels == nil {
		labels = map[string]string{}
	}
	labels["e2e-run"] = string(framework.RunID)

	// We don't use ObjectMeta.GenerateName feature, as in case of API call
	// failure we don't know whether the namespace was created and what is its
	// name.
	name := fmt.Sprintf("%v-%v", baseName, framework.RandomSuffix())

	namespaceObj := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "",
			Labels:    labels,
		},
		Status: v1.NamespaceStatus{},
	}
	// Be robust about making the namespace creation call.
	var got *v1.Namespace
	if err := wait.PollImmediate(2*time.Second, 30*time.Second, func() (bool, error) {
		var err error
		got, err = c.CoreV1().Namespaces().Create(context.TODO(), namespaceObj, metav1.CreateOptions{})
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				// regenerate on conflict
				framework.Logf("Namespace name %q was already taken, generate a new name and retry", namespaceObj.Name)
				namespaceObj.Name = fmt.Sprintf("%v-%v", baseName, framework.RandomSuffix())
			} else {
				framework.Logf("Unexpected error while creating namespace: %v", err)
			}
			return false, nil
		}
		return true, nil
	}); err != nil {
		return nil, err
	}

	if framework.TestContext.VerifyServiceAccount {
		if err := framework.WaitForDefaultServiceAccountInNamespace(c, got.Name); err != nil {
			// Even if we fail to create serviceAccount in the namespace,
			// we have successfully create a namespace.
			// So, return the created namespace.
			return got, err
		}
	}
	return got, nil
}

// TestMain does some initial setups before running the real TestE2E function
func TestMain(m *testing.M) {
	// Register test flags, then parse flags.
	handleFlags()

	flag.CommandLine.VisitAll(func(flag *flag.Flag) {
		log.Logf("FLAG: --%s=%q", flag.Name, flag.Value)
	})

	framework.AfterReadingAllFlags(&framework.TestContext)
	e2econfig.AfterReadingAllFlags()

	if framework.TestContext.RepoRoot != "" {
		testfiles.AddFileSource(testfiles.RootFileSource{Root: framework.TestContext.RepoRoot})
	}

	// See https://github.com/pingcap/tidb-operator/issues/2339
	framework.TestContext.CreateTestingNS = createTestingNS

	rand.Seed(time.Now().UnixNano())
	// run test functions with `go test`
	os.Exit(m.Run())
}

func TestE2E(t *testing.T) {
	RunE2ETests(t)
}
