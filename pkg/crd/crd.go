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

package crd

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/go-logr/logr"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	kyaml "sigs.k8s.io/yaml"

	"github.com/pingcap/tidb-operator/v2/pkg/client"
)

const (
	crdDirPath = "crd"

	versionAnnoKey = "pingcap.com/version"

	timeout = time.Second * 10
)

type Config struct {
	Client client.Client

	Version string

	AllowEmptyOldVersion bool
	IsDirty              bool

	CRDDir fs.FS
}

func ApplyCRDs(ctx context.Context, cfg *Config) error {
	entries, err := fs.ReadDir(cfg.CRDDir, "crd")
	if err != nil {
		return fmt.Errorf("failed to read crd dirs: %w", err)
	}

	var crds []string

	for _, file := range entries {
		if file.IsDir() {
			continue
		}

		fileName := file.Name()
		if !strings.HasSuffix(fileName, ".yaml") {
			continue
		}

		crdName, err := applyCRD(ctx, cfg, fileName)
		if err != nil {
			return err
		}

		crds = append(crds, crdName)
	}

	for _, crdName := range crds {
		if err := waitCRDEstablished(ctx, cfg.Client, crdName); err != nil {
			return fmt.Errorf("cannot wait crd established: %w", err)
		}
	}

	return nil
}

func applyCRD(ctx context.Context, cfg *Config, name string) (string, error) {
	logger := logr.FromContextOrDiscard(ctx)
	path := filepath.Join(crdDirPath, name)
	data, err := fs.ReadFile(cfg.CRDDir, path)
	if err != nil {
		return "", fmt.Errorf("cannot read file %s: %w", path, err)
	}

	var crd apiextensionsv1.CustomResourceDefinition
	if err := kyaml.Unmarshal(data, &crd); err != nil {
		return "", fmt.Errorf("cannot unmarshal yaml: %w", err)
	}

	oldCRD, err := getCurrentCRD(ctx, cfg.Client, crd.Name)
	if err != nil {
		return "", fmt.Errorf("cannot get crd %s: %w", crd.Name, err)
	}

	if oldCRD != nil {
		versionVal, ok := oldCRD.Annotations[versionAnnoKey]
		if !ok {
			if !cfg.AllowEmptyOldVersion {
				return "", fmt.Errorf("cannot find old version from crd %s, you can enable --allow-empty-old-version or apply manually", crd.Name)
			}
			logger.Info("warn: crd has no version", "crd", crd.Name)
		}

		if ok {
			changed, err := checkVersion(versionVal, cfg.Version, cfg.IsDirty)
			if err != nil {
				return "", err
			}

			if !changed {
				logger.Info("crd is up to date", "crd", crd.Name)
				return crd.Name, nil
			}
		}

		crd.ResourceVersion = oldCRD.ResourceVersion
	}

	logger.Info("try to apply crd", "crd", crd.Name)

	if crd.Annotations == nil {
		crd.Annotations = map[string]string{}
	}
	crd.Annotations[versionAnnoKey] = cfg.Version

	if err := cfg.Client.Apply(ctx, &crd); err != nil {
		return "", fmt.Errorf("cannot apply crd %s: %w", crd.Name, err)
	}

	return crd.Name, nil
}

func waitCRDEstablished(ctx context.Context, c client.Client, name string) error {
	crd := apiextensionsv1.CustomResourceDefinition{}
	if err := wait.PollUntilContextTimeout(ctx, time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		if err := c.Get(ctx, client.ObjectKey{
			Name: name,
		}, &crd); err != nil {
			return false, fmt.Errorf("can't get crd %s: %w", name, err)
		}

		for i := range crd.Status.Conditions {
			cond := &crd.Status.Conditions[i]
			if cond.Type != apiextensionsv1.Established {
				continue
			}
			if cond.Status == apiextensionsv1.ConditionTrue {
				return true, nil
			}
		}

		return false, nil
	}); err != nil {
		return fmt.Errorf("can't wait for crd %s established, error: %w", crd.Name, err)
	}

	return nil
}

func getCurrentCRD(ctx context.Context, c client.Client, crdName string) (*apiextensionsv1.CustomResourceDefinition, error) {
	crd := apiextensionsv1.CustomResourceDefinition{}
	if err := c.Get(ctx, client.ObjectKey{Name: crdName}, &crd); err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}
		return nil, nil
	}

	return &crd, nil
}

func checkVersion(o, n string, isDirty bool) (bool, error) {
	oldVersion, err := semver.NewVersion(o)
	if err != nil {
		return false, fmt.Errorf("old version %s is invalid: %w", o, err)
	}
	newVersion, err := semver.NewVersion(n)
	if err != nil {
		return false, fmt.Errorf("new version %s is invalid: %w", n, err)
	}
	// if version is not changed
	// - if dirty, assume there are changes
	if newVersion.Equal(oldVersion) {
		return isDirty, nil
	}

	if newVersion.LessThan(oldVersion) {
		return false, fmt.Errorf("cannot downgrade version from %s to %s", o, n)
	}

	return true, nil
}
