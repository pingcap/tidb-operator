package dump

import (
	"io"
)

// Dumping is the dump type for the output.
// For now, zip is the only supported encoding type.
type Dumping string

// Dumper is an interface for dumping collected information to an output file.
type Dumper interface {
	// Open returns an io.Writer to a specified path.
	Open(path string) (io.Writer, error)
	// Close closes the dumper.
	Close() (closeFn func() error)
}
