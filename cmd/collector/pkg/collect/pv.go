package collect

import (
	"context"
	"sync"

	_ "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PV struct {
	*BaseCollector
}

var _ Collector = (*PV)(nil)

func (pv *PV) Objects() (<-chan client.Object, error) {
	list := &corev1.PersistentVolumeList{}
	err := pv.Reader.List(context.Background(), list, pv.opts...)
	if err != nil {
		return nil, err
	}

	ch := make(chan client.Object, 1)
	go func() {
		for _, obj := range list.Items {
			ch <- &obj
		}
		close(ch)
	}()
	return ch, nil
}

func NewPVCollector(cli client.Reader) Collector {
	addCorev1Scheme()
	return &PV{
		BaseCollector: NewBaseCollector(cli),
	}
}

var corev1Scheme sync.Once

func addCorev1Scheme() {
	corev1Scheme.Do(func() {
		corev1.AddToScheme(scheme)
	})
}
