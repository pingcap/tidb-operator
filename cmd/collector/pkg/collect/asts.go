package collect

import (
	"context"
	"sync"

	"sigs.k8s.io/controller-runtime/pkg/client"

	asapps "github.com/pingcap/advanced-statefulset/client/apis/apps/v1"
)

type AdvancedStatefulSet struct {
	*BaseCollector
}

var _ Collector = (*AdvancedStatefulSet)(nil)

func (p *AdvancedStatefulSet) Objects() (<-chan client.Object, error) {
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
	return &AdvancedStatefulSet{
		BaseCollector: NewBaseCollector(cli),
	}
}

var asappsv1Scheme sync.Once

func addAsappsV1Scheme() {
	asappsv1Scheme.Do(func() {
		asapps.AddToScheme(scheme)
	})
}
