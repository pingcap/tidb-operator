package tests

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (oa *operatorActions) DumpAllLogs(operatorInfo *OperatorInfo, testClusters []*TidbClusterInfo) error {
	logPath := fmt.Sprintf("/%s/%s", oa.logDir, "operator-stability")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		err = os.MkdirAll(logPath, os.ModePerm)
		if err != nil {
			return err
		}
	}

	// dump all resources info
	resourceLogFile, err := os.Create(filepath.Join(logPath, "resources"))
	if err != nil {
		return err
	}
	defer resourceLogFile.Close()
	resourceWriter := bufio.NewWriter(resourceLogFile)
	dumpLog("kubectl get po -owide -n kube-system", resourceWriter)
	dumpLog(fmt.Sprintf("kubectl get po -owide -n %s", operatorInfo.Namespace), resourceWriter)
	dumpLog("kubectl get pv", resourceWriter)
	dumpLog("kubectl get pv -oyaml", resourceWriter)
	for _, testCluster := range testClusters {
		dumpLog(fmt.Sprintf("kubectl get po,pvc,svc,cm,cronjobs,jobs,statefulsets,tidbclusters -owide -n %s", testCluster.Namespace), resourceWriter)
		dumpLog(fmt.Sprintf("kubectl get po,pvc,svc,cm,cronjobs,jobs,statefulsets,tidbclusters -n %s -oyaml", testCluster.Namespace), resourceWriter)
	}

	// dump operator components's log
	operatorPods, err := oa.kubeCli.CoreV1().Pods(operatorInfo.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, pod := range operatorPods.Items {
		err := dumpPod(logPath, &pod)
		if err != nil {
			return err
		}
	}

	// dump all test clusters's logs
	for _, testCluster := range testClusters {
		clusterPodList, err := oa.kubeCli.CoreV1().Pods(testCluster.Namespace).List(metav1.ListOptions{})
		if err != nil {
			return err
		}
		for _, pod := range clusterPodList.Items {
			err := dumpPod(logPath, &pod)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func dumpPod(logPath string, pod *corev1.Pod) error {
	logFile, err := os.Create(filepath.Join(logPath, fmt.Sprintf("%s-%s.log", pod.Name, pod.Namespace)))
	if err != nil {
		return err
	}
	defer logFile.Close()
	plogFile, err := os.Create(filepath.Join(logPath, fmt.Sprintf("%s-%s-p.log", pod.Name, pod.Namespace)))
	if err != nil {
		return err
	}
	defer plogFile.Close()
	logWriter := bufio.NewWriter(logFile)
	plogWriter := bufio.NewWriter(plogFile)
	for _, c := range pod.Spec.Containers {
		dumpLog(fmt.Sprintf("kubectl logs -n %s %s -c %s", pod.Namespace, pod.GetName(), c.Name), logWriter)
		dumpLog(fmt.Sprintf("kubectl logs -n %s %s -c %s -p", pod.Namespace, pod.GetName(), c.Name), plogWriter)
	}

	return nil
}

func dumpLog(cmdStr string, writer *bufio.Writer) {
	writer.WriteString(fmt.Sprintf("$ %s\n", cmdStr))
	data, err := exec.Command("/bin/sh", "-c", "/usr/local/bin/"+cmdStr).CombinedOutput()
	if err != nil {
		writer.WriteString(err.Error())
	}
	writer.WriteString(string(data))
}
