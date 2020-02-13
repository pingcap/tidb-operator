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

package blockwriter

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pingcap/tidb-operator/tests/pkg/util"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
)

const (
	queryChanSize int = 100
)

// BlockWriterCase is for concurrent writing blocks.
type BlockWriterCase struct {
	bws []*blockWriter

	isRunning uint32
	isInit    uint32
	stopChan  chan struct{}

	cfg         Config
	ClusterName string

	sync.RWMutex
}

// Config defines the config of BlockWriterCase
type Config struct {
	TableNum    int `yaml:"table_num" json:"table_num"`
	Concurrency int `yaml:"concurrency" json:"concurrency"`
	BatchSize   int `yaml:"batch_size" json:"batch_size"`
	RawSize     int `yaml:"raw_size" json:"raw_size"`
}

type blockWriter struct {
	rawSize   int
	values    []string
	batchSize int
}

// NewBlockWriterCase returns the BlockWriterCase.
func NewBlockWriterCase(cfg Config) *BlockWriterCase {
	c := &BlockWriterCase{
		cfg:      cfg,
		stopChan: make(chan struct{}, 1),
	}

	if c.cfg.TableNum < 1 {
		c.cfg.TableNum = 1
	}
	c.initBlocks()

	return c
}

func (c *BlockWriterCase) GetConcurrency() int {
	return c.cfg.Concurrency
}

func (c *BlockWriterCase) initBlocks() {
	c.bws = make([]*blockWriter, c.cfg.Concurrency)
	for i := 0; i < c.cfg.Concurrency; i++ {
		c.bws[i] = c.newBlockWriter()
	}
}

func (c *BlockWriterCase) newBlockWriter() *blockWriter {
	return &blockWriter{
		rawSize:   c.cfg.RawSize,
		values:    make([]string, c.cfg.BatchSize),
		batchSize: c.cfg.BatchSize,
	}
}

func (c *BlockWriterCase) generateQuery(ctx context.Context, queryChan chan []string, wg *sync.WaitGroup) {
	defer func() {
		klog.Infof("[%s] [%s] [action: generate Query] stopped", c, c.ClusterName)
		wg.Done()
	}()

	for {
		tableN := rand.Intn(c.cfg.TableNum)
		var index string
		if tableN > 0 {
			index = fmt.Sprintf("%d", tableN)
		}

		var querys []string
		for i := 0; i < 100; i++ {
			values := make([]string, c.cfg.BatchSize)
			for i := 0; i < c.cfg.BatchSize; i++ {
				blockData := util.RandString(c.cfg.RawSize)
				values[i] = fmt.Sprintf("('%s')", blockData)
			}

			querys = append(querys, fmt.Sprintf(
				"INSERT INTO block_writer%s(raw_bytes) VALUES %s",
				index, strings.Join(values, ",")))
		}

		select {
		case <-ctx.Done():
			return
		case queryChan <- querys:
			continue
		default:
			klog.V(4).Infof("[%s] [%s] [action: generate Query] query channel is full, sleep 10 seconds", c, c.ClusterName)
			util.Sleep(ctx, 10*time.Second)
		}
	}
}

func (bw *blockWriter) batchExecute(db *sql.DB, query string) error {
	_, err := db.Exec(query)
	if err != nil {
		klog.V(4).Infof("exec sql [%s] failed, err: %v", query, err)
		return err
	}

	return nil
}

func (bw *blockWriter) run(ctx context.Context, db *sql.DB, queryChan chan []string) {
	defer klog.Infof("run stopped")
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		querys, ok := <-queryChan
		if !ok {
			// No more query
			return
		}

		for _, query := range querys {
			select {
			case <-ctx.Done():
				return
			default:
				if err := bw.batchExecute(db, query); err != nil {
					klog.V(4).Info(err)
					time.Sleep(5 * time.Second)
					continue
				}
			}
		}
	}
}

// Initialize inits case
func (c *BlockWriterCase) initialize(db *sql.DB) error {
	klog.Infof("[%s] [%s] start to init...", c, c.ClusterName)
	defer func() {
		atomic.StoreUint32(&c.isInit, 1)
		klog.Infof("[%s] [%s] init end...", c, c.ClusterName)
	}()

	for i := 0; i < c.cfg.TableNum; i++ {
		var s string
		if i > 0 {
			s = fmt.Sprintf("%d", i)
		}

		tmt := fmt.Sprintf("CREATE TABLE IF NOT EXISTS block_writer%s %s", s, `
	(
      id BIGINT NOT NULL AUTO_INCREMENT,
      raw_bytes BLOB NOT NULL,
      PRIMARY KEY (id)
)`)

		err := wait.PollImmediate(5*time.Second, 30*time.Second, func() (bool, error) {
			_, err := db.Exec(tmt)
			if err != nil {
				klog.Warningf("[%s] exec sql [%s] failed, err: %v, retry...", c, tmt, err)
				return false, nil
			}

			return true, nil
		})

		if err != nil {
			klog.Errorf("[%s] exec sql [%s] failed, err: %v", c, tmt, err)
			return err
		}
	}

	return nil
}

// Start starts to run cases
func (c *BlockWriterCase) Start(db *sql.DB) error {
	if !atomic.CompareAndSwapUint32(&c.isRunning, 0, 1) {
		err := fmt.Errorf("[%s] [%s] is running, you can't start it again", c, c.ClusterName)
		klog.Error(err)
		return nil
	}

	defer func() {
		c.RLock()
		klog.Infof("[%s] [%s] stopped", c, c.ClusterName)
		atomic.SwapUint32(&c.isRunning, 0)
	}()

	if c.isInit == 0 {
		if err := c.initialize(db); err != nil {
			return err
		}
	}

	klog.Infof("[%s] [%s] start to execute case...", c, c.ClusterName)

	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())

	queryChan := make(chan []string, queryChanSize)

	for i := 0; i < c.cfg.Concurrency; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c.bws[i].run(ctx, db, queryChan)
		}(i)
	}

	wg.Add(1)
	go c.generateQuery(ctx, queryChan, &wg)

loop:
	for {
		select {
		case <-c.stopChan:
			klog.Infof("[%s] stoping...", c)
			cancel()
			break loop
		default:
			util.Sleep(context.Background(), 2*time.Second)
		}
	}

	wg.Wait()
	close(queryChan)

	return nil
}

// Stop stops cases
func (c *BlockWriterCase) Stop() {
	c.stopChan <- struct{}{}
}

// String implements fmt.Stringer interface.
func (c *BlockWriterCase) String() string {
	return "block_writer"
}
