package alias

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Tikv v1.Pod

type TikvList v1.PodList

func (t Tikv) GetObjectKind() schema.ObjectKind {
	return t.GetObjectKind()
}

func (t Tikv) DeepCopyObject() runtime.Object {
	return t.DeepCopyObject()
}

func (t TikvList) GetObjectKind() schema.ObjectKind {
	return t.GetObjectKind()
}

func (t TikvList) DeepCopyObject() runtime.Object {
	return t.DeepCopyObject()
}

func (t *Tikv) FromPod(pod *v1.Pod) {
	t.Labels = pod.Labels
	t.Annotations = pod.Annotations
	t.TypeMeta = pod.TypeMeta
	t.ObjectMeta = pod.ObjectMeta
	t.Spec = pod.Spec
	t.Status = pod.Status
}

func (t *Tikv) ToPod() *v1.Pod {
	pod := &v1.Pod{}
	pod.Labels = t.Labels
	pod.Annotations = t.Annotations
	pod.TypeMeta = t.TypeMeta
	pod.ObjectMeta = t.ObjectMeta
	pod.Spec = t.Spec
	t.Status = pod.Status
	return pod
}

func (t *TikvList) FromPodList(podList *v1.PodList) {
	t.TypeMeta = podList.TypeMeta
	t.ListMeta = podList.ListMeta
	t.Items = podList.Items
}

func (t *TikvList) ToPodList() *v1.PodList {
	podList := &v1.PodList{}
	podList.Items = t.Items
	podList.ListMeta = t.ListMeta
	podList.TypeMeta = t.TypeMeta
	return podList
}
