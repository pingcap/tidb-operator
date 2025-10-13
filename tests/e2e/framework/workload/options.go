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

package workload

import (
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

type Options struct {
	Port        int
	User        string
	Password    string
	TLS         bool
	CA          string
	CertKeyPair string

	RegionCount int

	// MaxExecutionTime is max_execution_time of tidb in millisecond
	// Default is 2000
	MaxExecutionTime int

	TiFlashReplicas int
}

type Option interface {
	With(opts *Options)
}

type WithOption func(opts *Options)

func (opt WithOption) With(opts *Options) {
	opt(opts)
}

func Port(port int) Option {
	return WithOption(func(opts *Options) {
		opts.Port = port
	})
}

func User(user, password string) Option {
	return WithOption(func(opts *Options) {
		opts.User = user
		opts.Password = password
	})
}

func TLS(ca, certKeyPair string) Option {
	return WithOption(func(opts *Options) {
		opts.TLS = true
		opts.CA = ca
		opts.CertKeyPair = certKeyPair
	})
}

func RegionCount(count int) Option {
	return WithOption(func(opts *Options) {
		opts.RegionCount = count
	})
}

func MaxExecutionTime(ms int) Option {
	return WithOption(func(opts *Options) {
		opts.MaxExecutionTime = ms
	})
}

func TiFlashReplicas(replicas int) Option {
	return WithOption(func(opts *Options) {
		opts.TiFlashReplicas = replicas
	})
}

func DefaultOptions() *Options {
	return &Options{
		Port: 4000,
		User: "root",

		MaxExecutionTime: 2000,

		RegionCount: 500,
	}
}

func ConfigJobWithTLS(job *batchv1.Job, o *Options) *batchv1.Job {
	if !o.TLS {
		return job
	}

	job.Spec.Template.Spec.Containers[0].Args = append(job.Spec.Template.Spec.Containers[0].Args,
		"--enable-tls",
		"--tls-mount-path", "/var/lib/tidb-tls",
	)

	var vol *corev1.Volume
	switch {
	case o.CA != "" && o.CertKeyPair != "" && o.CA != o.CertKeyPair:
		vol = &corev1.Volume{
			Name: "tls-certs",
			VolumeSource: corev1.VolumeSource{
				Projected: &corev1.ProjectedVolumeSource{
					Sources: []corev1.VolumeProjection{
						{
							Secret: &corev1.SecretProjection{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: o.CA,
								},
								Items: []corev1.KeyToPath{
									{
										Key:  corev1.ServiceAccountRootCAKey,
										Path: corev1.ServiceAccountRootCAKey,
									},
								},
							},
						},
						{
							Secret: &corev1.SecretProjection{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: o.CertKeyPair,
								},
								Items: []corev1.KeyToPath{
									{
										Key:  corev1.TLSCertKey,
										Path: corev1.TLSCertKey,
									},
									{
										Key:  corev1.TLSPrivateKeyKey,
										Path: corev1.TLSPrivateKeyKey,
									},
								},
							},
						},
					},
				},
			},
		}

	case o.CA != "" && o.CertKeyPair != "" && o.CA == o.CertKeyPair:
		vol = &corev1.Volume{
			Name: "tls-certs",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: o.CA,
				},
			},
		}

	case o.CA != "":
		vol = &corev1.Volume{
			Name: "tls-certs",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: o.CA,
				},
			},
		}
	case o.CertKeyPair != "":
		vol = &corev1.Volume{
			Name: "tls-certs",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: o.CertKeyPair,
				},
			},
		}
	}

	if vol == nil {
		job.Spec.Template.Spec.Containers[0].Args = append(job.Spec.Template.Spec.Containers[0].Args, "--tls-insecure-skip-verify")
		return job
	}

	job.Spec.Template.Spec.Volumes = append(job.Spec.Template.Spec.Volumes, *vol)
	job.Spec.Template.Spec.Containers[0].VolumeMounts = append(job.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
		Name:      "tls-certs",
		MountPath: "/var/lib/tidb-tls",
		ReadOnly:  true,
	})

	return job
}
