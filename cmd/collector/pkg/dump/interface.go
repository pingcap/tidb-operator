package dump

// Dumping is the dump type for the output.
// For now, zip is the only supported encoding type.
type Dumping string

// Dumper is an interface for dumping collected information to an output file.
type Dumper interface {
	// Write writes contents in byte array to a specified path.
	Write(path string, content []byte) error
	// Close closes the dumper.
	Close() (closeFn func() error)
}
