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

package volumes

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
)

func Test_areStringsDifferent(t *testing.T) {
	tests := []struct {
		name string
		pre  *string
		cur  *string
		want bool
	}{
		{
			name: "both nil",
			pre:  nil,
			cur:  nil,
			want: false,
		},
		{
			name: "both same non-nil",
			pre:  ptr.To("same"),
			cur:  ptr.To("same"),
			want: false,
		},
		{
			name: "from nil to non-nil",
			pre:  nil,
			cur:  ptr.To("non-nil"),
			want: true,
		},
		{
			name: "from non-nil to nil",
			pre:  ptr.To("non-nil"),
			cur:  nil,
			want: true,
		},
		{
			name: "different non-nil",
			pre:  ptr.To("pre"),
			cur:  ptr.To("cur"),
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := areStringsDifferent(tt.pre, tt.cur); got != tt.want {
				t.Errorf("areStringsDifferent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSyncPVCs(t *testing.T) {
	tests := []struct {
		name         string
		existingObjs []client.Object
		expectPVCs   []*corev1.PersistentVolumeClaim
		setup        func(*MockModifier)
		expectFunc   func(*WithT, client.Client)
		wantErr      bool
	}{
		{
			name: "create PVC",
			expectPVCs: []*corev1.PersistentVolumeClaim{
				fake.FakeObj[corev1.PersistentVolumeClaim]("pvc-0"),
			},
		},
		{
			name: "pvc not bound",
			existingObjs: []client.Object{
				fake.FakeObj[corev1.PersistentVolumeClaim]("pvc-0"),
			},
			expectPVCs: []*corev1.PersistentVolumeClaim{
				fake.FakeObj("pvc-0", fake.Label[corev1.PersistentVolumeClaim]("foo", "bar")),
			},
			expectFunc: func(g *WithT, cli client.Client) {
				var pvc corev1.PersistentVolumeClaim
				g.Expect(cli.Get(context.TODO(), client.ObjectKey{Name: "pvc-0"}, &pvc)).To(Succeed())
				g.Expect(pvc.Labels).To(BeEmpty())
			},
		},
		{
			name: "did not change PVC",
			existingObjs: []client.Object{
				fake.FakeObj("pvc-0", func(obj *corev1.PersistentVolumeClaim) *corev1.PersistentVolumeClaim {
					obj.Status.Phase = corev1.ClaimBound
					return obj
				}),
			},
			expectPVCs: []*corev1.PersistentVolumeClaim{
				fake.FakeObj("pvc-0", fake.Label[corev1.PersistentVolumeClaim]("foo", "bar")),
			},
			setup: func(vm *MockModifier) {
				vm.EXPECT().GetActualVolume(gomock.Any(), gomock.Any(), gomock.Any()).Return(&ActualVolume{
					PVC:     fake.FakeObj[corev1.PersistentVolumeClaim]("pvc-0"),
					Desired: &DesiredVolume{},
				}, nil)
				vm.EXPECT().ShouldModify(gomock.Any(), gomock.Any()).Return(false)
			},
			expectFunc: func(g *WithT, cli client.Client) {
				var pvc corev1.PersistentVolumeClaim
				g.Expect(cli.Get(context.TODO(), client.ObjectKey{Name: "pvc-0"}, &pvc)).To(Succeed())
				g.Expect(pvc.Labels).To(HaveKeyWithValue("foo", "bar"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			vm := NewMockModifier(ctrl)
			if tt.setup != nil {
				tt.setup(vm)
			}

			cli := client.NewFakeClient(tt.existingObjs...)
			if _, err := SyncPVCs(context.TODO(), cli, tt.expectPVCs, vm, logr.Discard()); (err != nil) != tt.wantErr {
				t.Errorf("SyncPVCs() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectFunc != nil {
				tt.expectFunc(NewGomegaWithT(t), cli)
			}
		})
	}
}

func TestUpgradeRevision(t *testing.T) {
	// no revision before
	pvc := fake.FakeObj[corev1.PersistentVolumeClaim]("pvc-0")
	upgradeRevision(pvc)
	if pvc.Annotations[annoKeyPVCSpecRevision] != "1" {
		t.Errorf("expect revision 1, got %s", pvc.Annotations[annoKeyPVCSpecRevision])
	}

	// has revision before
	upgradeRevision(pvc)
	if pvc.Annotations[annoKeyPVCSpecRevision] != "2" {
		t.Errorf("expect revision 2, got %s", pvc.Annotations[annoKeyPVCSpecRevision])
	}
}
