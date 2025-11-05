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

//nolint:gosec
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"math"
	"net/url"
	"os"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/pingcap/tidb-operator/pkg/pdapi/v1"
	"github.com/pingcap/tidb-operator/pkg/ticdcapi/v1"
	"github.com/pingcap/tidb-operator/pkg/tikvapi/v1"
)

const (
	defaultRequestTimout = 10 * time.Second
	maxFailTimes         = 30
)

type Options struct {
	Mode            string
	PD              string
	TiKVStatusAddr  string
	TiCDCStatusAddr string

	TLS  bool
	CA   string
	Cert string
	Key  string
}

func (opt *Options) Parse() {
	flag.StringVar(&opt.Mode, "mode", "", "mode of prestop checker, [tikv, ticdc]")
	flag.StringVar(&opt.PD, "pd", "", "pd url")
	flag.StringVar(&opt.TiKVStatusAddr, "tikv-status-addr", "", "tikv status addr url")
	flag.StringVar(&opt.TiCDCStatusAddr, "ticdc-status-addr", "", "ticdc status addr url")

	flag.BoolVar(&opt.TLS, "tls", false, "enable tls or not")
	flag.StringVar(&opt.CA, "ca", "", "ca path")
	flag.StringVar(&opt.Cert, "cert", "", "cert path")
	flag.StringVar(&opt.Key, "key", "", "key path")
	flag.Parse()

	fmt.Println("options: ", opt)
}

const (
	ModeTiKV  = "tikv"
	ModeTiCDC = "ticdc"
)

type Config struct {
	Mode  string
	TiKV  *TiKVConfig
	TiCDC *TiCDCConfig
}

type TiKVConfig struct {
	PDClient   pdapi.PDClient
	Client     tikvapi.TiKVClient
	StatusAddr string
}

type TiCDCConfig struct {
	Client ticdcapi.TiCDCClient
}

func (opt *Options) Complete() (*Config, error) {
	c := Config{}

	var tlsCfg *tls.Config
	if opt.TLS {
		cfg, err := loadTLSConfig(opt.CA, opt.Cert, opt.Key)
		if err != nil {
			return nil, fmt.Errorf("cannot load tls config: %w", err)
		}
		tlsCfg = cfg
	}
	c.Mode = opt.Mode

	switch opt.Mode {
	case ModeTiKV:
		c.TiKV = &TiKVConfig{}
		u, err := url.Parse(opt.PD)
		if err != nil {
			return nil, fmt.Errorf("cannot recognize pd addr: %w", err)
		}

		if opt.TLS {
			u.Scheme = "https"
		} else {
			u.Scheme = "http"
		}

		var tikvURL string
		if opt.TLS {
			tikvURL = "https://" + opt.TiKVStatusAddr
		} else {
			tikvURL = "http://" + opt.TiKVStatusAddr
		}
		if _, err := url.Parse(tikvURL); err != nil {
			return nil, fmt.Errorf("cannot recognize tikv status addr: %w", err)
		}

		c.TiKV.PDClient = pdapi.NewPDClient(u.String(), defaultRequestTimout, tlsCfg)
		c.TiKV.Client = tikvapi.NewTiKVClient(tikvURL, defaultRequestTimout, tlsCfg, false)
		c.TiKV.StatusAddr = opt.TiKVStatusAddr

	case ModeTiCDC:
		c.TiCDC = &TiCDCConfig{}

		var ticdcURL string
		if opt.TLS {
			ticdcURL = "https://" + opt.TiCDCStatusAddr
		} else {
			ticdcURL = "http://" + opt.TiCDCStatusAddr
		}
		if _, err := url.Parse(ticdcURL); err != nil {
			return nil, fmt.Errorf("cannot recognize ticdc status addr: %w", err)
		}

		c.TiCDC.Client = ticdcapi.NewTiCDCClient(opt.TiCDCStatusAddr, ticdcapi.WithTLS(tlsCfg))

	default:
		return nil, fmt.Errorf("invalid mode: %s, now only support [tikv, ticdc]", opt.Mode)
	}

	return &c, nil
}

