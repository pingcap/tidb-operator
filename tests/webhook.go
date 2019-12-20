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
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/ghodss/yaml"

	"github.com/pingcap/tidb-operator/tests/slack"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilversion "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/klog"
)

func (oa *operatorActions) SwitchOperatorWebhook(isWebhookEnabled, isPodWebhookEnable, isStsWebhookEnabled bool, info *OperatorConfig) error {
	klog.Infof("upgrading tidb-operator with admission webhook %v,pod %v,sts %v", isWebhookEnabled, isPodWebhookEnable, isStsWebhookEnabled)

	//listOptions := metav1.ListOptions{
	//	LabelSelector: labels.SelectorFromSet(
	//		label.New().Labels()).String(),
	//}
	//pods1, err := oa.kubeCli.CoreV1().Pods(metav1.NamespaceAll).List(listOptions)
	//if err != nil {
	//	return err
	//}
	switchWebhook := fmt.Sprintf("%s=%v,%s=%v,%s=%v",
		"admissionWebhook.create", isWebhookEnabled,
		"admissionWebhook.hooksEnabled.pds", isPodWebhookEnable,
		"admissionWebhook.hooksEnabled.statefulSets", isStsWebhookEnabled)

	setString := info.OperatorHelmSetString(nil)
	setString = strings.Join([]string{setString, switchWebhook}, ",")

	cmd := fmt.Sprintf(`helm upgrade %s %s --set-string %s `,
		info.ReleaseName,
		oa.operatorChartPath(info.Tag),
		setString)

	if isWebhookEnabled {
		serverVersion, err := oa.kubeCli.Discovery().ServerVersion()
		if err != nil {
			return fmt.Errorf("failed to get api server version")
		}
		sv := utilversion.MustParseSemantic(serverVersion.GitVersion)
		klog.Infof("ServerVersion: %v", serverVersion.String())
		if sv.LessThan(utilversion.MustParseSemantic("v1.13.0")) {
			cm, err := oa.kubeCli.CoreV1().ConfigMaps("kube-system").Get("extension-apiserver-authentication", metav1.GetOptions{})
			if err != nil {
				return err
			}
			cabundle := cm.Data["client-ca-file"]
			cabundleDir, err := ioutil.TempDir("", "test-e2e-cabundle")
			if err != nil {
				return err
			}
			defer os.RemoveAll(cabundleDir)
			cabundleFile, err := ioutil.TempFile(cabundleDir, "cabundle")
			if err != nil {
				return err
			}
			m := map[string]map[string]string{
				"admissionWebhook": {
					"cabundle": cabundle,
				},
			}
			data, err := yaml.Marshal(m)
			if err != nil {
				return err
			}
			err = ioutil.WriteFile(cabundleFile.Name(), data, 0644)
			if err != nil {
				return err
			}
			dataByte, err := ioutil.ReadFile(cabundleFile.Name())
			if err != nil {
				return err
			}
			klog.Infof("%s", string(dataByte[:]))
			cmd = fmt.Sprintf("%s -f %s", cmd, cabundleFile.Name())
		}
	}

	klog.Info(cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to deploy operator: %v, %s", err, string(res))
	}
	klog.Infof("success to execute helm operator upgrade")

	//// ensure pods unchanged when upgrading operator
	//waitFn := func() (done bool, err error) {
	//	pods2, err := oa.kubeCli.CoreV1().Pods(metav1.NamespaceAll).List(listOptions)
	//	if err != nil {
	//		glog.Error(err)
	//		return false, nil
	//	}
	//
	//	err = ensurePodsUnchanged(pods1, pods2)
	//	if err != nil {
	//		return true, err
	//	}
	//
	//	return false, nil
	//}
	//
	//err = wait.Poll(oa.pollInterval, 5*time.Minute, waitFn)
	//if err == wait.ErrWaitTimeout {
	//	return nil
	//}

	klog.Infof("success to upgrade operator with webhook switch %v, pod hook %v, sts hook %v", isWebhookEnabled, isPodWebhookEnable, isStsWebhookEnabled)
	return nil
}

func (oa *operatorActions) SwitchOperatorWebhookOrDie(isWebhookEnabled, isPodWebhookEnable, isStsWebhookEnabled bool, info *OperatorConfig) {
	if err := oa.SwitchOperatorWebhook(isWebhookEnabled, isPodWebhookEnable, isStsWebhookEnabled, info); err != nil {
		slack.NotifyAndPanic(err)
	}
}
