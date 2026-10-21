// Copyright 2026 PingCAP, Inc.
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

package restore

import (
	"bytes"
	"encoding/json"
	"flag"
	"strings"
	"testing"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func TestRestoreControllerUpdateRestoreRedactsContext(t *testing.T) {
	const marker = "restore-context-log-probe"
	restore := &v1alpha1.Restore{
		ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: "test"},
		Spec: v1alpha1.RestoreSpec{StorageProvider: v1alpha1.StorageProvider{
			S3: &v1alpha1.S3StorageProvider{SSEKMSEncryptionContext: &v1alpha1.S3EncryptionContext{
				Values: map[string]string{"tenant": marker},
			}},
		}},
		// Return after logging without contacting a cluster or enqueueing work.
		Status: v1alpha1.RestoreStatus{Conditions: []v1alpha1.RestoreCondition{{
			Type: v1alpha1.RestoreInvalid, Status: corev1.ConditionTrue,
		}}},
	}
	applied, err := json.Marshal(restore)
	if err != nil {
		t.Fatal(err)
	}
	restore.Annotations = map[string]string{"kubectl.kubernetes.io/last-applied-configuration": string(applied)}
	state := klog.CaptureState()
	t.Cleanup(state.Restore)
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	klog.InitFlags(flags)
	if err := flags.Set("v", "4"); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	klog.LogToStderr(false)
	klog.SetOutput(&output)
	(&Controller{}).updateRestore(restore)
	klog.Flush()
	if !strings.Contains(output.String(), "restore-manager update") {
		t.Fatal("controller update log was not reached")
	}
	if strings.Contains(output.String(), marker) {
		t.Fatal("controller update log exposed context from last-applied annotation")
	}
}
