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

// UpgradeFromSts upgrades Kubernetes builtin StatefulSet to Kruise Advanced StatefulSet.
//
// Basic procedure:
//
// - remove sts selector labels from controller revisions
// (can be skipped if Kubernetes cluster has http://issues.k8s.io/84982 fixed)
// - create kruise advanced sts
// - delete sts with DeletePropagationOrphan policy
func UpgradeFromSts(c clientset.Interface, kruiseCli kruiseclientset.Interface, sts *appsv1.StatefulSet) (*kruisev1beta1.StatefulSet, error) {
	oldRevisionList, err := getControllerRevisionList(c, sts.Spec.Selector, sts.Namespace)
	if err != nil {
		return nil, err
	}

	if err = emptyControllerRevisionLabel(c, oldRevisionList, sts.Spec.Selector.MatchLabels); err != nil {
		return nil, err
	}
	klog.V(2).Infof("Succesfully empty labels of all controller revisions (%d) of StatefulSet %s/%s", len(oldRevisionList.Items), sts.Namespace, sts.Name)

	// Create or Update Kruise StatefulSet
	kruiseAsts, err := createOrUpdateKruiseStatefulSet(kruiseCli, sts.Namespace, sts.Name, sts)
	if err != nil {
		return nil, err
	}

	// delete the builtin StatefulSet
	policy := metav1.DeletePropagationOrphan
	err = c.AppsV1().StatefulSets(sts.Namespace).Delete(sts.Name, &metav1.DeleteOptions{
		PropagationPolicy: &policy,
	})
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}
	klog.V(2).Infof("Succesfully deleted the old builtin StatefulSet %s/%s", sts.Namespace, sts.Name)

	// We wait for old StatefulSet to be deleted actually from etcd, then write back the labels to controller revisions
	// just like what is done in Advanced-Statefulset: https://github.com/pingcap/advanced-statefulset/blob/master/pkg/controller/statefulset/stateful_set.go#L358-L363
	err = retryUntilStsDeleted(c, oldRevisionList, sts.Namespace, sts.Name, func() error {
		_, getErr := c.AppsV1().StatefulSets(sts.Namespace).Get(sts.Name, metav1.GetOptions{})
		return getErr
	})

	return kruiseAsts, err
}

// UpgradeFromAsts upgrades PingCAP Advanced StatefulSet to Kruise Advanced StatefulSet.
// The procedure is basically the same as UpgradeFromSts
func UpgradeFromAsts(c clientset.Interface, asCli asclientset.Interface, kruiseCli kruiseclientset.Interface, asts *asappsv1.StatefulSet) (*kruisev1beta1.StatefulSet, error) {
	oldRevisionList, err := getControllerRevisionList(c, asts.Spec.Selector, asts.Namespace)
	if err != nil {
		return nil, err
	}

	if err = emptyControllerRevisionLabel(c, oldRevisionList, asts.Spec.Selector.MatchLabels); err != nil {
		return nil, err
	}
	klog.V(2).Infof("Succesfully empty labels of all controller revisions (%d) of StatefulSet %s/%s", len(oldRevisionList.Items), asts.Namespace, asts.Name)

	tmpSts, err := helper.ToBuiltinStatefulSet(asts)
	if err != nil {
		return nil, err
	}

	// Create or Update Kruise StatefulSet
	kruiseAsts, err := createOrUpdateKruiseStatefulSet(kruiseCli, asts.Namespace, asts.Name, tmpSts)
	if err != nil {
		return nil, err
	}

	// delete the PingCAP Advanced StatefulSet
	policy := metav1.DeletePropagationOrphan
	err = asCli.AppsV1().StatefulSets(asts.Namespace).Delete(asts.Name, &metav1.DeleteOptions{
		PropagationPolicy: &policy,
	})
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}
	klog.V(2).Infof("Succesfully deleted the old Pingcap Advanced StatefulSet %s/%s", asts.Namespace, asts.Name)

	// We wait for old StatefulSet to be deleted actually from etcd, then write back the labels to controller revisions
	// Just like what is done here: https://github.com/pingcap/advanced-statefulset/blob/master/pkg/controller/statefulset/stateful_set.go#L331-L341
	err = retryUntilStsDeleted(c, oldRevisionList, asts.Namespace, asts.Name, func() error {
		_, getErr := asCli.AppsV1().StatefulSets(asts.Namespace).Get(asts.Name, metav1.GetOptions{})
		return getErr
	})

	return kruiseAsts, err
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

