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
)

const (
	defaultPDRequestTimout = 10 * time.Second
)

var (
	pd   string
	addr string
	ca   string
	cert string
	key  string
)

func main() {
	flag.StringVar(&pd, "pd", "", "pd url")
	flag.StringVar(&addr, "addr", "", "tikv advertised client url")
	flag.StringVar(&addr, "ca", "", "ca path")
	flag.StringVar(&addr, "cert", "", "cert path")
	flag.StringVar(&addr, "key", "", "key path")
	flag.Parse()

	u, err := url.Parse(pd)
	if err != nil {
		panic("cannot recognize pd addr: " + err.Error())
	}

	var tlsCfg *tls.Config
	if u.Scheme == "https" {
		cfg, err := loadTLSConfig(ca, cert, key)
		if err != nil {
			panic("cannot load tls config: " + err.Error())
		}
		tlsCfg = cfg
	}

	c := pdapi.NewPDClient(pd, defaultPDRequestTimout, tlsCfg)
	var storeID string

	//nolint:mnd // refactor to a constant if needed
	if err := wait.PollUntilContextTimeout(context.TODO(), time.Second, time.Second*30,
		true, func(ctx context.Context) (done bool, err error) {
			info, err := c.GetStores(ctx)
			if err != nil {
				fmt.Printf("cannot list stores, try again: %v\n", err)
				return false, nil
			}

			for _, s := range info.Stores {
				if s.Store == nil {
					continue
				}
				if s.Store.Address != addr {
					continue
				}

				storeID = strconv.FormatUint(s.Store.Id, 10)
				return true, nil
			}

			return false, nil
		}); err != nil {
		panic("cannot find the store, arg addr is wrong or the store has been deleted")
	}

	fmt.Println("pre stop checking, store id:", storeID)

	if err := wait.PollUntilContextCancel(context.TODO(), time.Second, true, func(ctx context.Context) (done bool, err error) {
		s, err := c.GetStore(ctx, storeID)
		if err != nil {
			fmt.Printf("cannot get store, try again: %v\n", err)
			return false, nil
		}
		if s.Status.LeaderCount != 0 {
			fmt.Printf("pre stop checking, current leader count: %v\n", s.Status.LeaderCount)
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
