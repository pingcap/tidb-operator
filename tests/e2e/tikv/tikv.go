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

package tikv

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apicall"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/tests/e2e/data"
	"github.com/pingcap/tidb-operator/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/tests/e2e/label"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/cert"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/k8s"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/tidb"
)

var _ = ginkgo.Describe("TiKV", label.TiKV, func() {
	f := framework.New()
	f.Setup()

	ginkgo.DescribeTableSubtree("Leader Eviction", label.P1,
		func(tls bool) {
			if tls {
				f.SetupCluster(data.WithClusterTLS())
			}

			// NOTE(liubo02): this case is failed in e2e env because of the cgroup v2.
			// Enable it if env is fixed.
			ginkgo.PIt("leader evicted when delete tikv pod directly", func(ctx context.Context) {
				if tls {
					ns := f.Cluster.Namespace
					cn := f.Cluster.Name
					f.Must(cert.InstallTiDBIssuer(ctx, f.Client, ns, cn))
					f.Must(cert.InstallTiDBCertificates(ctx, f.Client, ns, cn, "dbg"))
					f.Must(cert.InstallTiDBComponentsCertificates(ctx, f.Client, ns, cn, "pdg", "kvg", "dbg", "flashg", "cdcg"))
				}
				pdg := f.MustCreatePD(ctx)
				kvg := f.MustCreateTiKV(ctx,
					data.WithReplicas[*runtime.TiKVGroup](3),
				)

				f.WaitForPDGroupReady(ctx, pdg)
				f.WaitForTiKVGroupReady(ctx, kvg)

				kvs, err := apicall.ListInstances[scope.TiKVGroup](ctx, f.Client, kvg)
				f.Must(err)

				kv := kvs[0]

				nctx, cancel := context.WithCancel(ctx)
				ch := make(chan struct{})
				go func() {
					defer close(ch)
					defer ginkgo.GinkgoRecover()
					f.WaitTiKVPreStopHookSuccess(nctx, kv)
				}()

				f.RestartTiKVPod(ctx, kv)

				cancel()
				<-ch
			})

			// If TLS is enabled, this test is not applicable because it relies on the PD API in this case.
			if !tls {
				ginkgo.It("should wait for leader count > 0 to restart the next TiKV", func(ctx context.Context) {
					pdg := f.MustCreatePD(ctx)
					kvg := f.MustCreateTiKV(ctx, data.WithReplicas[*runtime.TiKVGroup](3))
					dbg := f.MustCreateTiDB(ctx)
					f.WaitForPDGroupReady(ctx, pdg)
					f.WaitForTiKVGroupReady(ctx, kvg)
					f.WaitForTiDBGroupReady(ctx, dbg)

					// Import data to ensure enough leaders
					ns := f.Namespace.Name
					svcName := data.DefaultTiDBServiceName
					dsn, cancel, err := tidb.PortForwardAndGetTiDBDSN(f.PortForwarder, ns, svcName, "root", "", "test", "charset=utf8mb4")
					f.Must(err)
					defer cancel()
					db, err := sql.Open("mysql", dsn)
					f.Must(err)
					defer db.Close()
					_, err = db.Exec("CREATE DATABASE IF NOT EXISTS e2e_test")
					f.Must(err)
					_, err = db.Exec("USE e2e_test")
					f.Must(err)
					_, err = db.Exec("CREATE TABLE t1 (id INT PRIMARY KEY AUTO_INCREMENT, v VARCHAR(1000))")
					f.Must(err)
					batchSize := 1000
					total := 500000
					for i := 0; i < total; i += batchSize {
						end := i + batchSize
						if end > total {
							end = total
						}
						query := "INSERT INTO t1 (v) VALUES "
						args := make([]interface{}, 0, end-i)
						for j := i; j < end; j++ {
							if j > i {
								query += ","
							}
							query += "(?)"
							args = append(args, strings.Repeat("x", 900)+fmt.Sprintf("%d", j))
						}
						_, err = db.Exec(query, args...)
						f.Must(err)
					}
					_, err = db.Exec("SPLIT TABLE t1 BETWEEN (0) AND (500000) REGIONS 350")
					f.Must(err)

					// Wait until leader count is sufficient
					pdAddr, pdCancel := getFirstPDAddr(ctx, f, ns)
					defer pdCancel()
					ginkgo.By(fmt.Sprintf("Wait until leader count is sufficient, PD address: %s", pdAddr))
					gomega.Eventually(func(g gomega.Gomega) {
						count, err := getTotalLeaderCount(pdAddr)
						g.Expect(err).ToNot(gomega.HaveOccurred())
						g.Expect(count).To(gomega.BeNumerically(">", 300))
					}, 2*time.Minute, 5*time.Second).Should(gomega.Succeed())

					// Define record structs
					type LeaderRecord struct {
						Time        time.Time
						LeaderCount int
					}
					type RestartRecord struct {
						Time    time.Time
						PodName string
						StoreID string
					}

					var (
						leaderRecordsMu  sync.Mutex
						leaderRecords    = make(map[string][]LeaderRecord) // StoreID (string) -> records
						stopLeaderPoll   = make(chan struct{})
						restartRecordsMu sync.Mutex
						restartRecords   []RestartRecord
						stopRestartPoll  = make(chan struct{})
					)

					// Start a goroutine to poll leader count
					go func() {
						ticker := time.NewTicker(1 * time.Second)
						defer ticker.Stop()
						for {
							select {
							case <-stopLeaderPoll:
								return
							case <-ticker.C:
								stores, err := getAllStoresInfo(pdAddr)
								if err != nil {
									// Optionally log the error, e.g., ginkgo.GinkgoWriter.Printf("Error getting all stores info: %v\n", err)
									continue
								}
								leaderRecordsMu.Lock()
								now := time.Now()
								for _, store := range stores {
									// store.ID from getAllStoresInfo is already a string
									leaderRecords[store.ID] = append(leaderRecords[store.ID], LeaderRecord{
										Time:        now,
										LeaderCount: store.LeaderCount,
									})
								}
								leaderRecordsMu.Unlock()
							}
						}
					}()

					// Get old pod UIDs
					labelSelector := map[string]string{
						v1alpha1.LabelKeyCluster:   kvg.Spec.Cluster.Name,
						v1alpha1.LabelKeyGroup:     kvg.Name,
						v1alpha1.LabelKeyComponent: "tikv",
					}
					var oldPods corev1.PodList
					f.Must(f.Client.List(ctx, &oldPods, client.InNamespace(kvg.Namespace), client.MatchingLabels(labelSelector)))
					oldUIDs := make(map[string]struct{}, len(oldPods.Items))
					for _, pod := range oldPods.Items {
						oldUIDs[string(pod.UID)] = struct{}{}
					}

					// Start a goroutine to record pod restarts
					go func() {
						for {
							select {
							case <-stopRestartPoll:
								return
							default:
								var podList corev1.PodList
								f.Must(f.Client.List(ctx, &podList, client.InNamespace(kvg.Namespace), client.MatchingLabels(labelSelector)))
								for _, pod := range podList.Items {
									if _, exist := oldUIDs[string(pod.UID)]; exist {
										continue
									}
									if pod.Status.Phase == corev1.PodRunning {
										ready := false
										for _, cond := range pod.Status.Conditions {
											if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
												ready = true
												break
											}
										}
										if ready {
											storeID := pod.Labels[v1alpha1.LabelKeyStoreID]
											restartRecordsMu.Lock()
											already := false
											for _, r := range restartRecords {
												if r.PodName == pod.Name && r.StoreID == storeID {
													already = true
													break
												}
											}
											if !already {
												restartRecords = append(restartRecords, RestartRecord{
													Time:    time.Now(),
													PodName: pod.Name,
													StoreID: storeID,
												})
											}
											restartRecordsMu.Unlock()
										}
									}
								}
								time.Sleep(1 * time.Second)
							}
						}
					}()

					// Trigger rolling restart by updating TiKV config
					f.Must(updateTiKVConfig(ctx, f, kvg, "log.level = 'warn'"))

					// Wait for all pods to be restarted
					for {
						restartRecordsMu.Lock()
						done := len(restartRecords) >= int(*kvg.Spec.Replicas)
						restartRecordsMu.Unlock()
						if done {
							break
						}
						time.Sleep(2 * time.Second)
					}
					close(stopRestartPoll)
					close(stopLeaderPoll)

					// Analyze: for each TiKV restart, check leader recovery before next restart
					for i, restart := range restartRecords {
						leaderRecordsMu.Lock()
						records := leaderRecords[restart.StoreID]
						leaderRecordsMu.Unlock()
						var leaderRecoveredAt time.Time
						for _, rec := range records {
							if rec.Time.After(restart.Time) && rec.LeaderCount > 0 {
								leaderRecoveredAt = rec.Time
								break
							}
						}
						var nextRestartAt time.Time
						if i+1 < len(restartRecords) {
							nextRestartAt = restartRecords[i+1].Time
						} else {
							// If this is the last one, no need to check leader recovery
							break
						}
						ginkgo.GinkgoWriter.Printf("TiKV %s restarted at %v, leader recovered at %v, next restart at %v\n",
							restart.PodName, restart.Time, leaderRecoveredAt, nextRestartAt)
						gomega.Expect(leaderRecoveredAt).ToNot(gomega.BeZero(), "TiKV %s never recovered leader", restart.PodName)
						gomega.Expect(leaderRecoveredAt).To(gomega.BeTemporally("<", nextRestartAt),
							"TiKV %s leader did not recover before next restart", restart.PodName)
					}
				})
			}
		},
		func(tls bool) string {
			if tls {
				return "TLS"
			}
			return "NO TLS"
		},
		ginkgo.Entry(nil, false),
		ginkgo.Entry(nil, label.FeatureTLS, true),
	)
})

