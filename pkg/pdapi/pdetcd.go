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

	etcdclientv3 "go.etcd.io/etcd/client/v3"
	etcdclientv3util "go.etcd.io/etcd/client/v3/clientv3util"
)

type KeyValue struct {
	Key   string
	Value []byte
}

type PDEtcdClient interface {
	// Get the specific kvs.
	// if prefix is true will return all kvs with the specified key as prefix
	Get(key string, prefix bool) (kvs []*KeyValue, err error)
	// PutKey will put key to the target pd etcd cluster
	PutKey(key, value string) error
	// PutKey will put key with ttl to the target pd etcd cluster
	PutTTLKey(key, value string, ttl int64) error
	// DeleteKey will delete key from the target pd etcd cluster
	DeleteKey(key string) error
	// Close will close the etcd connection
	Close() error
}

type pdEtcdClient struct {
	timeout    time.Duration
	etcdClient *etcdclientv3.Client
}

func NewPdEtcdClient(url string, timeout time.Duration, tlsConfig *tls.Config) (PDEtcdClient, error) {
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

func (c *pdEtcdClient) Close() error {
	return c.etcdClient.Close()
}

func (c *pdEtcdClient) Get(key string, prefix bool) (kvs []*KeyValue, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	var ops []etcdclientv3.OpOption
	if prefix {
		ops = append(ops, etcdclientv3.WithPrefix())
	}
	resp, err := c.etcdClient.Get(ctx, key, ops...)
	if err != nil {
		return nil, err
	}

	for _, kv := range resp.Kvs {
		kvs = append(kvs, &KeyValue{
			Key:   string(kv.Key),
			Value: kv.Value,
		})
	}

	return
}

func (c *pdEtcdClient) PutKey(key, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	_, err := c.etcdClient.Put(ctx, key, value)
	return err
}

func (c *pdEtcdClient) PutTTLKey(key, value string, ttl int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	lease, err := c.etcdClient.Grant(ctx, ttl)
	if err != nil {
		return err
	}
	_, err = c.etcdClient.Put(ctx, key, value, etcdclientv3.WithLease(lease.ID))
	return err
}

func (c *pdEtcdClient) DeleteKey(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	kvc := etcdclientv3.NewKV(c.etcdClient)

	// perform a delete only if key already exists
	_, err := kvc.Txn(ctx).
		If(etcdclientv3util.KeyExists(key)).
		Then(etcdclientv3.OpDelete(key)).
		Commit()
	if err != nil {
		return err
	}
	return nil
}
