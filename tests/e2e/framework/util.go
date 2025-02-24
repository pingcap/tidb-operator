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

package framework

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/pingcap/tidb-operator/pkg/apicall"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/scheme"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

func newConfig(configPath, ctxName string) (*rest.Config, error) {
	rule := clientcmd.NewDefaultClientConfigLoadingRules()
	rule.ExplicitPath = configPath

	kubeconfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		rule,
		&clientcmd.ConfigOverrides{CurrentContext: ctxName}).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("can't parse kubeconfig: %w", err)
	}

	return kubeconfig, nil
}

func newClient(cfg *rest.Config) (client.Client, error) {
	c, err := client.New(cfg, client.Options{
		Scheme: scheme.Scheme,
	})
	if err != nil {
		return nil, fmt.Errorf("can't new client: %w", err)
	}
	return c, nil
}

func newRESTClientForPod(cfg *rest.Config) (rest.Interface, error) {
	httpClient, err := rest.HTTPClientFor(cfg)
	if err != nil {
		return nil, fmt.Errorf("cannot new http client: %w", err)
	}
	return apiutil.RESTClientForGVK(corev1.SchemeGroupVersion.WithKind("Pod"), false, cfg, scheme.Codecs, httpClient)
}

func logPod(ctx context.Context, c rest.Interface, pod *corev1.Pod, follow bool) (io.ReadCloser, error) {
	req := c.Get().
		Namespace(pod.Namespace).
		Name(pod.Name).
		Resource("pods").
		SubResource("log").
		VersionedParams(&corev1.PodLogOptions{Follow: follow}, scheme.ParameterCodec)
	return req.Stream(ctx)
}

func waitInstanceLogContains[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](ctx context.Context, f *Framework, instance F, substr string) {
	pod, err := apicall.GetPod[S](ctx, f.Client, instance)
	gomega.Expect(err).To(gomega.Succeed())

	logs, err := logPod(ctx, f.podLogClient, pod, true)
	gomega.Expect(err).To(gomega.Succeed())
	defer logs.Close()

	sb := strings.Builder{}
	reader := bufio.NewReader(logs)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			ginkgo.AddReportEntry("PodLogs", sb.String())
			ginkgo.Fail(fmt.Sprintf("cannot find str '%s' in logs", substr))
			return
		}

		if strings.Contains(line, substr) {
			return
		}

		sb.WriteString(line)
	}
}

func restartInstancePod[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](ctx context.Context, f *Framework, instance F) {
	pod, err := apicall.GetPod[S](ctx, f.Client, instance)
	gomega.Expect(err).To(gomega.Succeed())
	f.Must(f.Client.Delete(ctx, pod))

	createTime := pod.CreationTimestamp

	f.Must(waiter.WaitForObject(ctx, f.Client, pod, func() error {
		if !pod.CreationTimestamp.Equal(&createTime) {
			return nil
		}

		return fmt.Errorf("pod is not recreated")
	}, waiter.LongTaskTimeout))
}
