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

package manager

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestQMParseVMs(t *testing.T) {
	data := `
perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
	LANGUAGE = (unset),
	LC_ALL = (unset),
	LC_CTYPE = "zh_CN.UTF-8",
	LANG = "en_US.UTF-8"
    are supported and installed on your system.
perl: warning: Falling back to a fallback locale ("en_US.UTF-8").
      VMID NAME                 STATUS     MEM(MB)    BOOTDISK(GB) PID
       105 to20190930-chenxiaojing stopped    49152            500.00 0
       106 to20201231-zhanghailong running    32768            500.00 7898
       107 to20201231-zhanghailong running    32768            500.00 7961
`

	g := NewGomegaWithT(t)
	vmManager := QMVMManager{}
	vms := vmManager.parserVMs(data)

	var expectedVMs []*VM
	expectedVMs = append(expectedVMs, &VM{
		Name:   "105",
		Status: "stopped",
	})
	expectedVMs = append(expectedVMs, &VM{
		Name:   "106",
		Status: "running",
	})
	expectedVMs = append(expectedVMs, &VM{
		Name:   "107",
		Status: "running",
	})
	g.Expect(vms).To(Equal(expectedVMs))
}
