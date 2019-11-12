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
	"time"
)

type InitializerConfig struct {
	AdmissionWebhookName string
	WebhookEnabled       bool
	Timeout              time.Duration
	OwnerPodName         string
}

type Initializer struct {
	kubeCli kubernetes.Interface
	config  *InitializerConfig
}

const (
	allComponent = "all"
)

func NewInitializer(kubeCli kubernetes.Interface, config *InitializerConfig) *Initializer {
	return &Initializer{
		kubeCli: kubeCli,
		config:  config,
	}
}

// Initializer generate resources for each component
func (initializer *Initializer) Run(namespace string, components []string) error {
	for _, component := range components {
		err := initializer.run(namespace, component)
		if err != nil {
			return err
		}
	}
	return nil
}

func (initializer *Initializer) run(namespace, component string) error {
	switch component {
	case initializer.config.AdmissionWebhookName:
		return initializer.webhookResourceInitializer(namespace)
	case allComponent:
		//init all component resources,currently there is only one component
		return initializer.run(namespace, initializer.config.AdmissionWebhookName)
	default:
		return fmt.Errorf("unknown initialize component")
	}
}
