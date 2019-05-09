// Copyright 2018 PingCAP, Inc.
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

package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/webhook"
	"github.com/pingcap/tidb-operator/pkg/webhook/util"
)

func main() {

	cli, kubeCli, err := util.GetNewClient()
	if err != nil {
		glog.Fatalf("failed to get client: %v", err)
	}

	ns := os.Getenv("NAMESPACE")
	if ns == "" {
		glog.Fatalf("fail to get namespace in environment")
	}

	svc := os.Getenv("SERVICENAME")
	if svc == "" {
		glog.Fatalf("fail to get servicename in environment")
	}

	// create cert file
	cert, err := util.SetupServerCert(ns, svc)
	if err != nil {
		glog.Fatalf("fail to setup server cert: %v", err)
	}
	webhookServer := webhook.NewWebHookServer(kubeCli, cli, cert)

	// before start webhook server, create validating-webhook-configuration
	err = webhookServer.RegisterWebhook(ns, svc)
	if err != nil {
		glog.Fatalf("fail to create validaing webhook configuration: %v", err)
	}

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs

		// FIXME Consider whether delete the configuration when the service is shutdown.
		if err := webhookServer.UnregisterWebhook(); err != nil {
			glog.Errorf("fail to delete validating configuration %v", err)
		}

		// Graceful shutdown the server
		if err := webhookServer.Shutdown(); err != nil {
			glog.Errorf("fail to shutdown server %v", err)
		}

		done <- true
	}()

	if err := webhookServer.Run(); err != nil {
		glog.Errorf("stop http server %v", err)
	}

	<-done

	glog.Infof("webhook server terminate safely.")
}
