package k8s

import (
	"net/http"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/discovery"
	"github.com/pingcap/tidb-operator/pkg/discovery/server"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/pkg/version"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/util/logs"
	"k8s.io/client-go/rest"
)

// PrintVerstionInfo calls version.PrintVersionInfo
func PrintVersionInfo() {
	version.PrintVersionInfo()
}

// LogVerstionInfo calls version.LogVersionInfo
func LogVersionInfo() {
	version.LogVersionInfo()
}

// RunDiscoveryService blocks forever
// Errors in this function are fatal
func RunDiscoveryService(port int) {
	logs.InitLogs()
	defer logs.FlushLogs()

	cfg, err := rest.InClusterConfig()
	if err != nil {
		glog.Fatalf("failed to get config: %v", err)
	}
	cli, err := versioned.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("failed to create Clientset: %v", err)
	}

	tcGetFn := makeRealTCGetFn(cli)
	discover := newK8sClusterRefresh(tcGetFn)
	go wait.Forever(func() {
		server.StartServer(discover, port)
	}, 5*time.Second)
	glog.Fatal(http.ListenAndServe(":6060", nil))

}

type tiDBGetCluster struct {
	tcGetFn   func(ns, tcName string) (*v1alpha1.TidbCluster, error)
	pdControl pdapi.PDControlInterface
}

func (tgc tiDBGetCluster) getTC(clusterID string) (string, string, *v1alpha1.TidbCluster, error) {
	nsTCName := strings.Split(clusterID, "/")
	ns := nsTCName[0]
	tcName := nsTCName[1]
	tc, err := tgc.tcGetFn(ns, tcName)
	if err != nil {
		return ns, tcName, nil, err
	}
	return ns, tcName, tc, nil
}

func (tgc tiDBGetCluster) GetMembers(clusterID string) (*pdapi.MembersInfo, error) {
	ns, tcName, tc, err := tgc.getTC(clusterID)
	if err != nil {
		return nil, err
	}
	pdClient := tgc.pdControl.GetPDClient(pdapi.Namespace(ns), tcName, tc.Spec.EnableTLSCluster)
	return pdClient.GetMembers()
}

func (tgc tiDBGetCluster) GetCluster(clusterID string) (discovery.Cluster, error) {
	_, _, tc, err := tgc.getTC(clusterID)
	if err != nil {
		return discovery.Cluster{}, err
	}

	// TODO: the replicas should be the total replicas of pd sets.
	return discovery.Cluster{
		Replicas:        tc.Spec.PD.Replicas,
		ResourceVersion: tc.ResourceVersion,
		Scheme:          tc.Scheme(),
	}, nil
}

func newK8sClusterRefresh(tcGetFn func(ns, tcName string) (*v1alpha1.TidbCluster, error)) discovery.TiDBDiscovery {
	pdControl := pdapi.NewDefaultPDControl()
	return discovery.NewTiDBDiscoveryWaitMembers(tiDBGetCluster{
		tcGetFn:   tcGetFn,
		pdControl: pdControl,
	})
}

func makeRealTCGetFn(cli versioned.Interface) func(ns, tcName string) (*v1alpha1.TidbCluster, error) {
	return func(ns, tcName string) (*v1alpha1.TidbCluster, error) {
		return cli.PingcapV1alpha1().TidbClusters(ns).Get(tcName, metav1.GetOptions{})
	}
}
