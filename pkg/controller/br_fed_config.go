// Copyright 2023 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"flag"
	"time"
)

const (
	defaultFederationKubeConfigPath = "/etc/br-federation/federation-kubeconfig/kubeconfig"
)

// BrFedCLIConfig is used to save the configuration of br-federation-manager read from command line.
type BrFedCLIConfig struct {
	PrintVersion bool
	// The number of workers that are allowed to sync concurrently.
	// Larger number = more responsive management, but more CPU
	// (and network) load
	Workers int
	// Controls whether operator should manage kubernetes cluster
	// wide TiDB clusters
	ClusterScoped bool

	LeaseDuration time.Duration
	RenewDeadline time.Duration
	RetryPeriod   time.Duration
	WaitDuration  time.Duration
	// ResyncDuration is the resync time of informer
	ResyncDuration time.Duration

	// KubeClientQPS indicates the maximum QPS to the kubenetes API server from client.
	KubeClientQPS   float64
	KubeClientBurst int

	// FederationKubeConfigPath is the path to the kubeconfig file of data planes
	FederationKubeConfigPath string
}

// DefaultBrFedCLIConfig returns the default command line configuration
func DefaultBrFedCLIConfig() *BrFedCLIConfig {
	return &BrFedCLIConfig{
		Workers:        5,
		ClusterScoped:  true,
		LeaseDuration:  15 * time.Second,
		RenewDeadline:  10 * time.Second,
		RetryPeriod:    2 * time.Second,
		WaitDuration:   5 * time.Second,
		ResyncDuration: 30 * time.Second,

		FederationKubeConfigPath: defaultFederationKubeConfigPath,
	}
}

// AddFlag adds a flag for setting global feature gates to the specified FlagSet.
func (c *BrFedCLIConfig) AddFlag(_ *flag.FlagSet) {
	flag.BoolVar(&c.PrintVersion, "V", false, "Show version and quit")
	flag.BoolVar(&c.PrintVersion, "version", false, "Show version and quit")
	flag.IntVar(&c.Workers, "workers", c.Workers, "The number of workers that are allowed to sync concurrently. Larger number = more responsive management, but more CPU (and network) load")
	flag.BoolVar(&c.ClusterScoped, "cluster-scoped", c.ClusterScoped, "Whether br-federation-manager should manage kubernetes cluster-wide resources")
	flag.DurationVar(&c.ResyncDuration, "resync-duration", c.ResyncDuration, "Resync time of informer")

	// see https://pkg.go.dev/k8s.io/client-go/tools/leaderelection#LeaderElectionConfig for the config
	flag.DurationVar(&c.LeaseDuration, "leader-lease-duration", c.LeaseDuration, "leader-lease-duration is the duration that non-leader candidates will wait to force acquire leadership")
	flag.DurationVar(&c.RenewDeadline, "leader-renew-deadline", c.RenewDeadline, "leader-renew-deadline is the duration that the acting master will retry refreshing leadership before giving up")
	flag.DurationVar(&c.RetryPeriod, "leader-retry-period", c.RetryPeriod, "leader-retry-period is the duration the LeaderElector clients should wait between tries of actions")
	flag.Float64Var(&c.KubeClientQPS, "kube-client-qps", c.KubeClientQPS, "The maximum QPS to the kubenetes API server from client")
	flag.IntVar(&c.KubeClientBurst, "kube-client-burst", c.KubeClientBurst, "The maximum burst for throttle to the kubenetes API server from client")

	flag.StringVar(&c.FederationKubeConfigPath, "federation-kubeconfig-path", c.FederationKubeConfigPath, "The path to the kubeconfig file of data planes")
}
