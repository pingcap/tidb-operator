package collect

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// scheme is the scheme of kubernetes objects' to be collected.
var scheme = runtime.NewScheme()

// GetScheme returns the scheme of kubernetes objects to be collected.
func GetScheme() *runtime.Scheme {
	return scheme
}
