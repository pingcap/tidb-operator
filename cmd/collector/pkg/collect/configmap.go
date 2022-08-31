package collect

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ConfigMap struct {
	*BaseCollector
}

var _ Collector = (*ConfigMap)(nil)

func (p *ConfigMap) Objects() (<-chan client.Object, error) {
	list := &corev1.ConfigMapList{}
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

func NewConfigMapCollector(cli client.Reader) Collector {
	addCorev1Scheme()
	return &ConfigMap{
		BaseCollector: NewBaseCollector(cli),
	}
}
