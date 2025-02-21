package apicall

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
)

func GetPod[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](ctx context.Context, c client.Client, obj F) (*corev1.Pod, error) {
	pl := corev1.PodList{}
	if err := c.List(ctx, &pl, client.InNamespace(obj.GetNamespace()), client.MatchingLabels{
		v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
		v1alpha1.LabelKeyComponent: scope.Component[S](),
		v1alpha1.LabelKeyInstance:  obj.GetName(),
	}); err != nil {
		return nil, err
	}

	if len(pl.Items) != 1 {
		return nil, fmt.Errorf("expected only 1 pod, but now %d", len(pl.Items))
	}

	pod := &pl.Items[0]

	if !metav1.IsControlledBy(pod, obj) {
		return nil, fmt.Errorf("pod %s/%s(%s) is not controlled by obj %s/%s(%s)",
			pod.Namespace,
			pod.Name,
			pod.UID,
			obj.GetNamespace(),
			obj.GetName(),
			obj.GetUID(),
		)
	}

	return pod, nil
}
