package util

import (
	"strings"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/new-operator/pkg/apis/pingcap.com/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/kubelet/apis"
)

var (
	// weight is in range 1-100
	topologySchedulingWeight = map[string]int32{
		"region":           10,
		"zone":             20,
		"rack":             40,
		apis.LabelHostname: 80,
	}
)

const (
	// StoreUpState is state when tikv store is normal
	StoreUpState = "Up"
	// StoreOfflineState is state when tikv store is offline
	StoreOfflineState = "Offline"
	// StoreDownState is state when tikv store is down
	StoreDownState = "Down"
	// StoreTombstoneState is state when tikv store is tombstone
	StoreTombstoneState = "Tombstone"
)

// AntiAffinityForPod creates a PodAntiAffinity with antiLabels
func AntiAffinityForPod(namespace string, antiLabels map[string]string) *corev1.PodAntiAffinity {
	terms := []corev1.WeightedPodAffinityTerm{}
	for key, weight := range topologySchedulingWeight {
		term := corev1.WeightedPodAffinityTerm{
			Weight: weight,
			PodAffinityTerm: corev1.PodAffinityTerm{
				LabelSelector: &metav1.LabelSelector{MatchLabels: antiLabels},
				TopologyKey:   key,
				Namespaces:    []string{namespace},
			},
		}
		terms = append(terms, term)
	}
	return &corev1.PodAntiAffinity{
		PreferredDuringSchedulingIgnoredDuringExecution: terms,
	}
}

// AffinityForNodeSelector creates an Affinity for NodeSelector
// Externally we use NodeSelector for simplicity,
// while internally we convert it to affinity which can express complex scheduling rules
func AffinityForNodeSelector(namespace string, required bool, antiLabels, selector map[string]string) *corev1.Affinity {
	if selector == nil {
		return nil
	}
	affinity := &corev1.Affinity{}
	if antiLabels != nil {
		affinity.PodAntiAffinity = AntiAffinityForPod(namespace, antiLabels)
	}
	requiredTerms := []corev1.NodeSelectorTerm{}
	if required { // all nodeSelectors are required
		var exps []corev1.NodeSelectorRequirement
		for key, val := range selector {
			requirement := corev1.NodeSelectorRequirement{
				Key:      key,
				Operator: corev1.NodeSelectorOpIn,
				Values:   strings.Split(val, ","),
			}
			// NodeSelectorRequirement in the same MatchExpressions are ANDed otherwise ORed
			exps = append(exps, requirement)
		}
		requiredTerms = append(requiredTerms, corev1.NodeSelectorTerm{MatchExpressions: exps})
		affinity.NodeAffinity = &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: requiredTerms,
			},
		}
		return affinity
	}

	preferredTerms := []corev1.PreferredSchedulingTerm{}
	exps := []corev1.NodeSelectorRequirement{}
	for key, val := range selector {
		// region,zone,rack,host are preferred labels, others are must match labels
		if weight, ok := topologySchedulingWeight[key]; ok {
			if val == "" {
				continue
			}
			values := strings.Split(val, ",")
			t := corev1.PreferredSchedulingTerm{
				Weight: weight,
				Preference: corev1.NodeSelectorTerm{
					MatchExpressions: []corev1.NodeSelectorRequirement{
						{
							Key:      key,
							Operator: corev1.NodeSelectorOpIn,
							Values:   values,
						},
					},
				},
			}
			preferredTerms = append(preferredTerms, t)
		} else {
			requirement := corev1.NodeSelectorRequirement{
				Key:      key,
				Operator: corev1.NodeSelectorOpIn,
				Values:   []string{val},
			}
			// NodeSelectorRequirement in the same MatchExpressions are ANDed otherwise ORed
			exps = append(exps, requirement)
		}
	}
	requiredTerms = append(requiredTerms, corev1.NodeSelectorTerm{MatchExpressions: exps})

	affinity.NodeAffinity = &corev1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
			NodeSelectorTerms: requiredTerms,
		},
		PreferredDuringSchedulingIgnoredDuringExecution: preferredTerms,
	}

	return affinity
}

// ResourceRequirement creates ResourceRequirements for MemberSpec
// Optionally pass in a default value
func ResourceRequirement(spec v1.ContainerSpec, defaultRequests ...corev1.ResourceRequirements) corev1.ResourceRequirements {
	rr := corev1.ResourceRequirements{}
	if len(defaultRequests) > 0 {
		defaultRequest := defaultRequests[0]
		rr.Requests = make(map[corev1.ResourceName]resource.Quantity)
		rr.Requests[corev1.ResourceCPU] = defaultRequest.Requests[corev1.ResourceCPU]
		rr.Requests[corev1.ResourceMemory] = defaultRequest.Requests[corev1.ResourceMemory]
		rr.Limits = make(map[corev1.ResourceName]resource.Quantity)
		rr.Limits[corev1.ResourceCPU] = defaultRequest.Limits[corev1.ResourceCPU]
		rr.Limits[corev1.ResourceMemory] = defaultRequest.Limits[corev1.ResourceMemory]
	}
	if spec.Requests != nil {
		if rr.Requests == nil {
			rr.Requests = make(map[corev1.ResourceName]resource.Quantity)
		}
		if q, err := resource.ParseQuantity(spec.Requests.CPU); err != nil {
			glog.Errorf("failed to parse CPU resource %s to quantity: %v", spec.Requests.CPU, err)
		} else {
			rr.Requests[corev1.ResourceCPU] = q
		}
		if q, err := resource.ParseQuantity(spec.Requests.Memory); err != nil {
			glog.Errorf("failed to parse memory resource %s to quantity: %v", spec.Requests.Memory, err)
		} else {
			rr.Requests[corev1.ResourceMemory] = q
		}
	}
	if spec.Limits != nil {
		if rr.Limits == nil {
			rr.Limits = make(map[corev1.ResourceName]resource.Quantity)
		}
		if q, err := resource.ParseQuantity(spec.Limits.CPU); err != nil {
			glog.Errorf("failed to parse CPU resource %s to quantity: %v", spec.Limits.CPU, err)
		} else {
			rr.Limits[corev1.ResourceCPU] = q
		}
		if q, err := resource.ParseQuantity(spec.Limits.Memory); err != nil {
			glog.Errorf("failed to parse memory resource %s to quantity: %v", spec.Limits.Memory, err)
		} else {
			rr.Limits[corev1.ResourceMemory] = q
		}
	}
	return rr
}
