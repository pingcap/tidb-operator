package data

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
)

func NewTSOGroup(ns string, patches ...GroupPatch[*runtime.TSOGroup]) *v1alpha1.TSOGroup {
	tg := &runtime.TSOGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      defaultTSOGroupName,
		},
		Spec: v1alpha1.TSOGroupSpec{
			Cluster:  v1alpha1.ClusterReference{Name: defaultClusterName},
			Replicas: ptr.To[int32](1),
			Template: v1alpha1.TSOTemplate{
				Spec: v1alpha1.TSOTemplateSpec{
					Version: defaultVersion,
					Image:   ptr.To(defaultImageRegistry + "pd"),
				},
			},
		},
	}
	for _, p := range patches {
		p(tg)
	}

	return runtime.ToTSOGroup(tg)
}
