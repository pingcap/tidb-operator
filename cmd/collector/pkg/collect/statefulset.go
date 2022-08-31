package collect

import (
	"context"
	"sync"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StatefulSet struct {
	*BaseCollector
}

var _ Collector = (*StatefulSet)(nil)

func (p *StatefulSet) Objects() (<-chan client.Object, error) {
	list := &appsv1.StatefulSetList{}
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

func NewSSCollector(cli client.Reader) Collector {
	addAppsV1Scheme()
	return &StatefulSet{
		BaseCollector: NewBaseCollector(cli),
	}
}

var appsv1Scheme sync.Once

func addAppsV1Scheme() {
	appsv1Scheme.Do(func() {
		appsv1.AddToScheme(scheme)
	})
}
