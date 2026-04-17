package compact

import (
	"context"
	"testing"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
)

func newTestController(t *testing.T) *Controller {
	t.Helper()

	deps := controller.NewFakeDependencies()
	fakeDiscovery, ok := deps.KubeClientset.Discovery().(*fakediscovery.FakeDiscovery)
	if !ok {
		t.Fatalf("expected fake discovery client, got %T", deps.KubeClientset.Discovery())
	}
	fakeDiscovery.FakedServerVersion = &version.Info{Major: "1", Minor: "29"}

	tc := &v1alpha1.TidbCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-tc",
			Namespace: "demo-ns",
		},
		Spec: v1alpha1.TidbClusterSpec{
			Version: "v8.5.0",
			TiKV: &v1alpha1.TiKVSpec{
				BaseImage: "pingcap/tikv",
			},
		},
	}

	if err := deps.InformerFactory.Pingcap().V1alpha1().TidbClusters().Informer().GetIndexer().Add(tc); err != nil {
		t.Fatalf("failed to seed tidbcluster indexer: %v", err)
	}

	if _, err := deps.Clientset.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(context.TODO(), tc, metav1.CreateOptions{}); err != nil {
		t.Fatalf("failed to seed tidbcluster client: %v", err)
	}

	return NewController(deps)
}