// It's important to empty statefulset selector labels,
// otherwise sts will adopt it again on delete event and then
// GC will delete revisions because they are not orphans.
// https://github.com/kubernetes/kubernetes/issues/84982
func emptyControllerRevisionLabel(c clientset.Interface, oldRevisionList *appsv1.ControllerRevisionList, matchLabels map[string]string) error {
	for _, r := range oldRevisionList.Items {
		revision := r.DeepCopy()
		for key := range matchLabels {
			delete(revision.Labels, key)
		}
		_, err := c.AppsV1().ControllerRevisions(revision.Namespace).Update(revision)
		if err != nil {
			return err
		}
	}
	return nil
}

func getControllerRevisionList(c clientset.Interface, labelSelector *metav1.LabelSelector, ns string) (*appsv1.ControllerRevisionList, error) {
	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		return nil, err
	}

	revisionListOptions := metav1.ListOptions{LabelSelector: selector.String()}
	return c.AppsV1().ControllerRevisions(ns).List(revisionListOptions)
}

func createOrUpdateKruiseStatefulSet(kruiseCli kruiseclientset.Interface, ns, name string, sts *appsv1.StatefulSet) (*kruisev1beta1.StatefulSet, error) {
	kruiseAsts, err := kruiseCli.AppsV1beta1().StatefulSets(ns).Get(name, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}
	notFound := apierrors.IsNotFound(err)

	upgradedKruiseAsts, err := FromBuiltinStatefulSet(sts)
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
		// apps.kruise.io/v1beta1
		kruiseAsts.ObjectMeta.ManagedFields = nil
		kruiseAsts, err = kruiseCli.AppsV1beta1().StatefulSets(kruiseAsts.Namespace).Create(kruiseAsts)
		if err != nil {
			return nil, err
		}
		klog.V(2).Infof("Succesfully created the new Kruise Advanced StatefulSet %s/%s", kruiseAsts.Namespace, kruiseAsts.Name)
	} else {
		kruiseAsts.Spec = upgradedKruiseAsts.Spec
		kruiseAsts, err = kruiseCli.AppsV1beta1().StatefulSets(kruiseAsts.Namespace).Update(kruiseAsts)
		if err != nil {
			return nil, err
		}
		klog.V(2).Infof("Succesfully updated Kruise Advanced StatefulSet %s/%s", kruiseAsts.Namespace, kruiseAsts.Name)
	}

	// Status must be updated via UpdateStatus
	kruiseAsts.Status = upgradedKruiseAsts.Status
	kruiseAsts, err = kruiseCli.AppsV1beta1().StatefulSets(kruiseAsts.Namespace).UpdateStatus(kruiseAsts)
	if err != nil {
		return nil, err
	}

	return kruiseAsts, nil
}

func retryUntilStsDeleted(c clientset.Interface, oldRevisionList *appsv1.ControllerRevisionList, ns, name string, getStsFn func() error) (err error) {
	for i := 0; i < 10; i++ {
		err = getStsFn()
		if err != nil && apierrors.IsNotFound(err) {
			if err = syncControllerRevision(c, oldRevisionList); err != nil {
				return err
			}
			klog.V(2).Infof("successfully sync controller revisions owned by %s/%s", ns, name)
			return nil
		}
		time.Sleep(6 * time.Second)
	}
	return fmt.Errorf("statefulset %s/%s is not deleted from etcd yet", ns, name)
}
