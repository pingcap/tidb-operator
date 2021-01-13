package conversion

import (
	"fmt"
	"time"

	"k8s.io/client-go/util/retry"

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
	for _, r := range oldRevisionList.Items {
		revision := r.DeepCopy()
		for key := range sts.Spec.Selector.MatchLabels {
			delete(revision.Labels, key)
		}
		revision.Labels[UpgradeToKruiseAstsAnn] = sts.Name
		_, err = c.AppsV1().ControllerRevisions(revision.Namespace).Update(revision)
		if err != nil {
			return nil, err
		}
	}
	klog.V(2).Infof("Succesfully marked all controller revisions (%d) of StatefulSet %s/%s", len(oldRevisionList.Items), sts.Namespace, sts.Name)

	// Create or Update
	kAsts, err := kruiseCli.AppsV1beta1().StatefulSets(sts.Namespace).Get(sts.Name, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}
	notFound := apierrors.IsNotFound(err)
	upgradedSts, err := FromBuiltinStatefulSet(sts)
	if err != nil {
		return nil, err
	}
	if notFound {
		kAsts = upgradedSts.DeepCopy()
		// https://github.com/kubernetes/apiserver/blob/kubernetes-1.16.0/pkg/storage/etcd3/store.go#L141-L143
		kAsts.ObjectMeta.ResourceVersion = ""
		// https://kubernetes.io/docs/reference/using-api/api-concepts/#server-side-apply
		// old ManagedFields belongs to apps/v1 and kube-controller-manager,
		// nil it and the ownership will be transferred to
		// advanced-statefulset-controller-manager
		kAsts.ObjectMeta.ManagedFields = nil
		kAsts, err = kruiseCli.AppsV1beta1().StatefulSets(kAsts.Namespace).Create(kAsts)
		if err != nil {
			return nil, err
		}
		klog.V(2).Infof("Succesfully created the new Advanced StatefulSet %s/%s", kAsts.Namespace, kAsts.Name)
	} else {
		kAsts.Spec = upgradedSts.Spec
		kAsts, err = kruiseCli.AppsV1beta1().StatefulSets(kAsts.Namespace).Update(kAsts)
		if err != nil {
			return nil, err
		}
		klog.V(2).Infof("Succesfully updated the Advanced StatefulSet %s/%s", kAsts.Namespace, kAsts.Name)
	}

	// Status must be updated via UpdateStatus
	kAsts.Status = upgradedSts.Status
	kAsts, err = kruiseCli.AppsV1beta1().StatefulSets(kAsts.Namespace).UpdateStatus(kAsts)
	if err != nil {
		return nil, err
	}

	// delete the builtin StatefulSet
	policy := metav1.DeletePropagationOrphan
	err = c.AppsV1().StatefulSets(sts.Namespace).Delete(sts.Name, &metav1.DeleteOptions{
		PropagationPolicy: &policy,
	})
	if err != nil && !apierrors.IsNotFound(err) {
		// ignore IsNotFound error
		return nil, err
	}
	klog.V(2).Infof("Succesfully deleted the old builtin StatefulSet %s/%s", sts.Namespace, sts.Name)

	// We wait for old StatefulSet to be deleted actually from etcd, then write back the labels to controller revisions
	// just like what is done in Advanced-Statefulset: https://github.com/pingcap/advanced-statefulset/blob/master/pkg/controller/statefulset/stateful_set.go#L358-L363
	for i := 0; i < 5; i++ {
		_, err = c.AppsV1().StatefulSets(sts.Namespace).Get(sts.Name, metav1.GetOptions{})
		if err != nil && apierrors.IsNotFound(err) {
			if err = syncControllerRevision(c, oldRevisionList); err != nil {
				return nil, err
			}
			klog.V(2).Infof("successfully sync controller revisions owned by %s/%s", sts.Namespace, sts.Name)
			return kAsts, nil
		}
		time.Sleep(time.Duration(6) * time.Second)
	}

	return nil, fmt.Errorf("failed to sync controller revisions: statefulset %s/%s is not deleted from etcd", sts.Namespace, sts.Name)
}

// UpgradeFromAsts upgrades PingCAP Advanced StatefulSet to Kruise Advanced StatefulSet.
// The procedure is basically the same as UpgradeFromSts
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
	for _, r := range oldRevisionList.Items {
		revision := r.DeepCopy()
		for key := range asts.Spec.Selector.MatchLabels {
			delete(revision.Labels, key)
		}
		revision.Labels[UpgradeToKruiseAstsAnn] = asts.Name
		_, err = c.AppsV1().ControllerRevisions(revision.Namespace).Update(revision)
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

	// delete the PingCAP Advanced StatefulSet
	policy := metav1.DeletePropagationOrphan
	err = asCli.AppsV1().StatefulSets(asts.Namespace).Delete(asts.Name, &metav1.DeleteOptions{
		PropagationPolicy: &policy,
	})
	if err != nil && !apierrors.IsNotFound(err) {
		// ignore IsNotFound error
		return nil, err
	}
	klog.V(2).Infof("Succesfully deleted the old Pingcap Advanced StatefulSet %s/%s", asts.Namespace, asts.Name)

	// After old StatefulSet was deleted actually from etcd, we sync the labels to controller revisions
	// Just like what is done here: https://github.com/pingcap/advanced-statefulset/blob/master/pkg/controller/statefulset/stateful_set.go#L331-L341
	for i := 0; i < 5; i++ {
		_, err = asCli.AppsV1().StatefulSets(asts.Namespace).Get(asts.Name, metav1.GetOptions{})
		if err != nil && apierrors.IsNotFound(err) {
			if err = syncControllerRevision(c, oldRevisionList); err != nil {
				return nil, err
			}
			klog.V(2).Infof("successfully sync controller revisions owned by %s/%s", asts.Namespace, asts.Name)
			return kruiseAsts, nil
		}
		time.Sleep(time.Duration(6) * time.Second)
	}

	return nil, fmt.Errorf("failed to sync controller revisions: statefulset %s/%s is not deleted from etcd", asts.Namespace, asts.Name)
}

func syncControllerRevision(c clientset.Interface, oldRevisionList *appsv1.ControllerRevisionList) error {
	for _, revision := range oldRevisionList.Items {

		revision.ObjectMeta.ResourceVersion = ""
		revision.ObjectMeta.OwnerReferences = nil
		revision.ObjectMeta.ManagedFields = nil
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			_, updateErr := c.AppsV1().ControllerRevisions(revision.Namespace).Update(&revision)
			return updateErr
		})
		if err != nil {
			return err
		}
	}
	return nil
}
