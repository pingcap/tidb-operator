// Copyright 2023 PingCAP, Inc.
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
	"flag"
	"fmt"

	"github.com/pingcap/tidb-operator/http-service/version"
)

const (
	defaultAddr             = ":9080"
	defaultInternalGRPCAddr = "127.0.0.1:9081"
	defaultLogLevel         = "info"
)

type Config struct {
	flagSet *flag.FlagSet

	// printVersion is a flag to print version information.
	printVersion bool

	// LogFile is the path of the log file.
	LogFile string `toml:"log-file" json:"log-file"`
	// LogLevel is the log level.
	LogLevel string `toml:"log-level" json:"log-level"`

	// Addr is the address to listen on.
	Addr string `toml:"addr" json:"addr"`

	// InternalGRPCAddr is the address of internal grpc server.
	InternalGRPCAddr string `toml:"internal-grpc-addr" json:"internal-grpc-addr"`

	// Kubeconfig is the path to Kubeconfig.
	Kubeconfig string `toml:"kubeconfig" json:"kubeconfig"`
}

func NewConfig() *Config {
	cfg := &Config{
		flagSet:          flag.NewFlagSet("http-service", flag.ContinueOnError),
		LogLevel:         defaultLogLevel,
		Addr:             defaultAddr,
		InternalGRPCAddr: defaultInternalGRPCAddr,
	}

	cfg.flagSet.BoolVar(&cfg.printVersion, "V", false, "print version information and exit")
	cfg.flagSet.StringVar(&cfg.LogFile, "log-file", "", "log file path")
	cfg.flagSet.StringVar(&cfg.LogLevel, "L", cfg.LogLevel, "log level")
	cfg.flagSet.StringVar(&cfg.Addr, "addr", cfg.Addr, "address to listen on")
	cfg.flagSet.StringVar(&cfg.InternalGRPCAddr, "internal-grpc-addr", cfg.InternalGRPCAddr, "address of internal grpc server")
	cfg.flagSet.StringVar(&cfg.Kubeconfig, "kubeconfig", cfg.Kubeconfig, "path to kubeconfig")

	return cfg
}

func (c *Config) Parse(args []string) error {
	err := c.flagSet.Parse(args)
	if err != nil {
		return err
	}

	if c.printVersion {
		fmt.Println(version.GetRawInfo())
		return flag.ErrHelp
	}
	return nil
}
