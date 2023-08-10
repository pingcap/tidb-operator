package worker

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/docker/go-units"
	"github.com/ncw/directio"
	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/cmd/ebs-warmup/worker/tasks"
	"golang.org/x/time/rate"
	"k8s.io/klog/v2"
)

// Worker is the handler for reading tasks.
// It shall be !Send.
type Worker struct {
	mailbox <-chan tasks.ReadFile
	buf     []byte

	OnStep      func(file os.FileInfo, readBytes int, take time.Duration)
	RateLimiter *rate.Limiter
}

func New(input <-chan tasks.ReadFile) Worker {
	return Worker{
		mailbox: input,
		// Anyway... Aligning with block doesn't cost many.
		buf: directio.AlignedBlock(256 * units.KiB),
	}
}

func (w *Worker) MainLoop(cx context.Context) error {
	for {
		select {
		case <-cx.Done():
			return cx.Err()
		case task, ok := <-w.mailbox:
			if !ok {
				return nil
			}
			if err := w.handleReadFile(task); err != nil {
				klog.InfoS("Failed to read file.", "err", err)
			}
		}
	}
}

func (w *Worker) handleReadFile(task tasks.ReadFile) error {
	if sync, ok := task.Type.(tasks.Sync); ok {
		close(sync.C)
		return nil
	}
	fd, err := w.openFileByTask(task)
	if err != nil {
		return errors.Annotatef(err, "failed to open file for task %s", task)
	}
	defer fd.Close()
	err = w.execReadFile(fd, task)
	if err != nil {
		return errors.Annotatef(err, "failed to read file for opened file %s", task.FilePath)
	}
	return nil
}

func (w *Worker) openFileByTask(task tasks.ReadFile) (*os.File, error) {
	var (
		fd  *os.File
		err error
	)
	if task.Direct {
		fd, err = directio.OpenFile(task.FilePath, os.O_RDONLY, 0o644)
	} else {
		fd, err = os.OpenFile(task.FilePath, os.O_RDONLY, 0o644)
	}
	if err != nil {
		return nil, err
	}
	return fd, nil
}

func (w *Worker) execReadFile(fd *os.File, task tasks.ReadFile) error {
	begin := time.Now()
	reader, err := w.makeReaderByTask(fd, task)
	if err != nil {
		return errors.Annotatef(err, "failed to make reader for task %s", task)
	}
	n, err := w.fetchAll(reader)
	if w.OnStep != nil {
		w.OnStep(task.File, n, time.Since(begin))
	}
	return errors.Annotatef(err, "failed to read from file %s", fd.Name())
}

func (w *Worker) makeReaderByTask(fd *os.File, task tasks.ReadFile) (io.Reader, error) {
	switch t := task.Type.(type) {
	case tasks.ReadLastNBytes:
		s, err := fd.Stat()
		if err != nil {
			return nil, errors.Annotate(err, "failed to open the file stat")
		}
		seekPoint := int64(0)
		if s.Size() > int64(t) {
			seekPoint = s.Size() - int64(t)
		}
		if task.Direct {
			seekPoint = alignBlockBackward(seekPoint)
		}
		if _, err := fd.Seek(seekPoint, io.SeekStart); err != nil {
			return nil, errors.Annotatef(err, "failed to seek to %d", seekPoint)
		}
		return fd, nil
	case tasks.ReadOffsetAndLength:
		if task.Direct {
			// Move the base cursor back to align the block edge.
			// Move the length cursor forward to align
			newOffset := alignBlockBackward(t.Offset)
			t.Length = alignBlockForward(t.Length + newOffset - t.Offset)
			t.Offset = newOffset
		}
		if _, err := fd.Seek(int64(t.Offset), io.SeekStart); err != nil {
			return nil, errors.Annotatef(err, "failed to seek to %d", t.Offset)
		}
		return io.LimitReader(fd, int64(t.Length)), nil
	case tasks.ReadFull:
		return fd, nil
	default:
		return nil, errors.Errorf("unknown read type %T", t)
	}
}

func (w *Worker) fetchAll(r io.Reader) (int, error) {
	total := 0
	for {
		n, err := r.Read(w.buf)
		if err == io.EOF {
			return total + n, nil
		}
		if err != nil {
			return 0, err
		}
		total += n
		if w.RateLimiter != nil {
			res := w.RateLimiter.ReserveN(time.Now(), n)
			if !res.OK() {
				return 0, fmt.Errorf("the read block size %d is larger than rate limit %d", n, w.RateLimiter.Burst())
			}
			time.Sleep(res.Delay())
		}
	}
}

// alignBlockBackward aligns the pointer with the block size by moving it backward.
func alignBlockBackward(n int64) int64 {
	// or n & ~directio.BlockSize
	return n - n%directio.BlockSize
}

// alignBlockForward aligns the pointer with the block size by moving it forward.
func alignBlockForward(n int64) int64 {
	// or n & ~directio.BlockSize + directio.BlockSize
	return directio.BlockSize * (n/directio.BlockSize + 1)
}
