package predicates

import (
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func fakeThreeNodes() []apiv1.Node {
	return []apiv1.Node{
		{
			TypeMeta:   metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{Name: "kube-node-1"},
		},
		{
			TypeMeta:   metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{Name: "kube-node-2"},
		},
		{
			TypeMeta:   metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{Name: "kube-node-3"},
		},
	}
}

func fakeTwoNodes() []apiv1.Node {
	return []apiv1.Node{
		{
			TypeMeta:   metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{Name: "kube-node-1"},
		},
		{
			TypeMeta:   metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{Name: "kube-node-3"},
		},
	}
}

func fakeOneNode() []apiv1.Node {
	return []apiv1.Node{
		{
			TypeMeta:   metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{Name: "kube-node-3"},
		},
	}
}

func fakeZeroNode() []apiv1.Node {
	return []apiv1.Node{}
}
