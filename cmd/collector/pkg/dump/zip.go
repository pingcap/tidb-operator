package dump

import (
	"archive/zip"
	"os"
)

const Zip Dumping = "zip"

type ZipDumper struct {
	*BaseDumper
	writer *zip.Writer
}

var _ Dumper = (*ZipDumper)(nil)

func (z *ZipDumper) Write(path string, content []byte) error {
	iowriter, err := z.writer.Create(path)
	if err != nil {
		return err
	}

	_, err = iowriter.Write(content)
	return err
}

// NewZipDumper takes returns a dumper that writes files to input path.
func NewZipDumper(path string) (*ZipDumper, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	writer := zip.NewWriter(file)
	return &ZipDumper{
		BaseDumper: NewBaseDumper(writer.Close),
		writer:     writer,
	}, nil
}
