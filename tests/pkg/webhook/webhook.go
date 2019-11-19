package webhook

import (
	"net/http"

	"k8s.io/apimachinery/pkg/util/sets"
)

type Interface interface {
	ServePods(w http.ResponseWriter, r *http.Request)
}

type webhook struct {
	tidbClusters sets.String
}

func NewWebhook(tidbClusters []string) Interface {
	return &webhook{
		tidbClusters: sets.NewString(tidbClusters...),
	}
}
