package k8s

import (
	"net/http"
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

func PrintVersionInfo() {
	version.PrintVersionInfo()
}

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
	makeGetCluster := makeGetTidbCluster(tcGetFn)
	go wait.Forever(func() {
		server.StartServer(makeGetCluster, port)
	}, 5*time.Second)
	glog.Fatal(http.ListenAndServe(":6060", nil))

}

func makeGetTidbCluster(tcGetFn func(ns, tcName string) (*v1alpha1.TidbCluster, error)) discovery.MakeGetCluster {
	return func(pdControl pdapi.PDControlInterface) discovery.GetCluster {
		return func(ns, tcName string) (discovery.Cluster, error) {
			tc, err := tcGetFn(ns, tcName)
			if err != nil {
				return discovery.Cluster{}, err
			}
			return discovery.Cluster{
				PDClient:        pdControl.GetPDClient(pdapi.Namespace(ns), tcName, tc.Spec.EnableTLSCluster),
				Replicas:        tc.Spec.PD.Replicas,
				ResourceVersion: tc.ResourceVersion,
				Scheme:          tc.Scheme(),
			}, nil
		}
	}
}

func makeRealTCGetFn(cli versioned.Interface) func(ns, tcName string) (*v1alpha1.TidbCluster, error) {
	return func(ns, tcName string) (*v1alpha1.TidbCluster, error) {
		return cli.PingcapV1alpha1().TidbClusters(ns).Get(tcName, metav1.GetOptions{})
	}
}
