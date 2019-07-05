package member

import (
	corev1 "k8s.io/api/core/v1"
)

// WaitForPDContainer gives the container spec for the wait-for-pd init container
func WaitForPDContainer(tcName string, operatorImage string) corev1.Container {
	initEnvs := []corev1.EnvVar{
		{
			Name: "NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		{
			Name:  "CLUSTER_NAME",
			Value: tcName,
		},
	}

	return corev1.Container{
		Name:    "wait-for-pd",
		Image:   operatorImage,
		Command: []string{"wait-for-pd"},
		Env:     initEnvs,
	}
}
