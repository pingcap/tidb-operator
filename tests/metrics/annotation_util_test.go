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
	"github.com/onsi/gomega"
	"reflect"
	"testing"
	"time"
)

func TestAnnotationGetBody(t *testing.T) {
	tags := []string{"1", "2", "3"}
	annotation := Annotation{
		DashboardId:         1,
		PanelId:             2,
		Tags:                tags,
		TimestampInMilliSec: time.Now().Unix() * 1000,
		Text:                "abc",
	}

	b, _ := annotation.getBody()

	re := make(map[string]interface{})
	json.Unmarshal(b, &re)

	g := gomega.NewGomegaWithT(t)

	g.Expect(fmt.Sprintf("%v", re["DashboardId"])).To(gomega.Equal(fmt.Sprintf("%v", 1)))
	g.Expect(re["Text"]).To(gomega.Equal("abc"))
}

func TestErrorMetric(t *testing.T) {
	metric := initErrorMetric()
	metric.Inc()
	metric.Inc()

	g := gomega.NewGomegaWithT(t)

	v := reflect.ValueOf(metric).Elem()

	g.Expect(fmt.Sprintf("%v", v.FieldByName("valInt"))).To(gomega.Equal("2"))
}
