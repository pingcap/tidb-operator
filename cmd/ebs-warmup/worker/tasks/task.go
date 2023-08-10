package tasks

import (
	"fmt"
	"os"
)

type ReadType interface {
	fmt.Stringer
}

type ReadOffsetAndLength struct {
	Offset int64
	Length int64
}

func (r ReadOffsetAndLength) String() string {
	return fmt.Sprintf("ReadOffsetAndLength(%d, %d)", r.Offset, r.Length)
}

type ReadLastNBytes int

func (r ReadLastNBytes) String() string {
	return fmt.Sprintf("ReadLastNBytes(%d)", r)
}

type ReadFull struct{}

func (r ReadFull) String() string {
	return "ReadFull"
}

type Sync struct {
	C chan<- struct{}
}

func (r Sync) String() string {
	return "Sync"
}

type ReadFile struct {
	Type     ReadType
	File     os.FileInfo
	FilePath string
	Direct   bool
}

func (r ReadFile) String() string {
	return fmt.Sprintf("ReadFile(%q, %s)", r.FilePath, r.Type)
}
