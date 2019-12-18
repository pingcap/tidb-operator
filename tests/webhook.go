package tests

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/pingcap/tidb-operator/pkg/label"
	"k8s.io/apimachinery/pkg/labels"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/pingcap/tidb-operator/tests/slack"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilversion "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/klog"
	glog "k8s.io/klog"
)

func (oa *operatorActions) UpgradeOperatorWithWebhookEnabled(info *OperatorConfig) error {
	klog.Infof("upgrading tidb-operator with admission webhook enabled")

	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(
			label.New().Labels()).String(),
	}
	pods1, err := oa.kubeCli.CoreV1().Pods(metav1.NamespaceAll).List(listOptions)
	if err != nil {
		return err
	}

	m := map[string]string{
		"admissionWebhook.enabled": "true",
	}

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
		m["admissionWebhook.cabundle"] = cabundle
	}
	cmd := fmt.Sprintf(`helm upgrade %s %s --set-string %s`,
		oa.operatorChartPath(info.Tag),
		info.ReleaseName,
		info.OperatorHelmSetString(m))

	glog.Info(cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to deploy operator: %v, %s", err, string(res))
	}

	// ensure pods unchanged when upgrading operator
	waitFn := func() (done bool, err error) {
		pods2, err := oa.kubeCli.CoreV1().Pods(metav1.NamespaceAll).List(listOptions)
		if err != nil {
			glog.Error(err)
			return false, nil
		}

		err = ensurePodsUnchanged(pods1, pods2)
		if err != nil {
			return true, err
		}

		return false, nil
	}

	err = wait.Poll(oa.pollInterval, 5*time.Minute, waitFn)
	if err == wait.ErrWaitTimeout {
		return nil
	}

	return nil
}

func (oa *operatorActions) UpgradeOperatorWithWebhookEnabledOrDie(info *OperatorConfig) error {
	if err := oa.UpgradeOperatorWithWebhookEnabled(info); err != nil {
		slack.NotifyAndPanic(err)
	}
	return nil
}
