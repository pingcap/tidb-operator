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

package initializer

import (
	"fmt"
	glog "k8s.io/klog"
	"os/exec"
	"strconv"
)

func SecretNameForServiceCert(serviceName string) string {
	return fmt.Sprintf("%s-cert", serviceName)
}

func VACNameForService(serviceName, namespace string) string {
	return fmt.Sprintf("%s.%s", serviceName, namespace)
}

func GenerateSecretAndCSR(serviceName, namespace string, days int) error {

	_, err := exec.Command("/bin/sh", executePath, "-n", namespace, "-s", serviceName, "-d", strconv.Itoa(days)).CombinedOutput()
	if err != nil {
		glog.Errorf("execute ca cert failed for service[%s/%s],%v", namespace, serviceName, err)
		return err
	}
	return nil
}
