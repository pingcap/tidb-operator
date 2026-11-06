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

package validation

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"slices"
	"testing"

	celgo "github.com/google/cel-go/cel"
	celtypes "github.com/google/cel-go/common/types"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apiserver/pkg/admission"
	admissioncel "k8s.io/apiserver/pkg/admission/plugin/cel"
	celconfig "k8s.io/apiserver/pkg/apis/cel"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/cel/environment"
)

type securityExpression string

func (s securityExpression) GetExpression() string    { return string(s) }
func (securityExpression) ReturnTypes() []*celgo.Type { return []*celgo.Type{celgo.BoolType} }
func renderSecurityChart(t *testing.T, args ...string) []unstructured.Unstructured {
	t.Helper()
	helm, err := exec.LookPath("helm")
	if err != nil {
		t.Skip("helm required for chart security tests")
	}
	command := []string{"template", "security-test", "../../charts/tidb-operator", "--namespace", "operator-system"}
	out, err := exec.CommandContext(context.Background(), helm, append(command, args...)...).CombinedOutput()
	require.NoError(t, err, string(out))
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(out), 4096)
	var result []unstructured.Unstructured
	for {
		var object unstructured.Unstructured
		err := decoder.Decode(&object)
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		if object.GetKind() != "" {
			result = append(result, object)
		}
	}
	return result
}
func TestSecurityChartNamespaceGrants(t *testing.T) {
	objects := renderSecurityChart(t, "--set", "watchNamespaces={tenant-a,tenant-b}", "--set", "appendReleaseSuffix=true", "--set", "skipCRDsManagement=true")
	bindings := map[string]bool{}
	for _, object := range objects {
		if object.GetKind() == "Deployment" {
			containers, _, err := unstructured.NestedSlice(object.Object, "spec", "template", "spec", "containers")
			require.NoError(t, err)
			args, _, err := unstructured.NestedStringSlice(containers[0].(map[string]any), "args")
			require.NoError(t, err)
			require.Contains(t, args, "--watch-namespaces=tenant-a,tenant-b")
		}
		if object.GetKind() == "Role" && object.GetName() == "security-test:tidb-operator-leader-election" {
			rules, _, _ := unstructured.NestedSlice(object.Object, "rules")
			found := false
			for _, r := range rules {
				rule := r.(map[string]any)
				resources, _, _ := unstructured.NestedStringSlice(rule, "resources")
				if slices.Contains(resources, "events") {
					verbs, _, _ := unstructured.NestedStringSlice(rule, "verbs")
					require.Contains(t, verbs, "create")
					require.Contains(t, verbs, "patch")
					found = true
				}
			}
			require.True(t, found)
		}
		if object.GetKind() == "ClusterRole" {
			rules, _, err := unstructured.NestedSlice(object.Object, "rules")
			require.NoError(t, err)
			for _, r := range rules {
				rule := r.(map[string]any)
				verbs, _, _ := unstructured.NestedStringSlice(rule, "verbs")
				require.NotContains(t, verbs, "*")
				resources, _, _ := unstructured.NestedStringSlice(rule, "resources")
				require.NotContains(t, resources, "*")
				if object.GetName() == "security-test:tidb-operator-cluster" {
					require.NotContains(t, resources, "secrets")
					require.NotContains(t, resources, "roles")
					require.NotContains(t, resources, "rolebindings")
				}
			}
		}
		if object.GetKind() == "RoleBinding" || object.GetKind() == "ClusterRoleBinding" {
			role, _, _ := unstructured.NestedString(object.Object, "roleRef", "name")
			if role == "security-test:tidb-operator" {
				require.Equal(t, "RoleBinding", object.GetKind())
				bindings[object.GetNamespace()] = true
			}
			subjects, _, _ := unstructured.NestedSlice(object.Object, "subjects")
			for _, subject := range subjects {
				require.Equal(t, "tidb-operator-security-test", subject.(map[string]any)["name"])
			}
		}
	}
	require.Equal(t, map[string]bool{"tenant-a": true, "tenant-b": true}, bindings)
}
func TestWorkloadAuthorityPolicy(t *testing.T) {
	var expression string
	var actions []string
	for _, object := range renderSecurityChart(t) {
		if object.GetKind() == "ValidatingAdmissionPolicyBinding" {
			name, _, _ := unstructured.NestedString(object.Object, "spec", "policyName")
			if name == "security-test.workload-authority.pingcap.com" {
				actions, _, _ = unstructured.NestedStringSlice(object.Object, "spec", "validationActions")
			}
		}
		if object.GetKind() == "ValidatingAdmissionPolicy" && object.GetName() == "security-test.workload-authority.pingcap.com" {
			validations, _, err := unstructured.NestedSlice(object.Object, "spec", "validations")
			require.NoError(t, err)
			expression = validations[0].(map[string]any)["expression"].(string)
		}
	}
	require.NotEmpty(t, expression)
	require.Equal(t, []string{"Deny"}, actions)
	filter := admissioncel.NewFilterCompiler(environment.MustBaseEnvSet(environment.DefaultCompatibilityVersion(), true)).Compile([]admissioncel.ExpressionAccessor{securityExpression(expression)}, admissioncel.OptionalVariableDeclarations{HasAuthorizer: true, StrictCost: true}, environment.NewExpressions)
	require.Empty(t, filter.CompilationErrors())
	for _, allow := range []bool{false, true} {
		t.Run(map[bool]string{false: "CR-only writer denied", true: "workload administrator allowed"}[allow], func(t *testing.T) {
			gvk := schema.GroupVersionKind{Group: "br.pingcap.com", Version: "v1alpha1", Kind: "Backup"}
			gvr := schema.GroupVersionResource{Group: "br.pingcap.com", Version: "v1alpha1", Resource: "backups"}
			object := &unstructured.Unstructured{Object: map[string]any{"apiVersion": "br.pingcap.com/v1alpha1", "kind": "Backup", "metadata": map[string]any{"name": "backup", "namespace": "tenant"}}}
			attrs := admission.NewAttributesRecord(object, nil, gvk, "tenant", "backup", gvr, "", admission.Create, &metav1.CreateOptions{}, false, &user.DefaultInfo{Name: "writer"})
			versioned := &admission.VersionedAttributes{Attributes: attrs, VersionedKind: gvk, VersionedObject: object}
			calls := 0
			auth := authorizer.AuthorizerFunc(func(_ context.Context, a authorizer.Attributes) (authorizer.Decision, string, error) {
				calls++
				require.Equal(t, "tenant", a.GetNamespace())
				require.Equal(t, "pods", a.GetResource())
				require.Equal(t, "create", a.GetVerb())
				require.Empty(t, a.GetAPIGroup())
				require.Equal(t, "writer", a.GetUser().GetName())
				if allow {
					return authorizer.DecisionAllow, "", nil
				}
				return authorizer.DecisionNoOpinion, "", nil
			})
			results, _, err := filter.ForInput(context.Background(), versioned, &admissionv1.AdmissionRequest{Namespace: "tenant", Name: "backup", Operation: admissionv1.Create}, admissioncel.OptionalVariableBindings{Authorizer: auth}, nil, celconfig.RuntimeCELCostBudget)
			require.NoError(t, err)
			require.Len(t, results, 1)
			require.NoError(t, results[0].Error)
			require.Equal(t, celtypes.Bool(allow), results[0].EvalResult)
			require.Equal(t, 1, calls)
		})
	}
}

