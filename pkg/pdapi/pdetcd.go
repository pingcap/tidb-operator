// Copyright 2020 PingCAP, Inc.
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

package pdapi

import (
	"context"
	"crypto/tls"
	"time"

	etcdclientv3 "github.com/coreos/etcd/clientv3"
)

type PDEtcdApi interface {
	PutKey(key, value string) error

	DeleteKey(key string) error
}

type pdEtcdClient struct {
	timeout    time.Duration
	etcdClient *etcdclientv3.Client
}

func NewPdEtcdClient(url string, timeout time.Duration, tlsConfig *tls.Config) (PDEtcdApi, error) {
	etcdClient, err := etcdclientv3.New(etcdclientv3.Config{
		Endpoints:   []string{url},
		DialTimeout: timeout,
		TLS:         tlsConfig,
	})
	if err != nil {
		return nil, err
	}
	return &pdEtcdClient{
		etcdClient: etcdClient,
		timeout:    timeout,
	}, nil
}

func (pec *pdEtcdClient) PutKey(key, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), pec.timeout)
	defer cancel()
	_, err := pec.etcdClient.Put(ctx, key, value)
	return err
}

func (pec *pdEtcdClient) DeleteKey(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), pec.timeout)
	defer cancel()
	_, err := pec.etcdClient.Delete(ctx, key)
	return err
}
