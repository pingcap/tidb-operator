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
// See the License for the specific language governing permissions and
// limitations under the License.

package compact

import (
	"context"
	"fmt"

	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/compact/options"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apis/util/config"
	pkgutil "github.com/pingcap/tidb-operator/pkg/backup/util"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
)

type Kubelink struct {
	kube kubernetes.Interface
	cr   versioned.Interface

	ref      *CompactionRef
	recorder record.EventRecorder
}

func NewKubelink(kubeconfig string) (*Kubelink, error) {
	kube, cr, err := util.NewKubeAndCRCli(kubeconfig)
	if err != nil {
		return nil, err
	}
	return &Kubelink{
		kube: kube,
		cr:   cr,
	}, nil
}

func (k *Kubelink) GetCompaction(ctx context.Context, opts CompactionRef) (options.CompactOpts, error) {
	if k.ref != nil {
		return options.CompactOpts{}, errors.New("GetCompaction called twice")
	}
	k.ref = &opts
	k.recorder = util.NewEventRecorder(k.kube, "compact.Kubelink")

	cb, err := k.cr.PingcapV1alpha1().
		CompactBackups(opts.Namespace).
		Get(ctx, opts.Name, v1.GetOptions{})
	if err != nil {
		return options.CompactOpts{}, err
	}

	out := options.CompactOpts{}
	args, err := pkgutil.GenStorageArgsForFlag(cb.Spec.StorageProvider, "")
	if err != nil {
		return options.CompactOpts{}, err
	}
	out.StorageOpts = args

	startTs, err := config.ParseTSString(cb.Spec.StartTs)
	if err != nil {
		return options.CompactOpts{}, errors.Annotatef(err, "failed to parse startTs %s", cb.Spec.StartTs)
	}
	endTs, err := config.ParseTSString(cb.Spec.EndTs)
	if err != nil {
		return options.CompactOpts{}, errors.Annotatef(err, "failed to parse endTs %s", cb.Spec.EndTs)
	}
	out.FromTS = startTs
	out.UntilTS = endTs

	out.Name = cb.ObjectMeta.Name
	out.Concurrency = uint64(cb.Spec.Concurrency)

	if err := out.Verify(); err != nil {
		return options.CompactOpts{}, err
	}

	return out, nil
}

type cOP func(*v1alpha1.CompactBackup) error

func (k *Kubelink) setState(newState string) cOP {
	return func(cb *v1alpha1.CompactBackup) error {
		cb.Status.State = newState
		return nil
	}
}

func (k *Kubelink) event(ty, reason, msg string) cOP {
	return func(cb *v1alpha1.CompactBackup) error {
		k.recorder.Event(cb, ty, reason, msg)
		return nil
	}
}

func (k *Kubelink) edit(ctx context.Context, extraOps ...cOP) error {
	lister := k.cr.PingcapV1alpha1().
		CompactBackups(k.ref.Namespace)
	cb, err := lister.
		Get(ctx, k.ref.Name, v1.GetOptions{})
	if err != nil {
		return err
	}

	for _, op := range extraOps {
		if err := op(cb); err != nil {
			return err
		}
	}

	_, err = lister.Update(ctx, cb, v1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (k *Kubelink) editOrWarn(ctx context.Context, extraOps ...cOP) {
	if err := k.edit(ctx, extraOps...); err != nil {
		klog.Warningf(
			"failed to edit state for %s/%s: %v",
			k.ref.Namespace,
			k.ref.Name,
			err,
		)
	}
}

func (k *Kubelink) OnStart(ctx context.Context) {
	k.editOrWarn(ctx, k.setState("STARTED"), k.event(corev1.EventTypeNormal, "Started", "CompactionStarted"))
}

func (k *Kubelink) OnProgress(ctx context.Context, p Progress) {
	message := fmt.Sprintf("RUNNING[READ_META(%d/%d),COMPACT_WORK(%d/%d)]",
		p.MetaCompleted, p.MetaTotal, p.BytesCompacted, p.BytesToCompact)
	k.editOrWarn(ctx, k.setState(message))
}

func (k *Kubelink) OnFinish(ctx context.Context, err error) {
	if err != nil {
		k.editOrWarn(ctx, k.setState(fmt.Sprintf("ERR[%s]", err)), k.event(corev1.EventTypeWarning, "Failed", err.Error()))
	} else {
		k.editOrWarn(ctx, k.setState("DONE"), k.event(corev1.EventTypeNormal, "Succeeded", "CompactionDone"))
	}
}
