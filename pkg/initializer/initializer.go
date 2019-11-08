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

type Initializer struct {
	kubeCli kubernetes.Interface
}

var (
	WebhookEnabled bool
)

const (
	executePath          = "/etc/initializer/create-cert-script"
	allComponent         = "all"
	AdmissionWebhookName = "admission-webhook"
)

func NewInitializer(kubeCli kubernetes.Interface) *Initializer {
	return &Initializer{
		kubeCli: kubeCli,
	}
}

// Initializer generate resources for each component
func (initializer *Initializer) Run(namespace string, component string, days int) error {
	switch component {
	case AdmissionWebhookName:
		return initializer.webhookResourceIntializer(namespace, days)
	case allComponent:
		//init all component resources,currently there is only one component
		return initializer.Run(namespace, AdmissionWebhookName, days)
	default:
		return fmt.Errorf("unknown initialize component")
	}
}
