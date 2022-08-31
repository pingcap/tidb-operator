package collect

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Event struct {
	*BaseCollector
}

var _ Collector = (*Event)(nil)

func (p *Event) Objects() (<-chan client.Object, error) {
	list := &corev1.EventList{}
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

func NewEventCollector(cli client.Reader) Collector {
	addCorev1Scheme()
	return &Event{
		BaseCollector: NewBaseCollector(cli),
	}
}
