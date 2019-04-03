package main

import (
	"flag"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/golang/glog"
	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/tests/pkg/util"
	"github.com/pingcap/tidb-operator/tests/pkg/util/client"
	"github.com/pingcap/tidb-operator/tests/pkg/util/ops"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func main() {
	// TODO: the cluster need to be setup manually
	const maxStoreDownTime = 5 * time.Minute
	client.SetConfigPath(filepath.Join(os.Getenv("HOME"), ".kube/config"))

	cli := client.NewOrDie()
	tikvOps := ops.TiKVOps{ClientOps: client.ClientOps{Client: cli}}

	opts := ops.TruncateOptions{
		Namespace: "tidb",
		Cluster:   "demo",
	}

	// load the cluster
	tc, err := cli.PingcapV1alpha1().TidbClusters(opts.Namespace).Get(opts.Cluster, metav1.GetOptions{})
	if err != nil {
		glog.Fatalf("failed to get cluster: ns=%s name=%s", opts.Namespace, opts.Cluster)
	}

	// find a store
	var store v1alpha1.TiKVStore
	for k, v := range tc.Status.TiKV.Stores {
		if v.State != v1alpha1.TiKVStateUp {
			continue
		}
		opts.Store, store = k, v
		break
	}
	if len(opts.Store) == 0 {
		glog.Fatalf("failed to get a store")
	} else {
		glog.Infof("use store: id=%s", opts.Store)
	}

	// checkout pod status
	podBeforeRestart, err := cli.CoreV1().Pods(opts.Namespace).Get(store.PodName, metav1.GetOptions{})
	if err != nil {
		glog.Fatalf("get pod: pod=%s err=%s", store.PodName, err.Error())
	}

	var rc int32
	if c := util.GetContainerStatusFromPod(podBeforeRestart, func(status corev1.ContainerStatus) bool {
		return status.Name == "tikv"
	}); c != nil {
		rc = c.RestartCount
	} else {
		glog.Fatalf("get container status from tikv pod")
	}

	// trigger restart of tikv to ensure sst files
	err = tikvOps.KillProcess(opts.Namespace, store.PodName, "tikv", 1, syscall.SIGTERM)
	if err != nil {
		glog.Fatalf("kill tikv: pod=%s err=%s", store.PodName, err.Error())
	}

	err = tikvOps.WaitForPod(opts.Namespace, store.PodName,
		func(pod *corev1.Pod, err error) (bool, error) {
			if pod == nil {
				glog.Warningf("pod is nil: err=%s", err.Error())
				return false, nil
			}
			tikv := util.GetContainerStatusFromPod(pod, func(status corev1.ContainerStatus) bool {
				return status.Name == "tikv"
			})

			if pod.Status.Phase == corev1.PodRunning && tikv != nil && tikv.RestartCount > rc {
				return true, nil
			}
			return false, nil
		})
	if err != nil {
		glog.Fatalf("pod isn't running: err=", err.Error())
	}

	err = tikvOps.TruncateSSTFile(opts, func(sst string) error {

		tc, err := cli.PingcapV1alpha1().TidbClusters(opts.Namespace).Get(opts.Cluster, metav1.GetOptions{})
		if err != nil {
			return errors.Annotate(err, "get cluster info before kill tikv")
		}
		failures := len(tc.Status.TiKV.FailureStores)

		err = tikvOps.KillProcess(opts.Namespace, store.PodName, "tikv", 1, syscall.SIGTERM)
		if err != nil {
			glog.Fatalf("kill tikv: pod=%s err=%s", store.PodName, err.Error())
		}

		return tikvOps.WaitForTiDBCluster(opts.Namespace, store.PodName,
			func(tc *v1alpha1.TidbCluster, err error) (bool, error) {
				if len(tc.Status.TiKV.FailureStores) > failures {
					return true, nil
				}
				return false, nil
			}, client.Poll(time.Minute, maxStoreDownTime+5*time.Minute))
	})

	if err != nil {
		glog.Fatalf("store is not up in time")
	}

	glog.Infof("tikv failover test passed")
}

func init() {
	// parse flags for glog
	flag.Parse()
}
