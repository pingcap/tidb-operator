package options

import (
	"math"

	"github.com/pingcap/errors"

	"github.com/spf13/pflag"
)

const (
	fromTSUnset  = math.MaxUint64
	untilTSUnset = 0

	namespaceFlag     = "namespace"
	resourceNameFlag  = "resourceName"
	tiKVVersionFlag   = "tikvVersion"
	storageStringFlag = "storage-string"
	fromTSFlag        = "from-ts"
	untilTSFlag       = "until-ts"
	nameFlag          = "name"
	concurrencyFlag   = "concurrency"
)

type KubeOpts struct {
	// This should be fill by the caller.
	Kubeconfig   string `json:"-"`
	Namespace    string `json:"namespace"`
	ResourceName string `json:"resourceName"`
	TiKVVersion  string `json:"tikvVersion"`
}

type CompactOpts struct {
	FromTS      uint64
	UntilTS     uint64
	Name        string
	Concurrency uint64
	StorageOpts []string
}

func DefineFlags(fs *pflag.FlagSet) {
	fs.String(tiKVVersionFlag, "", "TiKV version of the resource")
	fs.String(namespaceFlag, "", "Namespace of the resource")
	fs.String(resourceNameFlag, "", "Name of the resource")
}

func (k *KubeOpts) ParseFromFlags(fs *pflag.FlagSet) error {
	var err error
	k.Namespace, err = fs.GetString(namespaceFlag)
	if err != nil {
		return errors.Trace(err)
	}
	k.ResourceName, err = fs.GetString(resourceNameFlag)
	if err != nil {
		return errors.Trace(err)
	}
	k.TiKVVersion, err = fs.GetString(tiKVVersionFlag)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (c *CompactOpts) Verify() error {
	if c.UntilTS < c.FromTS {
		if c.UntilTS == untilTSUnset {
			return errors.New("until-ts must be set")
		}
		if c.FromTS == fromTSUnset {
			return errors.New("from-ts must be set")
		}
		return errors.Errorf("until-ts %d must be greater than from-ts %d", c.UntilTS, c.FromTS)
	}
	if c.Concurrency <= 0 {
		return errors.Errorf("concurrency %d must be greater than 0", c.Concurrency)
	}
	return nil
}
