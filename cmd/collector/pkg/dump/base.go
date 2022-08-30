package dump

import (
	"io"
)

// BaseDumper is a base dumper that implements the Dumper interface.
// It's used to host common objects required by all dumpers.
type BaseDumper struct {
	close func() error
}

var _ Dumper = (*BaseDumper)(nil)

func (b *BaseDumper) Open(string) (io.Writer, error) {
	panic("not implemented")
}

func (b *BaseDumper) Close() (closeFn func() error) {
	return b.close
}

// NewBaseDumper returns an instance of the BaseDumper.
func NewBaseDumper(closeFn func() error) *BaseDumper {
	return &BaseDumper{
		close: closeFn,
	}
}
