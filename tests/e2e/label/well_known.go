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

package label

import (
	"github.com/onsi/ginkgo/v2"

	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
)

var (
	// Components
	Cluster = ginkgo.Label("c:Cluster")
	PD      = ginkgo.Label("c:PD")
	TiDB    = ginkgo.Label("c:TiDB")
	TiKV    = ginkgo.Label("c:TiKV")
	TiFlash = ginkgo.Label("c:TiFlash")
	TiCDC   = ginkgo.Label("c:TiCDC")
	TiProxy = ginkgo.Label("c:TiProxy")

	// Priority
	P0 = ginkgo.Label("P0")
	P1 = ginkgo.Label("P1")
	P2 = ginkgo.Label("P2")

	// Operations
	Update  = ginkgo.Label("op:Update")
	Scale   = ginkgo.Label("op:Scale")
	Suspend = ginkgo.Label("op:Suspend")
	Delete  = ginkgo.Label("op:Delete")

	// Env
	MultipleAZ = ginkgo.Label("env:MultipleAZ")

	// Feature
	// TODO(liubo02): prefix 'f:' will be used for feature gates,
	// rename these features labels
	FeatureTLS          = ginkgo.Label("f:TLS")
	FeatureAuthToken    = ginkgo.Label("f:AuthToken")
	FeatureBootstrapSQL = ginkgo.Label("f:BootstrapSQL")
	FeatureHotReload    = ginkgo.Label("f:HotReload")
	FeaturePDMS         = ginkgo.Label("f:PDMS")

	// Mode
	ModeDisaggregatedTiFlash = ginkgo.Label("m:DisaggregatedTiFlash")

	// Overlay
	OverlayEphemeralVolume = ginkgo.Label("o:EphemeralVolume")

	// Kind
	//
	// KindExample are tests for example dir
	KindExample = ginkgo.Label("k:Example")
	// KindBasic are basic tests
	KindBasic = ginkgo.Label("k:Basic")
	// KindBR are tests for br
	KindBR = ginkgo.Label("k:BR")
	// KindAvail are tests to test availablity
	KindAvail = ginkgo.Label("k:Avail")
	// KindNextGen are tests to test next-gen
	KindNextGen = ginkgo.Label("k:NextGen")
)

func Features(fs ...metav1alpha1.Feature) ginkgo.Labels {
	var ls []string
	for _, f := range fs {
		ls = append(ls, "f:"+string(f))
	}

	return ginkgo.Label(ls...)
}
