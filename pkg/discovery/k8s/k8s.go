package k8s

import (
	"strings"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/discovery"
	"github.com/pingcap/tidb-operator/pkg/discovery/server"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/component-base/logs"
	"k8s.io/client-go/rest"
)

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

	discover := newK8sClusterRefresh(cli)
	glog.Fatal(server.StartServer(discover, port))

}

type tiDBGetCluster struct {
	tcGetFn   func(ns, tcName string) (*v1alpha1.TidbCluster, error)
	pdControl pdapi.PDControlInterface
}

var _ discovery.ClusterRefreshMembers = &tiDBGetCluster{}

func clusterIDSplit(clusterID string) (string, string) {
	nsTCName := strings.Split(clusterID, "/")
	ns := nsTCName[0]
	tcName := nsTCName[1]
	return ns, tcName
}

func (tgc tiDBGetCluster) getTC(clusterID string) (*v1alpha1.TidbCluster, error) {
	ns, tcName := clusterIDSplit(clusterID)
	tc, err := tgc.tcGetFn(ns, tcName)
	if err != nil {
		return nil, err
	}
	return tc, nil
}

func (tgc tiDBGetCluster) GetMembers(clusterID string) (*pdapi.MembersInfo, error) {
	tc, err := tgc.getTC(clusterID)
	if err != nil {
		return nil, err
	}
	ns, tcName := clusterIDSplit(clusterID)
	pdClient := tgc.pdControl.GetPDClient(pdapi.Namespace(ns), tcName, tc.Spec.EnableTLSCluster)
	return pdClient.GetMembers()
}

func (tgc tiDBGetCluster) GetCluster(clusterID string) (discovery.Cluster, error) {
	tc, err := tgc.getTC(clusterID)
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

func newK8sClusterRefresh(cli versioned.Interface) discovery.TiDBDiscovery {
	return discovery.NewTiDBDiscoveryWaitMembers(tiDBGetCluster{
		tcGetFn:   makeRealTCGetFn(cli),
		pdControl: pdapi.NewDefaultPDControl(),
	})
}

func makeRealTCGetFn(cli versioned.Interface) func(ns, tcName string) (*v1alpha1.TidbCluster, error) {
	return func(ns, tcName string) (*v1alpha1.TidbCluster, error) {
		return cli.PingcapV1alpha1().TidbClusters(ns).Get(tcName, metav1.GetOptions{})
	}
}
