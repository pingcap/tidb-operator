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

package main

import (
	"database/sql"
	goflag "flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	// To register MySQL driver
	_ "github.com/go-sql-driver/mysql"

	"github.com/pingcap/tidb-operator/tests/pkg/blockwriter"
	"github.com/pingcap/tidb-operator/tests/pkg/util"
	flag "github.com/spf13/pflag"
	"k8s.io/component-base/logs"
	"k8s.io/klog"
)

var (
	optConfig      blockwriter.Config
	optNamespace   string
	optClusterName string
	optDatabase    string
	optPassword    string
)

func init() {
	flag.IntVar(&optConfig.TableNum, "table-num", optConfig.TableNum, "table num")
	flag.IntVar(&optConfig.Concurrency, "concurrency", optConfig.Concurrency, "concurrency")
	flag.IntVar(&optConfig.BatchSize, "batch-size", optConfig.BatchSize, "batch size")
	flag.IntVar(&optConfig.RawSize, "raw-size", optConfig.RawSize, "raw size")
	flag.StringVar(&optNamespace, "namespace", optNamespace, "namespace of the cluster")
	flag.StringVar(&optClusterName, "cluster-name", optClusterName, "cluster name")
	flag.StringVar(&optDatabase, "database", optDatabase, "database")
	flag.StringVar(&optPassword, "password", optPassword, "password")
}

func getDSN(ns, tcName, databaseName, password string) string {
	return fmt.Sprintf("root:%s@(%s-tidb.%s:4000)/%s?charset=utf8", password, tcName, ns, databaseName)
}

func main() {
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	flag.Parse()
	logs.InitLogs()
	defer logs.FlushLogs()

	err := initDB()
	if err != nil {
		klog.Fatal(err)
	}
	dsn := getDSN(optNamespace, optClusterName, optDatabase, optPassword)
	db, err := util.OpenDB(dsn, optConfig.Concurrency)
	if err != nil {
		klog.Fatal(err)
	}

	writer := blockwriter.NewBlockWriterCase(optConfig)
	writer.ClusterName = optClusterName
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	go writer.Start(db)

	sig := <-signalCh
	klog.Infof("signal %v received, stopping blockwriter", sig)
	writer.Stop()
}

func initDB() error {
	s := fmt.Sprintf("root:%s@(%s-tidb.%s:4000)/?charset=utf8", optPassword, optClusterName, optNamespace)
	db, err := sql.Open("mysql", s)
	if err != nil {
		return err
	}
	defer db.Close()
	exec := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", optDatabase)
	_, err = db.Exec(exec)
	if err != nil {
		return err
	}
	return nil
}
