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

package util

import (
	"fmt"
	"strings"

	br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
)

// ValidateBROptions prevents data writes from diverging from the destination used by cleanup.
func ValidateBROptions(config *br.BRConfig) error {
	if config == nil {
		return nil
	}
	for _, option := range config.Options {
		if option == "--" {
			return fmt.Errorf("spec.br.options cannot terminate flag parsing before the managed storage arguments")
		}
		flag, _, _ := strings.Cut(option, "=")
		switch flag {
		case "--storage", "--full-backup-storage", "--s3.endpoint", "--s3.region", "--s3.provider",
			"--gcs.endpoint", "--azblob.endpoint", "--azblob.account-name":
			return fmt.Errorf("spec.br.options cannot override managed storage flag %s; configure the storage provider", flag)
		}
	}
	return nil
}
