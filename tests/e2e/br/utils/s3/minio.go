// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package s3

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/minio/minio-go/v6"
	"github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/k8s"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

const (
	minioName  = "minio"
	minioImage = "gcr.io/pingcap-public/third-party/minio/minio:RELEASE.2024-09-13T20-26-02Z"

	minioBucket = "local" // the bucket for e2e test
	minioSecret = "minio-secret"
)

type minioStorage struct {
	c client.Client
	// use portforward to visit service if e2e is not run in cluster
	fw k8s.PortForwarder
}

func NewMinio(c client.Client, fw k8s.PortForwarder) Interface {
	return &minioStorage{
		c:  c,
		fw: fw,
	}
}

func (s *minioStorage) Init(ctx context.Context, ns, accessKey, secretKey string) error {
	ginkgo.By("init minio s3 storage")
	pod := getMinioPod(ns)
	if err := s.c.Create(context.TODO(), pod); err != nil {
		return err
	}
	svc := getMinioService(ns)
	if err := s.c.Create(context.TODO(), svc); err != nil {
		return err
	}
	secret := getMinioSecret(ns, accessKey, secretKey)
	if err := s.c.Create(context.TODO(), secret); err != nil {
		return err
	}
	ginkgo.By("wait for minio s3 storage ready")

	if err := waiter.WaitForPodReadyInNamespace(ctx, s.c, pod, 5*time.Minute); err != nil {
		return err
	}

	ep, err := s.forwardPort(ctx, ns)
	if err != nil {
		return err
	}

	mc, err := minio.New(ep, accessKey, secretKey, false)
	if err != nil {
		return err
	}
	if err = mc.MakeBucket(minioBucket, ""); err != nil {
		return err
	}
	return nil
}

func (s *minioStorage) forwardPort(ctx context.Context, ns string) (string, error) {
	if s.fw == nil {
		return getDefaultAddr(ns), nil
	}
	host, port, _, err := k8s.ForwardOnePort(s.fw, ns, "svc/"+minioName, 9000)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%d", host, port), nil
}

func getDefaultAddr(ns string) string {
	return fmt.Sprintf("%s.%s:9000", minioName, ns)
}

func (s *minioStorage) Config(ns, prefix string) *v1alpha1.S3StorageProvider {
	return &v1alpha1.S3StorageProvider{
		Provider:   v1alpha1.S3StorageProviderTypeCeph,
		Prefix:     prefix,
		SecretName: minioSecret,
		Bucket:     minioBucket,
		// scheme is necessary
		Endpoint: "http://" + getDefaultAddr(ns),
	}
}

// clean by deleting namespace, so just return
func (s *minioStorage) Clean(ctx context.Context, ns string) error {
	return nil
}

func (s *minioStorage) Bucket() string {
	return minioBucket
}

func (s *minioStorage) IsDataCleaned(ctx context.Context, ns, prefix string) (bool, error) {
	accessKey, secretKey, err := s.accessSecret(ns)
	if err != nil {
		return false, err
	}
	ep, err := s.forwardPort(ctx, ns)
	if err != nil {
		return false, err
	}
	mc, err := minio.New(ep, accessKey, secretKey, false)
	if err != nil {
		return false, err
	}
	doneCh := make(chan struct{})
	defer close(doneCh)
	objs := mc.ListObjects(minioBucket, prefix, true, doneCh)
	if len(objs) == 0 {
		return true, nil
	}
	return false, nil
}

func (s *minioStorage) accessSecret(ns string) (string, string, error) {
	secret := &corev1.Secret{}
	err := s.c.Get(context.TODO(), types.NamespacedName{Namespace: ns, Name: minioSecret}, secret)
	if err != nil {
		return "", "", err
	}
	accessKeyBytes, ok1 := secret.Data["access_key"]
	secretKeyBytes, ok2 := secret.Data["secret_key"]
	if !ok1 || !ok2 {
		return "", "", fmt.Errorf("access_key or secret_key not found")
	}
	accessKey, err := base64DecodeToString(accessKeyBytes)
	if err != nil {
		return "", "", err
	}
	secretKey, err := base64DecodeToString(secretKeyBytes)
	if err != nil {
		return "", "", err
	}

	return accessKey, secretKey, nil
}

func base64DecodeToString(src []byte) (string, error) {
	dstLen := base64.StdEncoding.DecodedLen(len(src))
	dst := make([]byte, dstLen)
	if _, err := base64.StdEncoding.Decode(dst, src); err != nil {
		return "", err
	}
	return string(dst), nil
}

func getMinioSecret(ns, accessKey, secretKey string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      minioSecret,
			Namespace: ns,
		},
		StringData: map[string]string{
			"access_key": accessKey,
			"secret_key": secretKey,
		},
		Type: corev1.SecretTypeOpaque,
	}
}

func getMinioService(ns string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      minioName,
			Namespace: ns,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": minioName,
			},
			Ports: []corev1.ServicePort{
				{
					Port:       9000,
					TargetPort: intstr.FromInt(9000),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}

func getMinioPod(ns string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      minioName,
			Namespace: ns,
			Labels: map[string]string{
				"app": minioName,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  minioName,
					Image: minioImage,
					Args: []string{
						"server", "/data",
					},
					Env: []corev1.EnvVar{
						{
							Name: "MINIO_ACCESS_KEY",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: minioSecret,
									},
									Key: "access_key",
								},
							},
						},
						{
							Name: "MINIO_SECRET_KEY",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: minioSecret,
									},
									Key: "secret_key",
								},
							},
						},
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path: "/minio/health/ready",
								Port: intstr.FromInt(9000),
							},
						},
					},
					LivenessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path: "/minio/health/live",
								Port: intstr.FromInt(9000),
							},
						},
					},
				},
			},
		},
	}
}
