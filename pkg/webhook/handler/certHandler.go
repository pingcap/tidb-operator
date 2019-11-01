package handler

import (
	"fmt"
	"github.com/pingcap/tidb-operator/pkg/initializer"
	certUtil "github.com/pingcap/tidb-operator/pkg/util"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	glog "k8s.io/klog"
	"time"
)

type WebhookHandler struct {
	kubeCli kubernetes.Interface
}

type RefreshJobConfig struct {
	Namespace       string
	Image           string
	OwnerUid        string
	OwnerKind       string
	OwnerVersion    string
	OwnerName       string
	RefreshInterval int
}

func NewRefreshJobConfig(namespace, image, ownerUid, ownerKind, ownerVersion, ownerName string, refreshInterval int) *RefreshJobConfig {
	return &RefreshJobConfig{
		Namespace:       namespace,
		Image:           image,
		OwnerUid:        ownerUid,
		OwnerKind:       ownerKind,
		OwnerVersion:    ownerVersion,
		OwnerName:       ownerName,
		RefreshInterval: refreshInterval,
	}
}

func NewWebhookHandler(kubeCli kubernetes.Interface) *WebhookHandler {
	return &WebhookHandler{
		kubeCli: kubeCli,
	}
}

func (wh *WebhookHandler) RefreshCertPEMExpirationHandler(config *RefreshJobConfig) error {
	refreshIntervalHour := config.RefreshInterval
	secretName := initializer.SecretName
	secret, err := wh.kubeCli.CoreV1().Secrets(config.Namespace).Get(secretName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	certByte := secret.Data["cert.pem"]
	cert, err := certUtil.DecodeCertPem(certByte)
	if err != nil {
		return err
	}
	now := time.Now()
	expireDate := cert.NotAfter
	internal := expireDate.Sub(now)
	if internal.Hours() <= float64(refreshIntervalHour) {

		glog.Info("tidb-operator start to refresh ca cert")

		job := createNewJobToRefreshCert(config, generateRefreshJobName(now))
		_, err := wh.kubeCli.BatchV1().Jobs(config.Namespace).Create(job)

		if err != nil {
			glog.Infof("create refresh Job failed,:%v", err)
			return err
		}

	}
	return nil
}

func generateRefreshJobName(now time.Time) string {
	year, month, day := now.Date()
	return fmt.Sprintf("operator-cert-refresh-%d-%d-%d", year, month, day)
}

func createNewJobToRefreshCert(config *RefreshJobConfig, name string) *batchv1.Job {
	job := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: config.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: config.OwnerVersion,
					Kind:       config.OwnerKind,
					Name:       config.OwnerName,
					UID:        types.UID(config.OwnerUid),
				},
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            func() *int32 { a := int32(1000); return &a }(),
			TTLSecondsAfterFinished: func() *int32 { a := int32(60); return &a }(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: config.Namespace,
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure,
					Containers: []corev1.Container{
						{
							Name:            "tidb-operator-refresh-cert-job",
							Image:           config.Image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command: []string{
								"/usr/local/bin/tidb-initializer",
							},
							Env: []corev1.EnvVar{
								{
									Name: "NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											APIVersion: "v1",
											FieldPath:  "metadata.namespace",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return &job
}
