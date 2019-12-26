package webhook

import (
	"net/http"

	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
)

type Interface interface {
	ServePods(w http.ResponseWriter, r *http.Request)
}

type webhook struct {
	namespaces sets.String
}

func NewWebhook(kubeCli kubernetes.Interface, versionCli versioned.Interface, namespaces []string) Interface {
	return &webhook{
		namespaces: sets.NewString(namespaces...),
	}
}
