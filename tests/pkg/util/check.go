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
	"strconv"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

type WaitResult struct {
	Done bool
	Err  error
}

func WaitDone() WaitResult {
	return WaitResult{Done: true}
}

func WaitContinue(msg string, fs ...zap.Field) WaitResult {
	if len(msg) > 0 || len(fs) > 0 {
		zap.L().WithOptions(zap.AddCallerSkip(1)).Info(msg, fs...)
	}
	return WaitResult{}
}

func WaitErrBreak(err error, msg string, fs ...zap.Field) WaitResult {
	zap.L().WithOptions(zap.AddCallerSkip(1)).Error(msg, append(fs, zap.Error(err))...)
	return WaitResult{Err: err}
}

func WaitWrap(f func() WaitResult) wait.ConditionFunc {
	return func() (done bool, err error) {
		res := f()
		return res.Done, res.Err
	}
}

func Chain(cfs ...wait.ConditionFunc) wait.ConditionFunc {
	return func() (bool, error) {
		return ApplyChain(cfs...)
	}
}

func ApplyChain(cfs ...wait.ConditionFunc) (bool, error) {
	for _, cf := range cfs {
		if done, err := cf(); !done || err != nil {
			return done, err
		}
	}
	return true, nil
}

type MemberStatus struct {
	Replicas int
	Image    string
}

type ClusterStatus struct {
	PD   MemberStatus
	TiKV MemberStatus
	TiDB MemberStatus
}

func dotJoin(strs ...string) string { return strings.Join(strs, ".") }

func getClusterComponentTotalReplicas(tc Result, component string, failureMembers string) int {
	failures := len(tc.Get(dotJoin("status", component, failureMembers)).Map())
	return int(tc.Get(dotJoin("spec", component, "replicas")).Int()) + failures
}

func checkStatefulSet(ss Result, exp MemberStatus) wait.ConditionFunc {
	return WaitWrap(func() WaitResult {
		var set appsv1.StatefulSet

		if err := ss.Into(&set); err != nil {
			return WaitContinue("result is not a stateful set", zap.Error(err))
		}

		for _, path := range []string{
			"spec.replicas",
			"status.replicas",
			"status.readyReplicas",
		} {
			if actReplicas := int(ss.Get(path).Int()); actReplicas != exp.Replicas {
				return WaitContinue("check replicas",
					zap.Int("exp", exp.Replicas),
					zap.Int("act", actReplicas),
					zap.String("path", path),
					zap.String("type", "statefulset"),
				)
			}
		}

		cname := set.Labels[label.ComponentLabelKey]
		for i, c := range set.Spec.Template.Spec.Containers {
			if c.Name == cname && c.Image != exp.Image {
				return WaitContinue("check image",
					zap.String("exp", exp.Image),
					zap.String("act", c.Image),
					zap.String("path", "spec.template.spec.containers."+strconv.Itoa(i)+".image"),
					zap.String("type", "statefulset"),
				)
			}
		}

		return WaitDone()
	})
}

func checkClusterComponentStatus(tc Result, component string, members string, exp MemberStatus,
	checkMember func(path string, name string, result gjson.Result) WaitResult) wait.ConditionFunc {
	return WaitWrap(func() WaitResult {
		var cluster v1alpha1.TidbCluster

		if err := tc.Into(&cluster); err != nil {
			return WaitContinue("result is not a tidb cluster", zap.Error(err))
		}

		if path := dotJoin("status", component, "statefulSet"); !tc.Get(path).IsObject() {
			return WaitContinue("check stateful set",
				zap.String("path", path),
				zap.String("type", "tidbcluster"),
			)
		}

		failureMembers := "failure" + strings.Title(members)
		if totalReplicas := getClusterComponentTotalReplicas(tc, component, failureMembers); totalReplicas != exp.Replicas {
			return WaitContinue("check total replicas",
				zap.Int("exp", exp.Replicas),
				zap.Int("act", totalReplicas),
				zap.String("path1", dotJoin("spec", component, "replicas")),
				zap.String("path2", dotJoin("status", component, failureMembers)),
				zap.String("type", "tidbcluster"),
			)
		}

		memberPath := dotJoin("status", component, members)

		if actMembers := len(tc.Get(memberPath).Map()); actMembers != exp.Replicas {
			return WaitContinue("check total replicas",
				zap.Int("exp", exp.Replicas),
				zap.Int("act", actMembers),
				zap.String("path", memberPath),
				zap.String("type", "tidbcluster"),
			)
		}

		for n, r := range tc.Get(memberPath).Map() {
			res := checkMember(memberPath, n, r)
			if !res.Done {
				return res
			}
		}

		return WaitDone()
	})
}

func checkPDComponentStatus(tc Result, exp MemberStatus) wait.ConditionFunc {
	return checkClusterComponentStatus(tc, "pd", "members", exp,
		func(path string, name string, result gjson.Result) WaitResult {
			if !result.Get("health").Bool() {
				return WaitContinue("check member",
					zap.Bool("exp", true),
					zap.Bool("act", false),
					zap.String("path", dotJoin(path, name, "health")),
					zap.String("type", "tidbcluster"),
				)
			}
			return WaitDone()
		})
}

func checkTiKVComponentStatus(tc Result, exp MemberStatus) wait.ConditionFunc {
	return checkClusterComponentStatus(tc, "tikv", "stores", exp,
		func(path string, name string, result gjson.Result) WaitResult {
			if result.Get("state").String() != "Up" {
				return WaitContinue("check member",
					zap.String("exp", "Up"),
					zap.String("act", result.Get("state").String()),
					zap.String("path", dotJoin(path, name, "state")),
					zap.String("type", "tidbcluster"),
				)
			}
			return WaitDone()
		})
}

func checkTiDBComponentStatus(tc Result, exp MemberStatus) wait.ConditionFunc {
	return checkClusterComponentStatus(tc, "tidb", "members", exp,
		func(path string, name string, result gjson.Result) WaitResult {
			if !result.Get("health").Bool() {
				return WaitContinue("check member",
					zap.Bool("exp", true),
					zap.Bool("act", false),
					zap.String("path", dotJoin(path, name, "health")),
					zap.String("type", "tidbcluster"),
				)
			}
			return WaitDone()
		})
}

func CheckAllMembersReady(cli *GenericClient, ns string, name string, exp ClusterStatus) wait.ConditionFunc {
	return func() (bool, error) {
		tc := cli.GetTiDBByName(ns, name)
		ssPD := cli.GetAppsByName("statefulsets", ns, name+"-pd")
		ssTiKV := cli.GetAppsByName("statefulsets", ns, name+"-tikv")
		ssTiDB := cli.GetAppsByName("statefulsets", ns, name+"-tidb")
		return ApplyChain(
			checkPDComponentStatus(tc, exp.PD),
			checkStatefulSet(ssPD, exp.PD),

			checkTiKVComponentStatus(tc, exp.TiKV),
			checkStatefulSet(ssTiKV, exp.TiKV),

			checkTiDBComponentStatus(tc, exp.TiDB),
			checkStatefulSet(ssTiDB, exp.TiDB),

			WaitWrap(func() WaitResult {
				for _, name := range []string{
					name + "-pd",
					name + "-tidb",
					name + "-pd-peer",
					name + "-tikv-peer",
					name + "-tidb-peer",
				} {
					if svc := cli.GetCoreByName("services", ns, name); svc.Error() != nil {
						return WaitContinue("service is not available",
							zap.String("name", name),
							zap.Error(svc.Error()),
						)
					}
				}
				return WaitDone()
			}),
		)
	}
}