func RunTiKVPrestopHook(ctx context.Context, cfg *TiKVConfig) error {
	var storeID string

	//nolint:mnd // refactor to a constant if needed
	if err := wait.PollUntilContextTimeout(ctx, time.Second, time.Second*15, true,
		func(ctx context.Context) (done bool, err error) {
			info, err := cfg.PDClient.GetStores(ctx)
			if err != nil {
				fmt.Printf("cannot list stores, try again: %v\n", err)
				return false, nil
			}

			for _, s := range info.Stores {
				if s.Store == nil {
					continue
				}
				if s.Store.StatusAddress != cfg.StatusAddr {
					continue
				}

				storeID = strconv.FormatUint(s.Store.Id, 10)
				return true, nil
			}

			return false, nil
		},
	); err != nil {
		return fmt.Errorf("cannot find the store, arg addr is wrong or the store has been deleted")
	}

	fmt.Println("pre stop checking, store id:", storeID)

	minCount := math.MaxInt
	var failTimes int
	if err := wait.PollUntilContextCancel(ctx, time.Second, true, func(ctx context.Context) (done bool, err error) {
		leaderCount, err := cfg.Client.GetLeaderCount()
		if err != nil {
			fmt.Printf("cannot get leader count, try again: %v\n", err)
			return false, nil
		}
		if leaderCount == 0 {
			return true, nil
		}
		if leaderCount >= minCount {
			failTimes += 1
			// leader count doesn't desc in 10s, stop graceful shutdown
			if failTimes > maxFailTimes {
				return false, fmt.Errorf("leader count does not desc in 10 times, stop graceful shutdown")
			}
		} else {
			minCount = leaderCount
			failTimes = 0
		}

		fmt.Printf("pre stop checking, current leader count: %v\n", leaderCount)
		return false, nil
	}); err != nil {
		return fmt.Errorf("poll error: %w", err)
	}

	fmt.Printf("all leaders have been evicted\n")

	return nil
}

func RunTiCDCPrestopHook(ctx context.Context, cfg *TiCDCConfig) error {
	fmt.Println("run ticdc prestop hook")
	if err := wait.PollUntilContextCancel(ctx, time.Second, true, func(ctx context.Context) (done bool, err error) {
		fmt.Println("try to resign owner")
		resigned, err := cfg.Client.ResignOwner(ctx)
		if err != nil {
			return false, err
		}
		if !resigned {
			fmt.Println("wait for resigning owner")
			return false, nil
		}

		fmt.Println("try to drain capture")
		tableCount, err := cfg.Client.DrainCapture(ctx)
		if err != nil {
			return false, err
		}
		if tableCount > 0 {
			fmt.Printf("wait for draining capture, tableCount: %v\n", tableCount)
			return false, nil
		}

		return true, nil
	}); err != nil {
		return fmt.Errorf("poll error: %w", err)
	}

	fmt.Printf("prestop hook is successfully executed\n")
	return nil
}

func main() {
	opt := Options{}
	opt.Parse()

	cfg, err := opt.Complete()
	if err != nil {
		Fatalf("invalid config for prestop-checker: %v\n", err)
	}

	switch cfg.Mode {
	case ModeTiKV:
		if err := RunTiKVPrestopHook(context.Background(), cfg.TiKV); err != nil {
			Fatalf("failed to run tikv prestop hook: %v\n", err)
		}
	case ModeTiCDC:
		if err := RunTiCDCPrestopHook(context.Background(), cfg.TiCDC); err != nil {
			Fatalf("failed to run ticdc prestop hook: %v\n", err)
		}
	default:
		Fatalf("unreachable, invalid mode\n")
	}
}

func loadTLSConfig(ca, cert, key string) (*tls.Config, error) {
	rootCAs := x509.NewCertPool()
	caData, err := os.ReadFile(ca)
	if err != nil {
		return nil, err
	}

	if !rootCAs.AppendCertsFromPEM(caData) {
		return nil, fmt.Errorf("failed to append ca certs")
	}

	certData, err := os.ReadFile(cert)
	if err != nil {
		return nil, err
	}
	keyData, err := os.ReadFile(key)
	if err != nil {
		return nil, err
	}
	pair, err := tls.X509KeyPair(certData, keyData)
	if err != nil {
		return nil, err
	}

	//nolint:gosec // we didn't force to use a specific TLS version yet
	return &tls.Config{
		RootCAs:      rootCAs,
		ClientCAs:    rootCAs,
		Certificates: []tls.Certificate{pair},
	}, nil
}

func Fatalf(format string, args ...any) {
	fmt.Printf(format, args...)
	os.Exit(1)
}
