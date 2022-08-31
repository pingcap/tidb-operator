package collect

import (
	"context"
	"sync"

	"sigs.k8s.io/controller-runtime/pkg/client"

	asapps "github.com/pingcap/advanced-statefulset/client/apis/apps/v1"
)

type StatefulSet struct {
	*BaseCollector
}

var _ Collector = (*StatefulSet)(nil)

func (p *StatefulSet) Objects() (<-chan client.Object, error) {
	list := &asapps.StatefulSetList{}
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

func NewASTSCollector(cli client.Reader) Collector {
	addAsappsV1Scheme()
	return &StatefulSet{
		BaseCollector: NewBaseCollector(cli),
	}
}

var asappsv1Scheme sync.Once

func addAsappsV1Scheme() {
	asappsv1Scheme.Do(func() {
		asapps.AddToScheme(scheme)
	})
}
