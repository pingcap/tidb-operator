// Copyright 2021 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package portforward

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"k8s.io/kubectl/pkg/polymorphichelpers"
	"k8s.io/kubectl/pkg/util/podutils"
	"k8s.io/kubernetes/test/e2e/framework/log"
)

const (
	getPodTimeout = time.Minute
)

// PortForwarder represents an interface which can forward local ports to a pod.
type PortForwarder interface {
	Forward(ctx context.Context, namespace, resourceName string, remotePorts []int) ([]portforward.ForwardedPort, error)
}

// portForwarder implements PortForward interface
type portForwarder struct {
	config *rest.Config
	client kubernetes.Interface
	dc     discovery.CachedDiscoveryInterface
	mapper meta.RESTMapper

	df DialerFunc
}

var _ PortForwarder = &portForwarder{}

func (f *portForwarder) forwardPorts(ctx context.Context, podKey string, url *url.URL, addresses []string, ports []string) ([]portforward.ForwardedPort, error) {
	dialer := f.df(url)

	cleanCh := make(chan struct{})
	r, w := io.Pipe()

	go func() {
		<-cleanCh
		w.Close()
	}()

	readyChan := make(chan struct{})
	fw, err := portforward.NewOnAddresses(dialer, addresses, ports, ctx.Done(), readyChan, w, w)
	if err != nil {
		close(cleanCh)
		return nil, err
	}

	go func() {
		lineScanner := bufio.NewScanner(r)
		for lineScanner.Scan() {
			log.Logf("log from port forwarding %q: %s", podKey, lineScanner.Text())
		}
	}()

	errChan := make(chan error)
	// run port forwarding
	go func() {
		if err := fw.ForwardPorts(); err != nil {
			errChan <- err
		}
		close(cleanCh)
	}()

	// wait for ready or error
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context is done before ready: %v", ctx.Err())
	case <-readyChan:
	case err := <-errChan:
		return nil, err
	}

	forwardedPorts, err := fw.GetPorts()
	if err != nil {
		return nil, err
	}

	return forwardedPorts, nil
}

func (f *portForwarder) Forward(ctx context.Context, namespace, resourceName string, remotePorts []int) ([]portforward.ForwardedPort, error) {
	builder := resource.NewBuilder(f).
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		ContinueOnError().
		NamespaceParam(namespace).DefaultNamespace()

	builder.ResourceNames("pods", resourceName)

	obj, err := builder.Do().Object()
	if err != nil {
		return nil, err
	}
	pod, err := attachablePodForObject(f.client, obj, getPodTimeout)
	if err != nil {
		return nil, err
	}

	// use random local port to avoid conflict
	ports := make([]string, 0, len(remotePorts))
	for _, p := range remotePorts {
		ports = append(ports, ":"+strconv.Itoa(p))
	}

	return f.forwardPod(ctx, pod, []string{"localhost"}, ports)
}

// attachablePodForObject returns the pod to which to attach given an object.
// It is copied from k8s.io/kubectl/pkg/polymorphichelpers/attachablepodforobject.go
// to avoid implementing genericclioptions.RESTClientGetter
func attachablePodForObject(clientset kubernetes.Interface, object runtime.Object, timeout time.Duration) (*corev1.Pod, error) {
	switch t := object.(type) {
	case *corev1.Pod:
		return t, nil
	}

	namespace, selector, err := polymorphichelpers.SelectorsForObject(object)
	if err != nil {
		return nil, fmt.Errorf("cannot attach to %T: %v", object, err)
	}
	sortBy := func(pods []*corev1.Pod) sort.Interface { return sort.Reverse(podutils.ActivePods(pods)) }
	pod, _, err := polymorphichelpers.GetFirstPod(clientset.CoreV1(), namespace, selector.String(), timeout, sortBy)
	return pod, err
}

func (f *portForwarder) forwardPod(ctx context.Context, pod *corev1.Pod, addresses []string, ports []string) (forwardedPorts []portforward.ForwardedPort, err error) {
	if pod.Status.Phase != corev1.PodRunning {
		return nil, fmt.Errorf("unable to forward port because pod is not running. Current status=%v", pod.Status.Phase)
	}

	req := f.client.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(pod.Namespace).
		Name(pod.Name).
		SubResource("portforward")

	return f.forwardPorts(ctx, fmt.Sprintf("%s/%s", pod.Namespace, pod.Name), req.URL(), addresses, ports)
}

func NewPortForwarderForConfig(config *rest.Config) (PortForwarder, error) {
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	dc := memory.NewMemCacheClient(client.Discovery())
	mapper := restmapper.NewShortcutExpander(restmapper.NewDeferredDiscoveryRESTMapper(dc), dc)

	df, err := NewDialerFunc(config)
	if err != nil {
		return nil, err
	}
	f := &portForwarder{
		config: config,
		client: client,
		dc:     dc,
		mapper: mapper,
		df:     df,
	}
	return f, nil
}

type DialerFunc func(u *url.URL) httpstream.Dialer

func NewDialerFunc(config *rest.Config) (DialerFunc, error) {
	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return nil, err
	}
	httpClient := &http.Client{Transport: transport}
	return func(u *url.URL) httpstream.Dialer {
		return spdy.NewDialer(upgrader, httpClient, "POST", u)
	}, nil
}

func (f *portForwarder) ToRESTConfig() (*rest.Config, error) {
	return f.config, nil
}
func (f *portForwarder) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	return f.dc, nil
}
func (f *portForwarder) ToRESTMapper() (meta.RESTMapper, error) {
	return f.mapper, nil
}
