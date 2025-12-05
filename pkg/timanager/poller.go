// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package timanager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/pingcap/tidb-operator/v2/pkg/client"
)

// Poller polls from tidb and sends watch event to channel
type Poller interface {
	// Sync will initialize the state of poller
	// It will poll and make sure state is available
	// Sync must be called before Run
	Sync(ctx context.Context) (runtime.Object, error)
	// IF Run is called multiple times, the previous one will be stopped
	// immediately and start a new one to push event to new channel
	Run(ctx context.Context, ch chan<- watch.Event)
	// Refresh once immediately
	Refresh()
}

type Lister[T any, PT Object[T], L client.ObjectList] interface {
	List(ctx context.Context) (L, error)
	GetItems(l L) []PT
	MarkAsInvalid(PT) bool
}

type Equality[T any, PT Object[T]] interface {
	Equal(prev, cur PT) bool
}

func NewPoller[T any, PT Object[T], L client.ObjectList](
	name string,
	logger logr.Logger,
	lister Lister[T, PT, L],
	eq Equality[T, PT],
	interval time.Duration,
) Poller {
	return &poller[T, PT, L]{
		name:      name,
		interval:  interval,
		lister:    lister,
		equality:  eq,
		logger:    logger,
		refreshCh: make(chan struct{}, bufSize),
	}
}

type poller[T any, PT Object[T], L client.ObjectList] struct {
	name   string
	logger logr.Logger

	lock    sync.Mutex
	started bool

	resultCh chan watch.Event

	cancel   context.CancelFunc
	interval time.Duration

	refreshCh chan struct{}

	state map[string]PT

	lister   Lister[T, PT, L]
	equality Equality[T, PT]
}

func (p *poller[T, PT, L]) renew(ctx context.Context) context.Context {
	p.stop()
	p.resultCh = make(chan watch.Event, bufSize)

	nctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel

	return nctx
}

func (p *poller[T, PT, L]) Sync(ctx context.Context) (runtime.Object, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.started {
		return nil, fmt.Errorf("poller has started")
	}

	list, err := p.lister.List(ctx)
	if err != nil {
		return nil, err
	}

	items := p.lister.GetItems(list)
	p.updateState(ctx, p.newState(items), false)

	return list, nil
}

func (p *poller[T, PT, L]) stop() {
	if p.cancel != nil {
		p.cancel()
		close(p.resultCh)
		p.cancel = nil
	}
}

func (p *poller[T, PT, L]) Stop() {
	p.stop()
	p.state = nil
}

func (p *poller[T, PT, L]) Run(ctx context.Context, ch chan<- watch.Event) {
	p.lock.Lock()
	p.started = true
	p.lock.Unlock()

	nctx := p.renew(ctx)
	defer p.Stop()

	go func() {
		for {
			select {
			case event := <-p.resultCh:
				ch <- event
			case <-nctx.Done():
				return
			}
		}
	}()

	timer := time.NewTicker(p.interval)
	defer timer.Stop()
	for {
		select {
		case <-nctx.Done():
			p.logger.Info("poller is stopped", "cluster", p.name, "type", new(T))
			return
		case <-p.refreshCh:
			p.poll(ctx)
		case <-timer.C:
			p.poll(ctx)
		}
		timer.Reset(p.interval)
	}
}

func (p *poller[T, PT, L]) Refresh() {
	p.refreshCh <- struct{}{}
}

func (p *poller[T, PT, L]) poll(ctx context.Context) {
	list, err := p.lister.List(ctx)
	if err != nil {
		p.logger.Error(err, "poll err", "cluster", p.name, "type", new(T))
		p.markStateInvalid(ctx)
		return
	}
	objs := p.lister.GetItems(list)

	p.updateState(ctx, p.newState(objs), true)
}

func (*poller[T, PT, L]) newState(objs []PT) map[string]PT {
	s := map[string]PT{}
	for _, obj := range objs {
		s[obj.GetName()] = obj
	}

	return s
}

func (p *poller[T, PT, L]) markStateInvalid(ctx context.Context) {
	for _, v := range p.state {
		if p.lister.MarkAsInvalid(v) {
			p.sendEvent(ctx, &watch.Event{
				Type:   watch.Modified,
				Object: v,
			})
		}
	}
}

func (p *poller[T, PT, L]) updateState(ctx context.Context, newState map[string]PT, sendEvent bool) {
	oldState := p.state
	p.state = newState

	if sendEvent {
		p.generateEvents(ctx, oldState, newState)
	}
}

func (p *poller[T, PT, L]) generateEvents(ctx context.Context, prevState, curState map[string]PT) {
	for _, obj := range curState {
		if preObj, ok := prevState[obj.GetName()]; ok {
			if p.equality.Equal(preObj, obj) {
				continue
			}
			p.sendEvent(ctx, &watch.Event{
				Type:   watch.Modified,
				Object: obj,
			})
		} else {
			p.sendEvent(ctx, &watch.Event{
				Type:   watch.Added,
				Object: obj,
			})
		}
	}

	for name := range prevState {
		if _, ok := curState[name]; !ok {
			p.sendEvent(ctx, &watch.Event{
				Type:   watch.Deleted,
				Object: prevState[name],
			})
		}
	}
}

func (p *poller[T, PT, L]) sendEvent(ctx context.Context, e *watch.Event) {
	select {
	case p.resultCh <- *e:
		p.logger.Info("poller send event", "type", e.Type, "object", e.Object)
	case <-ctx.Done():
	}
}
