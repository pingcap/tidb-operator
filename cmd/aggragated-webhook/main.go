package main

import (
	"github.com/openshift/generic-admission-server/pkg/cmd"
	"github.com/pingcap/tidb-operator/pkg/webhook"
)

func main() {
	podAh := &webhook.PodAdmissionHook{}
	cmd.RunAdmissionServer(podAh)
}
