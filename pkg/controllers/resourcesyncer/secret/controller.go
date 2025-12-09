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

package secret

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"
	"github.com/spf13/afero"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/pingcap/tidb-operator/v2/pkg/utils/k8s"
)

const (
	resourceVersionFile = "resourceVersion"

	maxRetryTimes  = 10
	defaultTimeout = time.Second * 30

	defaultFilePerm fs.FileMode = 0o644
	defaultDirPerm  fs.FileMode = 0o755
)

type Reconciler struct {
	Client    client.Client
	Namespace string
	Labels    map[string]string
	BaseDirFS afero.Fs

	// secret name to it's resourceVersion
	cache map[string]string
	lock  sync.Mutex

	hasSynced atomic.Bool
}

func Setup(
	mgr manager.Manager,
	dir string,
	ns string,
	labels map[string]string,
) error {
	r := &Reconciler{
		Client:    mgr.GetClient(),
		BaseDirFS: afero.NewBasePathFs(afero.NewOsFs(), dir),
		Namespace: ns,
		Labels:    labels,
		cache:     map[string]string{},
	}
	if err := mgr.AddHealthzCheck("synced", r.Checker()); err != nil {
		return fmt.Errorf("cannot add synced for healthz checker: %w", err)
	}
	if err := mgr.AddReadyzCheck("synced", r.Checker()); err != nil {
		return fmt.Errorf("cannot add synced for readyz checker: %w", err)
	}
	go func() {
		// if cache is not synced, mgr will quit, so wait infinite here
		ctx := context.Background()
		logger := mgr.GetLogger()
		if mgr.GetCache().WaitForCacheSync(ctx) {
			sl := corev1.SecretList{}
			failed := true
			for range maxRetryTimes {
				// has synced, no need to list secrets again
				if r.hasSynced.Load() {
					return
				}

				nctx, cancel := context.WithTimeout(ctx, defaultTimeout)

				if err := r.Client.List(
					nctx,
					&sl,
					client.InNamespace(r.Namespace),
					client.MatchingLabels(r.Labels),
				); err != nil {
					logger.Error(err, "cannot list secrets", "namepsace", r.Namespace, "labels", r.Labels)
				} else {
					failed = false
					cancel()
					break
				}
				cancel()
			}

			if !failed && len(sl.Items) == 0 {
				if r.hasSynced.CompareAndSwap(false, true) {
					logger.Info("no secrets, hasSynced set to true")
				}

				return
			}
		}
	}()
	return ctrl.NewControllerManagedBy(mgr).For(&corev1.Secret{}).
		WithOptions(controller.Options{RateLimiter: k8s.RateLimiter}).
		Complete(r)
}

func (r *Reconciler) Reconcile(ctx context.Context, _ ctrl.Request) (ctrl.Result, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	logger := logr.FromContextOrDiscard(ctx)
	startTime := time.Now()
	logger.Info("start reconcile")
	defer func() {
		dur := time.Since(startTime)
		logger.Info("end reconcile", "duration", dur)
	}()

	if err := r.sync(ctx); err != nil {
		return ctrl.Result{}, err
	}

	if r.hasSynced.CompareAndSwap(false, true) {
		logger.Info("sync at least once")
	}

	return ctrl.Result{}, nil
}

func (r *Reconciler) sync(ctx context.Context) error {
	logger := logr.FromContextOrDiscard(ctx)
	sl := corev1.SecretList{}
	if err := r.Client.List(
		ctx,
		&sl,
		client.InNamespace(r.Namespace),
		client.MatchingLabels(r.Labels),
	); err != nil {
		return fmt.Errorf("cannot list secrets with labels %v in %s: %w", r.Labels, r.Namespace, err)
	}
	logger.Info("list secrets", "total", len(sl.Items))

	var errs []error
	dirs := map[string]struct{}{}

	for i := range sl.Items {
		s := &sl.Items[i]
		if err := r.syncSecret(ctx, s); err != nil {
			errs = append(errs, err)
		}

		dirs[s.Name] = struct{}{}
	}
	if err := cleanupDir(ctx, r.BaseDirFS, "", dirs); err != nil {
		errs = append(errs, err)
	}

	return errors.NewAggregate(errs)
}

