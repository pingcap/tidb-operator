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

package tests

import (
	"encoding/base64"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilversion "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/klog"
)

func (oa *operatorActions) setCabundleFromApiServer(info *OperatorConfig) error {

	serverVersion, err := oa.kubeCli.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("failed to get api server version")
	}
	sv := utilversion.MustParseSemantic(serverVersion.GitVersion)
	klog.Infof("ServerVersion: %v", serverVersion.String())

	if sv.LessThan(utilversion.MustParseSemantic("v1.13.0")) && len(info.Cabundle) < 1 {
		namespace := "kube-system"
		name := "extension-apiserver-authentication"
		cm, err := oa.kubeCli.CoreV1().ConfigMaps(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		content, existed := cm.Data["client-ca-file"]
		if !existed {
			return fmt.Errorf("failed to get caBundle from configmap[%s/%s]", namespace, name)
		}
		info.Cabundle = base64.StdEncoding.EncodeToString([]byte(content))
		return nil
	}
	return nil
}
