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

// Copyright 2018 PingCAP, Inc.
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
package metrics

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/onsi/gomega"
)

func TestAnnotationGetBody(t *testing.T) {
	tags := []string{"1", "2", "3"}

	options := AnnotationOptions{
		DashboardID: 1,
		PanelID:     2,
	}

	annotation := Annotation{
		Tags:                tags,
		TimestampInMilliSec: 1,
		Text:                "abc",
	}

	annotation.AnnotationOptions = options

	b, _ := annotation.getBody()
	re := make(map[string]interface{})
	json.Unmarshal(b, &re)

	g := gomega.NewGomegaWithT(t)
	g.Expect(fmt.Sprintf("%v", re["dashboardId"])).To(gomega.Equal(fmt.Sprintf("%v", 1)))
	g.Expect(re["text"]).To(gomega.Equal("abc"))
	g.Expect(re["time"]).To(gomega.Equal(float64(1)))
}
