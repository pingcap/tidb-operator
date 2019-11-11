// Copyright 2019 PingCAP, Inc.
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

package initializer

import (
	"fmt"
	"k8s.io/client-go/kubernetes"
)

type InitializerConfig struct {
	AdmissionWebhookName string
	WebhookEnabled       bool
}

type Initializer struct {
	kubeCli kubernetes.Interface
	config  *InitializerConfig
}

const (
	executePath  = "/etc/initializer/create-cert-script"
	allComponent = "all"
)

func NewInitializer(kubeCli kubernetes.Interface, config *InitializerConfig) *Initializer {
	return &Initializer{
		kubeCli: kubeCli,
		config:  config,
	}
}

// Initializer generate resources for each component
func (initializer *Initializer) Run(podName, namespace string, components []string, days int) error {
	for _, component := range components {
		err := initializer.run(podName, namespace, component, days)
		if err != nil {
			return err
		}
	}
	return nil
}

func (initializer *Initializer) run(podName, namespace, component string, days int) error {
	switch component {
	case initializer.config.AdmissionWebhookName:
		return initializer.webhookResourceInitializer(podName, namespace, days)
	case allComponent:
		//init all component resources,currently there is only one component
		return initializer.run(podName, namespace, initializer.config.AdmissionWebhookName, days)
	default:
		return fmt.Errorf("unknown initialize component")
	}
}
