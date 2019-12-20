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

func (oa *operatorActions) SwitchOperatorWebhook(info *OperatorConfig) error {
	isWebhookEnabled := info.WebhookEnabled
	isPodWebhookEnable := info.PodWebhookEnabled
	isStsWebhookEnabled := info.StsWebhookEnabled
	klog.Infof("upgrading tidb-operator with admission webhook %v,pod %v,sts %v", isWebhookEnabled, isPodWebhookEnable, isStsWebhookEnabled)

	switchWebhook := fmt.Sprintf("%s=%v,%s=%v,%s=%v",
		"admissionWebhook.create", isWebhookEnabled,
		"admissionWebhook.hooksEnabled.pods", isPodWebhookEnable,
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
			cmd = fmt.Sprintf("%s -f %s", cmd, cabundleFile.Name())
		}
	}

	klog.Info(cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to deploy operator: %v, %s", err, string(res))
	}
	klog.Infof("success to upgrade operator with webhook by %s", cmd)
	return nil
}

func (oa *operatorActions) SwitchOperatorWebhookOrDie(info *OperatorConfig) {
	if err := oa.SwitchOperatorWebhook(info); err != nil {
		slack.NotifyAndPanic(err)
	}
}
