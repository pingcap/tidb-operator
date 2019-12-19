package tests

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"time"

	apps "k8s.io/api/apps/v1"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"

	"github.com/ghodss/yaml"

	"github.com/pingcap/tidb-operator/pkg/label"
	"k8s.io/apimachinery/pkg/labels"
	glog "k8s.io/klog"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/pingcap/tidb-operator/tests/slack"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilversion "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/klog"
)

func (oa *operatorActions) SwitchOperatorWebhook(isEnabled bool, info *OperatorConfig) error {
	klog.Infof("upgrading tidb-operator with admission webhook %v", isEnabled)

	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(
			label.New().Labels()).String(),
	}
	pods1, err := oa.kubeCli.CoreV1().Pods(metav1.NamespaceAll).List(listOptions)
	if err != nil {
		return err
	}
	m := map[string]string{}

	cmd := fmt.Sprintf(`helm upgrade %s %s --set-string %s `,
		info.ReleaseName,
		oa.operatorChartPath(info.Tag),
		info.OperatorHelmSetString(m))

	if isEnabled {
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
			data, err := yaml.Marshal(cabundle)
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
			cmd = fmt.Sprintf("%s --set-file admissionWebhook.cabundle=%s", cmd, cabundleFile.Name())
		}
	}

	klog.Info(cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to deploy operator: %v, %s", err, string(res))
	}
	klog.Infof("success to execute helm operator upgrade")

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

	klog.Infof("success to upgrade operator with webhook switch %v", isEnabled)
	return nil
}

func (oa *operatorActions) SwitchOperatorWebhookOrDie(isEnabled bool, info *OperatorConfig) {
	if err := oa.SwitchOperatorWebhook(isEnabled, info); err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (oa *operatorActions) SwitchOperatorStatefulSetWebhook(isEnabled bool, info *OperatorConfig) error {
	klog.Infof("switch Operator StatefulSet Webhook %v", isEnabled)
	m := map[string]string{
		"admissionWebhook.hooksEnabled.statefulSets": strconv.FormatBool(isEnabled),
	}
	cmd := fmt.Sprintf(`helm upgrade %s %s --set-string %s`,
		info.ReleaseName,
		oa.operatorChartPath(info.Tag),
		info.OperatorHelmSetString(m))

	klog.Info(cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to deploy operator: %v, %s", err, string(res))
	}

	// wait 10 sec to let sts webhook config honored
	time.Sleep(10 * time.Second)
	return nil
}

func (oa *operatorActions) SwitchOperatorStatefulSetWebhookOrDie(isEnabled bool, info *OperatorConfig) {
	if err := oa.SwitchOperatorStatefulSetWebhook(isEnabled, info); err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (oa *operatorActions) SwitchOperatorPodWebhook(isEnabled bool, info *OperatorConfig) error {
	klog.Infof("switch Operator Pod Webhook %v", isEnabled)
	m := map[string]string{
		"admissionWebhook.hooksEnabled.pods": strconv.FormatBool(isEnabled),
	}
	cmd := fmt.Sprintf(`helm upgrade %s %s --set-string %s`,
		oa.operatorChartPath(info.Tag),
		info.ReleaseName,
		info.OperatorHelmSetString(m))
	klog.Info(cmd)
	res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to deploy operator: %v, %s", err, string(res))
	}

	// wait 10 sec to let pod webhook config honored
	time.Sleep(10 * time.Second)
	return nil
}

func (oa *operatorActions) SwitchOperatorPodWebhookOrDie(isEnabled bool, info *OperatorConfig) {
	if err := oa.SwitchOperatorPodWebhook(isEnabled, info); err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (oa *operatorActions) CheckUpgradeWithPodWebhook(info *TidbClusterConfig) error {
	ns := info.Namespace
	tcName := info.ClusterName

	pdStsName := getStsName(tcName, v1alpha1.PDMemberType)
	tikvStsName := getStsName(tcName, v1alpha1.TiKVMemberType)
	tidbStsName := getStsName(tcName, v1alpha1.TiDBMemberType)

	pdDesiredReplicas, err := strconv.ParseInt(info.Resources["pd.replicas"], 10, 32)
	if err != nil {
		return err
	}
	tikvDesiredReplicas, err := strconv.ParseInt(info.Resources["tikv.replicas"], 10, 32)
	if err != nil {
		return err
	}
	tidbDesiredReplicas, err := strconv.ParseInt(info.Resources["tidb.replicas"], 10, 32)
	if err != nil {
		return err
	}

	tc, err := oa.cli.PingcapV1alpha1().TidbClusters(ns).Get(tcName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get tidbcluster: %s/%s, %v", ns, tcName, err)
	}

	f := func(stsName, namespace string, desiredReplicas int64) (*apps.StatefulSet, bool, error) {
		sts, err := oa.kubeCli.AppsV1().StatefulSets(ns).Get(stsName, metav1.GetOptions{})
		if err != nil {
			return nil, false, err
		}
		if sts.Status.UpdatedReplicas != int32(desiredReplicas) {
			return nil, false, nil
		}
		if sts.Status.CurrentReplicas != int32(desiredReplicas) {
			return nil, false, nil
		}
		return sts, true, nil
	}

	return wait.Poll(10*time.Second, 50*time.Minute, func() (done bool, err error) {

		pdsts, ready, err := f(pdStsName, ns, pdDesiredReplicas)
		if !ready || err != nil {
			return ready, err
		}
		if pdsts.Spec.Template.Spec.Containers[0].Image != info.PDImage {
			return false, nil
		}
		pdClient, cancel, err := oa.getPDClient(tc)
		if err != nil {
			return false, err
		}
		defer cancel()

		membersInfo, err := pdClient.GetMembers()
		if err != nil {
			return false, nil
		}
		if len(membersInfo.Members) != int(pdDesiredReplicas) {
			return false, nil
		}

		tikvsts, ready, err := f(tikvStsName, ns, tikvDesiredReplicas)
		if !ready || err != nil {
			return ready, err
		}
		if tikvsts.Spec.Template.Spec.Containers[0].Image != info.TiKVImage {
			return false, nil
		}
		storesInfo, err := pdClient.GetStores()
		if err != nil {
			return false, nil
		}
		if storesInfo.Count != int(tikvDesiredReplicas) {
			return false, nil
		}

		tidbsts, ready, err := f(tidbStsName, ns, tidbDesiredReplicas)
		if !ready || err != nil {
			return ready, err
		}

		if tidbsts.Spec.Template.Spec.Containers[0].Image != info.TiDBImage && tidbsts.Spec.Template.Spec.Containers[1].Image != info.TiDBImage {
			return false, nil
		}
		return true, nil
	})

}

func (oa *operatorActions) CheckUpgradeWithPodWebhookOrDie(info *TidbClusterConfig) {
	if err := oa.CheckUpgradeWithPodWebhook(info); err != nil {
		slack.NotifyAndPanic(err)
	}
}

func getStsName(tcName string, memberType v1alpha1.MemberType) string {
	return fmt.Sprintf("%s-%s", tcName, memberType.String())
}
