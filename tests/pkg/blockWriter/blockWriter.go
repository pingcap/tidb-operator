package blockWriter

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/tests/pkg/util"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	statChanSize  int = 10000
	queryChanSize int = 10000

	defaultRowSize int = 512
)

// BlockWriterCase is for concurrent writing blocks.
type BlockWriterCase struct {
	cfg    Config
	result Result
	bws    []*blockWriter

	isRunning uint32
	isInit    uint32
	stopChan  chan struct{}

	sync.RWMutex
}

// Config defines the config of BlockWriterCase
type Config struct {
	TableNum    int
	Concurrency int
	BatchSize   int
}

// Result is the result of BlockWriterCase,
type Result struct {
	Total uint64
	Succ  uint64

	SingleCount uint64
	SingleSucc  uint64
}

type blockWriter struct {
	rowSize         int
	blockDataBuffer []byte
	values          []string
	batchSize       int
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

func (c *BlockWriterCase) initBlocks() {
	c.bws = make([]*blockWriter, c.cfg.Concurrency)
	for i := 0; i < c.cfg.Concurrency; i++ {
		c.bws[i] = c.newBlockWriter()
	}
}

func (c *BlockWriterCase) newBlockWriter() *blockWriter {
	return &blockWriter{
		rowSize:         defaultRowSize,
		blockDataBuffer: make([]byte, 1024),
		values:          make([]string, c.cfg.BatchSize),
		batchSize:       c.cfg.BatchSize,
	}
}

func (c *BlockWriterCase) generateQuery(ctx context.Context, queryChan chan string, gwg *sync.WaitGroup) {
	defer gwg.Done()
	var wg sync.WaitGroup
	var src = rand.NewSource(time.Now().UnixNano())
	rand := rand.New(src)

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				tableN := rand.Intn(c.cfg.TableNum)
				var index string
				if tableN > 0 {
					index = fmt.Sprintf("%d", tableN)
				}

				values := make([]string, c.cfg.BatchSize)
				for i := 0; i < c.cfg.BatchSize; i++ {
					blockData := util.RandString(defaultRowSize, rand)
					values[i] = fmt.Sprintf("('%s')", blockData)
					queryChan <- fmt.Sprintf(
						"INSERT INTO block_writer%s VALUES %s",
						index, strings.Join(values, ","))
				}
			}
		}()
	}

	wg.Wait()
}

func (c *BlockWriterCase) updateResult(ctx context.Context, statChan chan *stat, gwg *sync.WaitGroup) {
	defer gwg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		st, ok := <-statChan
		if !ok {
			return
		}

		c.Lock()
		atomic.AddUint64(&c.result.Total, 1)
		atomic.AddUint64(&c.result.SingleCount, 1)
		if st.succ {
			atomic.AddUint64(&c.result.Succ, 1)
			atomic.AddUint64(&c.result.SingleSucc, 1)
		}
		c.Unlock()
	}
}

func (bw *blockWriter) batchExecute(db *sql.DB, query string, statChan chan *stat) {
	_, err := db.Exec(query)
	if err != nil {
		glog.Warningf("[block_writer] exec sql [%s] failed, err: %v", query, err)
		statChan <- &stat{}
		return
	}

	statChan <- &stat{succ: true}
}

func (bw *blockWriter) run(ctx context.Context, db *sql.DB, queryChan chan string, statChan chan *stat) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		query, ok := <-queryChan
		if !ok {
			// No more query
			return
		}
		bw.batchExecute(db, query, statChan)
	}
}

// Initialize inits case
func (c *BlockWriterCase) Initialize(ctx context.Context, db *sql.DB) error {
	if atomic.CompareAndSwapUint32(&c.isInit, 0, 1) {
		err := fmt.Errorf("[%s] is inited", c)
		glog.Error(err)
		return err
	}

	glog.Infof("[%s] start to init...", c)
	defer func() {
		atomic.StoreUint32(&c.isInit, 1)
		glog.Infof("[%s] init end...", c)
	}()

	for i := 0; i < c.cfg.TableNum; i++ {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
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
				glog.Warningf("[%s] exec sql [%s] failed, err: %v, retry...", c, tmt, err)
				return false, nil
			}

			return true, nil
		})

		if err != nil {
			glog.Errorf("[%s] exec sql [%s] failed, err: %v", c, tmt, err)
			return err
		}
	}

	return nil
}

// Start starts to run cases
func (c *BlockWriterCase) Start(ctx context.Context, db *sql.DB) error {
	if atomic.CompareAndSwapUint32(&c.isRunning, 0, 1) {
		err := fmt.Errorf("[%s] is running, you can't start it again", c)
		glog.Error(err)
		return err
	}

	defer func() {
		c.RLock()
		glog.Infof("[%s] [Query: %d, Succ: %d, Failed: %d]", c, c.result.SingleCount,
			c.result.SingleSucc, c.result.SingleCount-c.result.SingleSucc)
		glog.Infof("[%s] [Total Query: %d, Succ: %d, Failed: %d]", c, c.result.Total,
			c.result.Succ, c.result.Total-c.result.Succ)
		glog.Infof("[%s] stoped...", c)
		c.Unlock()

		atomic.SwapUint32(&c.isRunning, 0)
	}()

	if !atomic.CompareAndSwapUint32(&c.isInit, 0, 1) {
		if err := c.Initialize(ctx, db); err != nil {
			return err
		}
	}

	glog.Infof("[%s] start to execute case...", c)

	var wg sync.WaitGroup

	cctx, cancel := context.WithCancel(ctx)

	queryChan := make(chan string, queryChanSize)
	statChan := make(chan *stat, statChanSize)

	go func() {
		for {
			select {
			case <-ctx.Done():
				close(statChan)
				close(queryChan)
				cancel()
				return
			case <-c.stopChan:
				close(statChan)
				close(queryChan)
				cancel()
				return
			default:
			}
		}
	}()

	for i := 0; i < c.cfg.Concurrency; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c.bws[i].run(ctx, db, queryChan, statChan)
		}(i)
	}

	wg.Add(2)
	go c.generateQuery(cctx, queryChan, &wg)
	go c.updateResult(ctx, statChan, &wg)

	wg.Wait()

	return nil
}

// Structure for stat result.
type stat struct {
	succ bool
}

// Stop stops cases
func (c *BlockWriterCase) Stop() {
	c.stopChan <- struct{}{}
}

// GetResult returns the results of the cases
func (c *BlockWriterCase) GetResult() Result {
	c.RLock()
	defer c.RUnlock()

	return c.result
}

// String implements fmt.Stringer interface.
func (c *BlockWriterCase) String() string {
	return "block_writer"
}
