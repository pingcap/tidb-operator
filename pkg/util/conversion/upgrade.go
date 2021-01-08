package conversion

import (
	kruisev1beta1 "github.com/openkruise/kruise-api/apps/v1beta1"
	kruiseclientset "github.com/openkruise/kruise-api/client/clientset/versioned"
	asappsv1 "github.com/pingcap/advanced-statefulset/client/apis/apps/v1"
	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	asclientset "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

const (
	// UpgradeToAdvancedStatefulSetAnn represents the annotation key used to
	// help migration to Advanced StatefulSet
	UpgradeToKruiseAstsAnn = "apps.pingcap.com/upgrade-to-kruise-asts"
)

// UpgradeFromSts upgrades Kubernetes builtin StatefulSet to Kruise Advanced StatefulSet.
//
// Basic procedure:
//
// - remove sts selector labels from controller revisions and set a special annotation for Advanced StatefulSet (can be skipped if Kubernetes cluster has http://issues.k8s.io/84982 fixed)
// - create advanced sts
// - delete sts with DeletePropagationOrphan policy
func UpgradeFromSts(c clientset.Interface, kruiseCli kruiseclientset.Interface, sts *appsv1.StatefulSet) (*kruisev1beta1.StatefulSet, error) {
	selector, err := metav1.LabelSelectorAsSelector(sts.Spec.Selector)
	if err != nil {
		return nil, err
	}
	// It's important to empty statefulset selector labels,
	// otherwise sts will adopt it again on delete event and then
	// GC will delete revisions because they are not orphans.
	// https://github.com/kubernetes/kubernetes/issues/84982
	revisionListOptions := metav1.ListOptions{LabelSelector: selector.String()}
	oldRevisionList, err := c.AppsV1().ControllerRevisions(sts.Namespace).List(revisionListOptions)
	if err != nil {
		return nil, err
	}
	for _, revision := range oldRevisionList.Items {
		for key := range sts.Spec.Selector.MatchLabels {
			delete(revision.Labels, key)
		}
		revision.Labels[UpgradeToKruiseAstsAnn] = sts.Name
		_, err = c.AppsV1().ControllerRevisions(revision.Namespace).Update(&revision)
		if err != nil {
			return nil, err
		}
	}
	klog.V(2).Infof("Succesfully marked all controller revisions (%d) of StatefulSet %s/%s", len(oldRevisionList.Items), sts.Namespace, sts.Name)

	// Create or Update
	asts, err := kruiseCli.AppsV1beta1().StatefulSets(sts.Namespace).Get(sts.Name, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}
	notFound := apierrors.IsNotFound(err)
	upgradedSts, err := FromBuiltinStatefulSet(sts)
	if err != nil {
		return nil, err
	}
	if notFound {
		asts = upgradedSts.DeepCopy()
		// https://github.com/kubernetes/apiserver/blob/kubernetes-1.16.0/pkg/storage/etcd3/store.go#L141-L143
		asts.ObjectMeta.ResourceVersion = ""
		// https://kubernetes.io/docs/reference/using-api/api-concepts/#server-side-apply
		// old ManagedFields belongs to apps/v1 and kube-controller-manager,
		// nil it and the ownership will be transferred to
		// advanced-statefulset-controller-manager
		asts.ObjectMeta.ManagedFields = nil
		asts, err = kruiseCli.AppsV1beta1().StatefulSets(asts.Namespace).Create(asts)
		if err != nil {
			return nil, err
		}
		klog.V(2).Infof("Succesfully created the new Advanced StatefulSet %s/%s", asts.Namespace, asts.Name)
	} else {
		asts.Spec = upgradedSts.Spec
		asts, err = kruiseCli.AppsV1beta1().StatefulSets(asts.Namespace).Update(asts)
		if err != nil {
			return nil, err
		}
		klog.V(2).Infof("Succesfully updated the Advanced StatefulSet %s/%s", asts.Namespace, asts.Name)
	}

	// Status must be updated via UpdateStatus
	asts.Status = upgradedSts.Status
	asts, err = kruiseCli.AppsV1beta1().StatefulSets(asts.Namespace).UpdateStatus(asts)
	if err != nil {
		return nil, err
	}

	// At the last, delete the builtin StatefulSet
	policy := metav1.DeletePropagationOrphan
	err = c.AppsV1().StatefulSets(sts.Namespace).Delete(sts.Name, &metav1.DeleteOptions{
		PropagationPolicy: &policy,
	})
	if err != nil && !apierrors.IsNotFound(err) {
		// ignore IsNotFound error
		return nil, err
	}
	klog.V(2).Infof("Succesfully deleted the old builtin StatefulSet %s/%s", sts.Namespace, sts.Name)

	// After old StatefulSet was deleted actually from etcd, we sync the label to controller revisions
	return asts, nil
}

// UpgradeFromAsts upgrades PingCAP Advanced StatefulSet to Kruise Advanced StatefulSet.
// Its procedure is basically the same as UpgradeFromSts
func UpgradeFromAsts(c clientset.Interface, asCli asclientset.Interface, kruiseCli kruiseclientset.Interface, asts *asappsv1.StatefulSet) (*kruisev1beta1.StatefulSet, error) {
	selector, err := metav1.LabelSelectorAsSelector(asts.Spec.Selector)
	if err != nil {
		return nil, err
	}
	// It's important to empty statefulset selector labels,
	// otherwise sts will adopt it again on delete event and then
	// GC will delete revisions because they are not orphans.
	// https://github.com/kubernetes/kubernetes/issues/84982
	revisionListOptions := metav1.ListOptions{LabelSelector: selector.String()}
	oldRevisionList, err := c.AppsV1().ControllerRevisions(asts.Namespace).List(revisionListOptions)
	if err != nil {
		return nil, err
	}
	for _, revision := range oldRevisionList.Items {
		for key := range asts.Spec.Selector.MatchLabels {
			delete(revision.Labels, key)
		}
		revision.Labels[UpgradeToKruiseAstsAnn] = asts.Name
		_, err = c.AppsV1().ControllerRevisions(revision.Namespace).Update(&revision)
		if err != nil {
			return nil, err
		}
	}
	klog.V(2).Infof("Succesfully marked all controller revisions (%d) of StatefulSet %s/%s", len(oldRevisionList.Items), asts.Namespace, asts.Name)

	// Create or Update
	kruiseAsts, err := kruiseCli.AppsV1beta1().StatefulSets(asts.Namespace).Get(asts.Name, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}
	notFound := apierrors.IsNotFound(err)

	tmpSts, err := helper.ToBuiltinStatefulSet(asts)
	if err != nil {
		return nil, err
	}
	upgradedKruiseAsts, err := FromBuiltinStatefulSet(tmpSts)
	if err != nil {
		return nil, err
	}
	if notFound {
		kruiseAsts = upgradedKruiseAsts.DeepCopy()
		// https://github.com/kubernetes/apiserver/blob/kubernetes-1.16.0/pkg/storage/etcd3/store.go#L141-L143
		kruiseAsts.ObjectMeta.ResourceVersion = ""
		// https://kubernetes.io/docs/reference/using-api/api-concepts/#server-side-apply
		// old ManagedFields belongs to apps/v1 and kube-controller-manager,
		// nil it and the ownership will be transferred to
		// advanced-statefulset-controller-manager
		kruiseAsts.ObjectMeta.ManagedFields = nil
		kruiseAsts, err = kruiseCli.AppsV1beta1().StatefulSets(kruiseAsts.Namespace).Create(kruiseAsts)
		if err != nil {
			return nil, err
		}
		klog.V(2).Infof("Succesfully created the new Advanced StatefulSet %s/%s", kruiseAsts.Namespace, kruiseAsts.Name)
	} else {
		kruiseAsts.Spec = upgradedKruiseAsts.Spec
		kruiseAsts, err = kruiseCli.AppsV1beta1().StatefulSets(kruiseAsts.Namespace).Update(kruiseAsts)
		if err != nil {
			return nil, err
		}
		klog.V(2).Infof("Succesfully updated the Advanced StatefulSet %s/%s", kruiseAsts.Namespace, kruiseAsts.Name)
	}

	// Status must be updated via UpdateStatus
	kruiseAsts.Status = upgradedKruiseAsts.Status
	kruiseAsts, err = kruiseCli.AppsV1beta1().StatefulSets(kruiseAsts.Namespace).UpdateStatus(kruiseAsts)
	if err != nil {
		return nil, err
	}

	// At the last, delete the PingCAP Advanced StatefulSet
	policy := metav1.DeletePropagationOrphan
	err = asCli.AppsV1().StatefulSets(asts.Namespace).Delete(asts.Name, &metav1.DeleteOptions{
		PropagationPolicy: &policy,
	})
	if err != nil && !apierrors.IsNotFound(err) {
		// ignore IsNotFound error
		return nil, err
	}
	klog.V(2).Infof("Succesfully deleted the old Pingcap Advanced StatefulSet %s/%s", asts.Namespace, asts.Name)
	return kruiseAsts, nil
}
