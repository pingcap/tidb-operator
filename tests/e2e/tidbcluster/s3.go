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
	"fmt"
	"github.com/minio/minio-go/v6"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	clientset "k8s.io/client-go/kubernetes"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	"os/exec"
	"time"
)

const (
	minioPodName = "minio"
)

func installAndWaitMinio(ns string, cli clientset.Interface) error {
	cmd := fmt.Sprintf(`kubectl apply -f /minio/minio.yaml -n %s`, ns)
	if data, err := exec.Command("sh", "-c", cmd).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to install minio %s %v", string(data), err)
	}
	return e2epod.WaitTimeoutForPodReadyInNamespace(cli, minioPodName, ns, 5*time.Minute)
}

func cleanMinio(ns string) error {
	cmd := fmt.Sprintf(`kubectl delete -f /minio/minio.yaml -n %s`, ns)
	if data, err := exec.Command("sh", "-c", cmd).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to clean minio %s %v", string(data), err)
	}
	return nil
}

func createMinioClient(fw portforward.PortForward, ns, accessKey, secretKey string, ssl bool) (*minio.Client, context.CancelFunc, error) {
	localHost, localPort, cancel, err := portforward.ForwardOnePort(fw, ns, "svc/minio-service", 9000)
	if err != nil {
		return nil, nil, err
	}
	ep := fmt.Sprintf("%s:%d", localHost, localPort)
	client, err := minio.New(ep, accessKey, secretKey, ssl)
	if err != nil {
		return nil, nil, err
	}
	return client, cancel, nil
}
