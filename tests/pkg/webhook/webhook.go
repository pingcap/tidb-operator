package webhook

import (
	"net/http"

	"k8s.io/apimachinery/pkg/util/sets"
)

type Interface interface {
	ServePods(w http.ResponseWriter, r *http.Request)
}

type webhook struct {
	namespaces sets.String
}

func NewWebhook(namespaces []string) Interface {
	return &webhook{
		namespaces: sets.NewString(namespaces...),
	}
}
