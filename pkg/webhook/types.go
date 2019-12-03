package webhook

import (
	"github.com/pingcap/tidb-operator/pkg/webhook/pod"
	"github.com/pingcap/tidb-operator/pkg/webhook/statefulset"
	"sync"
)

type PodAdmissionHook struct {
	lock        sync.RWMutex
	initialized bool

	podAC *pod.PodAdmissionControl
}

type StatefulSetAdmissionHook struct {
	lock        sync.RWMutex
	initialized bool
	stsAC       *statefulset.StatefulSetAdmissionControl
}