func TestClusterMetricsGrant(t *testing.T) {
	for _, args := range [][]string{nil, {"--set", "watchNamespaces={tenant-a}"}} {
		found := false
		for _, obj := range renderSecurityChart(t, args...) {
			if obj.GetKind() == "ClusterRole" && obj.GetName() == "security-test:tidb-operator-cluster" {
				rules, _, _ := unstructured.NestedSlice(obj.Object, "rules")
				for _, r := range rules {
					urls, _, _ := unstructured.NestedStringSlice(r.(map[string]any), "nonResourceURLs")
					if slices.Contains(urls, "/metrics") {
						verbs, _, _ := unstructured.NestedStringSlice(r.(map[string]any), "verbs")
						require.Contains(t, verbs, "get")
						found = true
					}
				}
			}
		}
		require.True(t, found, "both install modes must authorize startup metrics discovery")
	}
}

// TestGarbageCollectorMatch checks the narrow control-plane update exception.
func TestGarbageCollectorMatch(t *testing.T) {
	var expression string
	for _, obj := range renderSecurityChart(t) {
		if obj.GetKind() == "ValidatingAdmissionPolicy" && obj.GetName() == "security-test.workload-authority.pingcap.com" {
			conditions, _, _ := unstructured.NestedSlice(obj.Object, "spec", "matchConditions")
			expression = conditions[0].(map[string]any)["expression"].(string)
		}
	}
	require.NotEmpty(t, expression)
	filter := admissioncel.NewFilterCompiler(environment.MustBaseEnvSet(environment.DefaultCompatibilityVersion(), true)).Compile([]admissioncel.ExpressionAccessor{securityExpression(expression)}, admissioncel.OptionalVariableDeclarations{StrictCost: true}, environment.NewExpressions)
	require.Empty(t, filter.CompilationErrors())
	for _, principal := range []string{"system:serviceaccount:kube-system:generic-garbage-collector", "system:kube-controller-manager", "writer"} {
		for _, change := range []string{"orphan", "unblock", "gc finalizer", "annotation", "spec", "owner uid", "custom finalizer", "create"} {
			t.Run(principal+"/"+change, func(t *testing.T) {
				old := &unstructured.Unstructured{Object: map[string]any{"apiVersion": "core.pingcap.com/v1alpha1", "kind": "TiDB", "metadata": map[string]any{"name": "db", "namespace": "tenant", "ownerReferences": []any{map[string]any{"apiVersion": "core.pingcap.com/v1alpha1", "kind": "TiDBGroup", "name": "group", "uid": "group-uid", "controller": true, "blockOwnerDeletion": true}}, "finalizers": []any{"orphan", "foregroundDeletion", "custom"}}, "spec": map[string]any{"version": "v8.5.0"}}}
				obj := old.DeepCopy()
				operation := admission.Update
				switch change {
				case "orphan":
					delete(obj.Object["metadata"].(map[string]any), "ownerReferences")
				case "unblock":
					obj.Object["metadata"].(map[string]any)["ownerReferences"].([]any)[0].(map[string]any)["blockOwnerDeletion"] = false
				case "gc finalizer":
					obj.Object["metadata"].(map[string]any)["finalizers"] = []any{"custom"}
				case "custom finalizer":
					obj.Object["metadata"].(map[string]any)["finalizers"] = []any{"orphan", "foregroundDeletion"}
				case "annotation":
					obj.SetAnnotations(map[string]string{"br.pingcap.com/allowed-backup-namespaces": "other"})
				case "spec":
					obj.Object["spec"].(map[string]any)["version"] = "v9.0.0"
				case "owner uid":
					obj.Object["metadata"].(map[string]any)["ownerReferences"].([]any)[0].(map[string]any)["uid"] = "replacement"
				case "create":
					operation = admission.Create
				}
				gvk := schema.GroupVersionKind{Group: "core.pingcap.com", Version: "v1alpha1", Kind: "TiDB"}
				gvr := schema.GroupVersionResource{Group: "core.pingcap.com", Version: "v1alpha1", Resource: "tidbs"}
				attrs := admission.NewAttributesRecord(obj, old, gvk, "tenant", "db", gvr, "", operation, &metav1.UpdateOptions{}, false, &user.DefaultInfo{Name: principal})
				versioned := &admission.VersionedAttributes{Attributes: attrs, VersionedKind: gvk, VersionedObject: obj, VersionedOldObject: old}
				request := admissioncel.CreateAdmissionRequest(attrs, metav1.GroupVersionResource(gvr), metav1.GroupVersionKind(gvk))
				results, _, err := filter.ForInput(context.Background(), versioned, request, admissioncel.OptionalVariableBindings{}, nil, celconfig.RuntimeCELCostBudget)
				require.NoError(t, err)
				require.Len(t, results, 1)
				require.NoError(t, results[0].Error)
				exempt := principal != "writer" && (change == "orphan" || change == "unblock" || change == "gc finalizer")
				require.Equal(t, celtypes.Bool(!exempt), results[0].EvalResult)
			})
		}
	}
}
