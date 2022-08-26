package collect

import (
	"context"

	_ "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Service struct {
	*BaseCollector
}

var _ Collector = (*Service)(nil)

func (p *Service) Objects() (<-chan client.Object, error) {
	list := &corev1.ServiceList{}
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

func NewServiceCollector(cli client.Reader) Collector {
	addCorev1Scheme()
	return &Service{
		BaseCollector: NewBaseCollector(cli),
	}
}