// Helper struct for store information from PD
type StoreDetail struct {
	ID          string
	LeaderCount int
}

// getAllStoresInfo queries PD API to get details for all stores.
// It returns a list of StoreDetail, where ID is the string representation of the store ID.
func getAllStoresInfo(pdAddr string) ([]StoreDetail, error) {
	url := fmt.Sprintf("http://%s/pd/api/v1/stores", pdAddr)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Stores []struct {
			Store struct {
				ID uint64 `json:"id"`
			} `json:"store"`
			Status struct {
				LeaderCount int `json:"leader_count"`
			} `json:"status"`
		} `json:"stores"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var details []StoreDetail
	for _, s := range result.Stores {
		details = append(details, StoreDetail{
			ID:          fmt.Sprintf("%d", s.Store.ID), // Convert numeric ID to string
			LeaderCount: s.Status.LeaderCount,
		})
	}
	return details, nil
}

// getTotalLeaderCount queries PD API to get the total leader count.
func getTotalLeaderCount(pdAddr string) (int, error) {
	stores, err := getAllStoresInfo(pdAddr)
	if err != nil {
		return 0, err
	}

	var total int
	for _, store := range stores {
		total += store.LeaderCount
	}
	return total, nil
}

// getFirstPDAddr returns the local port-forwarded address for PD API.
func getFirstPDAddr(ctx context.Context, f *framework.Framework, ns string) (string, context.CancelFunc) {
	localHost, localPort, cancel, err := k8s.ForwardOnePort(
		f.PortForwarder,
		ns,
		"svc/pdg-pd", // PD service name
		2379,
	)
	f.Must(err)
	return fmt.Sprintf("%s:%d", localHost, localPort), cancel
}

// updateTiKVConfig updates the config to trigger rolling restart.
func updateTiKVConfig(ctx context.Context, f *framework.Framework, kvg *v1alpha1.TiKVGroup, config string) error {
	var kvgGet v1alpha1.TiKVGroup
	if err := f.Client.Get(ctx, client.ObjectKeyFromObject(kvg), &kvgGet); err != nil {
		return err
	}
	kvgGet.Spec.Template.Spec.Config = v1alpha1.ConfigFile(config)
	return f.Client.Update(ctx, &kvgGet)
}
