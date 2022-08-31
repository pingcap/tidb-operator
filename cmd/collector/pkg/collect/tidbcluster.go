package collect

import (
	"context"
	"sync"

	opv1alpha1 "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TidbCluster struct {
	*BaseCollector
}

var _ Collector = (*TidbCluster)(nil)

func (p *TidbCluster) Objects() (<-chan client.Object, error) {
	list := &opv1alpha1.TidbClusterList{}
	err := p.Reader.List(context.Background(), list, p.opts...)
	if err != nil {
		return nil, err
	}

	ch := make(chan client.Object)
	go func() {
		for _, obj := range list.Items {
			ch <- &obj
		}
		close(ch)
	}()
	return ch, nil
}

var opv1alpha1Scheme sync.Once

func addOpV1Alpha1Scheme() {
	opv1alpha1Scheme.Do(func() {
		opv1alpha1.AddToScheme(scheme)
	})
}

func NewTidbClusterCollector(cli client.Reader) Collector {
	addOpV1Alpha1Scheme()
	return &TidbCluster{
		BaseCollector: NewBaseCollector(cli),
	}
}
