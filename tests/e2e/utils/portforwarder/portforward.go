package portforwarder

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/pingcap/tidb-operator/pkg/scheme"
)

type PortForwarder interface {
	ForwardPod(ctx context.Context, pod *corev1.Pod, port string, otherPorts ...string) (Forwarded, error)
}

type Forwarded interface {
	context.Context
	Cancel()
	Local(remote uint16) (uint16, bool)
}

type forwarder struct {
	cfg  *rest.Config
	c    rest.Interface
	opts *Options
}

func New(cfg *rest.Config, opts ...Option) PortForwarder {
	o := Options{}

	for _, opt := range opts {
		opt.With(&o)
	}

	return &forwarder{
		cfg:  cfg,
		opts: &o,
	}
}

func (f *forwarder) newPodPortForwarder(
	ctx context.Context,
	pod *corev1.Pod,
	ports []string,
	readyCh chan struct{},
	w io.Writer,
) (*portforward.PortForwarder, error) {
	if f.c == nil {
		httpClient, err := rest.HTTPClientFor(f.cfg)
		if err != nil {
			return nil, fmt.Errorf("cannot new http client: %w", err)
		}
		nc, err := apiutil.RESTClientForGVK(corev1.SchemeGroupVersion.WithKind("Pod"), false, f.cfg, scheme.Codecs, httpClient)
		if err != nil {
			return nil, fmt.Errorf("cannot new rest.Interface: %w", err)
		}
		f.c = nc
	}

	req := f.c.Post().
		Resource("pods").
		Namespace(pod.Namespace).
		Name(pod.Name).
		SubResource("portforward")

	transport, upgrader, err := spdy.RoundTripperFor(f.cfg)
	if err != nil {
		return nil, fmt.Errorf("create round tripper: %w", err)
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())

	tunnelingDialer, err := portforward.NewSPDYOverWebsocketDialer(req.URL(), f.cfg)
	if err != nil {
		return nil, err
	}
	// First attempt tunneling (websocket) dialer, then fallback to spdy dialer.
	dialer = portforward.NewFallbackDialer(tunnelingDialer, dialer, func(err error) bool {
		if httpstream.IsUpgradeFailure(err) || httpstream.IsHTTPSProxyError(err) {
			return true
		}
		return false
	})

	pf, err := portforward.New(dialer, ports, ctx.Done(), readyCh, w, w)
	if err != nil {
		return nil, err
	}
	return pf, nil
}

func (f *forwarder) ForwardPod(ctx context.Context, pod *corev1.Pod, port string, others ...string) (Forwarded, error) {
	readyCh := make(chan struct{})
	ports := []string{port}
	ports = append(ports, others...)
	pf, err := f.newPodPortForwarder(ctx, pod, ports, readyCh, ginkgo.GinkgoWriter)
	if err != nil {
		return nil, err
	}
	nctx, cancel := context.WithCancel(ctx)
	go func() {
		if err := pf.ForwardPorts(); err != nil {
			cancel()
		}
	}()
	select {
	case <-readyCh:
	case <-nctx.Done():
		return nil, fmt.Errorf("port forward is canceled: %w", nctx.Err())
	}
	ps, err := pf.GetPorts()
	if err != nil {
		cancel()
		return nil, err
	}

	return &forwarded{
		Context: nctx,
		cancel:  cancel,
		ports:   ps,
	}, nil
}

type forwarded struct {
	context.Context
	cancel context.CancelFunc
	ports  []portforward.ForwardedPort
}

func (f *forwarded) Local(remote uint16) (uint16, bool) {
	for _, port := range f.ports {
		if port.Remote == remote {
			return port.Local, true
		}
	}

	return 0, false
}

func (f *forwarded) Cancel() {
	f.cancel()
}