func (r *Reconciler) syncSecret(ctx context.Context, s *corev1.Secret) error {
	logger := logr.FromContextOrDiscard(ctx)

	rv, ok := r.cache[s.Name]
	if ok && rv == s.ResourceVersion {
		logger.Info("hit cache, secret has synced", "secret", s.Name, "resourceVersion", s.ResourceVersion)
		return nil
	}

	hasDir := true

	info, err := r.BaseDirFS.Stat(s.Name)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		if err := r.BaseDirFS.Mkdir(s.Name, defaultDirPerm); err != nil {
			return err
		}

		hasDir = false
	} else if !info.IsDir() {
		return fmt.Errorf("%s is not dir", s.Name)
	}

	if hasDir {
		isChanged, err := r.isChanged(s)
		if err != nil {
			return err
		}

		if !isChanged {
			// has synced, just return
			logger.Info("secret has synced", "secret", s.Name, "resourceVersion", s.ResourceVersion)
			return nil
		}
	}

	var errs []error
	files := map[string]struct{}{}

	for name, data := range s.Data {
		logger.Info("sync secret file", "secret", s.Name, "resourceVersion", s.ResourceVersion, "file", name)
		if err := r.syncSecretFile(s.Name, name, data); err != nil {
			errs = append(errs, err)
		}
		files[name] = struct{}{}
	}
	if err := cleanupDir(ctx, r.BaseDirFS, s.Name, files); err != nil {
		errs = append(errs, err)
	}
	if len(errs) != 0 {
		return errors.NewAggregate(errs)
	}

	if err := afero.WriteFile(
		r.BaseDirFS,
		filepath.Join(s.Name, resourceVersionFile),
		[]byte(s.ResourceVersion),
		defaultFilePerm,
	); err != nil {
		return err
	}

	r.cache[s.Name] = s.ResourceVersion
	return nil
}

func (r *Reconciler) syncSecretFile(parent, name string, data []byte) error {
	return afero.WriteFile(r.BaseDirFS, filepath.Join(parent, name), data, defaultFilePerm)
}

func (r *Reconciler) isChanged(s *corev1.Secret) (bool, error) {
	data, err := afero.ReadFile(r.BaseDirFS, filepath.Join(s.Name, resourceVersionFile))
	if err != nil {
		// if rv doesn't exist, assume that secret is changed
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, fmt.Errorf("cannot read resourceVersion file: %w", err)
	}

	return s.ResourceVersion != string(data), nil
}

func (r *Reconciler) Checker() healthz.Checker {
	return func(req *http.Request) error {
		if r.hasSynced.Load() {
			return nil
		}
		return fmt.Errorf("waiting for at least once sync")
	}
}

func cleanupDir(ctx context.Context, base afero.Fs, dir string, keep map[string]struct{}) error {
	logger := logr.FromContextOrDiscard(ctx)
	entries, err := afero.ReadDir(base, dir)
	if err != nil {
		return fmt.Errorf("failed to read dir %s: %w", dir, err)
	}
	var errs []error
	for _, entry := range entries {
		if _, ok := keep[entry.Name()]; ok {
			continue
		}
		logger.Info("clean up", "path", filepath.Join(dir, entry.Name()))
		if err := base.RemoveAll(filepath.Join(dir, entry.Name())); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.NewAggregate(errs)
}

func EnsureDirExists(base afero.Fs, dir string) error {
	info, err := base.Stat(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := base.MkdirAll(dir, defaultDirPerm); err != nil {
			return err
		}

		return nil
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not dir", dir)
	}

	return nil
}
