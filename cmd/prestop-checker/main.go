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

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/pingcap/tidb-operator/pkg/pdapi/v1"
	"github.com/pingcap/tidb-operator/pkg/tikvapi/v1"
)

const (
	defaultRequestTimout = 10 * time.Second
)

type Options struct {
	Mode           string
	PD             string
	TiKVStatusAddr string

	TLS  bool
	CA   string
	Cert string
	Key  string
}

func (opt *Options) Parse() {
	flag.StringVar(&opt.Mode, "mode", "", "mode prestop checker, now support tikv only")
	flag.StringVar(&opt.PD, "pd", "", "pd url")
	flag.StringVar(&opt.TiKVStatusAddr, "tikv-status-addr", "", "tikv status addr url")

	flag.BoolVar(&opt.TLS, "tls", false, "enable tls or not")
	flag.StringVar(&opt.CA, "ca", "", "ca path")
	flag.StringVar(&opt.Cert, "cert", "", "cert path")
	flag.StringVar(&opt.Key, "key", "", "key path")
	flag.Parse()

	fmt.Println("options: ", opt)
}

type Config struct {
	PDClient   pdapi.PDClient
	TiKVClient tikvapi.TiKVClient

	TiKVStatusAddr string
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

	u, err := url.Parse(opt.PD)
	if err != nil {
		return nil, fmt.Errorf("cannot recognize pd addr: %w", err)
	}

	var tikvURL string

	if opt.TLS {
		tikvURL = "https://" + opt.TiKVStatusAddr
		u.Scheme = "https"
	} else {
		tikvURL = "http://" + opt.TiKVStatusAddr
		u.Scheme = "http"
	}

	if _, err := url.Parse(tikvURL); err != nil {
		return nil, fmt.Errorf("cannot recognize tikv status addr: %w", err)
	}

	c.PDClient = pdapi.NewPDClient(u.String(), defaultRequestTimout, tlsCfg)
	c.TiKVClient = tikvapi.NewTiKVClient(tikvURL, defaultRequestTimout, tlsCfg, false)
	c.TiKVStatusAddr = opt.TiKVStatusAddr

	return &c, nil
}

func main() {
	opt := Options{}
	opt.Parse()

	cfg, err := opt.Complete()
	if err != nil {
		panic("invalid config for prestop-checker: " + err.Error())
	}

	var storeID string

	//nolint:mnd // refactor to a constant if needed
	if err := wait.PollUntilContextTimeout(context.TODO(), time.Second, time.Second*30, true,
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
				if s.Store.StatusAddress != cfg.TiKVStatusAddr {
					continue
				}

				storeID = strconv.FormatUint(s.Store.Id, 10)
				return true, nil
			}

			return false, nil
		},
	); err != nil {
		panic("cannot find the store, arg addr is wrong or the store has been deleted")
	}

	fmt.Println("pre stop checking, store id:", storeID)

	if err := wait.PollUntilContextCancel(context.TODO(), time.Second, true, func(ctx context.Context) (done bool, err error) {
		leaderCount, err := cfg.TiKVClient.GetLeaderCount()
		if err != nil {
			fmt.Printf("cannot get leader count, try again: %v\n", err)
			return false, nil
		}
		if leaderCount != 0 {
			fmt.Printf("pre stop checking, current leader count: %v\n", leaderCount)
			return false, nil
		}
		return true, nil
	}); err != nil {
		panic("poll error: " + err.Error())
	}

	fmt.Printf("all leaders have been evicted\n")
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
