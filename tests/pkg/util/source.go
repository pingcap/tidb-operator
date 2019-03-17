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

package util

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/juju/errors"
)

type OperatorSource interface {
	Version() string
	BasePath() string
	ChartPath(name string) string
	Fetch(force bool) error
}

type operatorSource struct {
	base string
	ref  string
}

var _ = OperatorSource(operatorSource{})

func NewOperatorSource(version string) OperatorSource {
	return operatorSource{base: "/tmp/tidb-operator", ref: version}
}

func (s operatorSource) Version() string {
	return s.ref
}

func (s operatorSource) BasePath() string {
	return filepath.Join(s.base, s.ref)
}

func (s operatorSource) ChartPath(name string) string {
	return filepath.Join(s.BasePath(), "charts", name)
}

func (s operatorSource) Fetch(force bool) error {
	if _, err := os.Stat(s.BasePath()); !force && !os.IsNotExist(err) {
		return nil
	}

	if err := os.MkdirAll(s.BasePath(), 0755); err != nil {
		return errors.Trace(err)
	}

	cmd := exec.Command("sh", "-c", "curl -sL 'https://api.github.com/repos/pingcap/tidb-operator/tarball/"+s.ref+"' | tar -zx --strip-components=1 -C "+s.BasePath())

	if out, err := cmd.CombinedOutput(); err != nil {
		return errors.New(string(out))
	}
	return nil
}
