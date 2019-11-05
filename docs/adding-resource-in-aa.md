# Adding New Resources in TiDB Aggregated Apiserver

> This guide shows an example about how to add new resources in TiDB Aggregated Apiserver (AA)

## Project Structure

```shell
pkg/apiserver                                           # package root of AA
pkg/apiserver/apis                                      # all apis of AA
pkg/apiserver/apis/{group}/                             # un-versioned apis of a group
pkg/apiserver/apis/{group}/{version}                    # versiond apis of a group
pkg/apiserver/apis/{group}/{version}/{kind}_types.go    # type definition of a certain kind
pkg/apiserver/client                                    # generated go client
pkg/apiserver/cmd                                       # AA entrypoint
pkg/apiserver/openapi                                   # generated openapi definition
pkg/apiserver/storage                                   # kube-apiserver based storage implementation
```

We will create the type definition of a versioned resource in `pkg/apiserver/apis/{group}/{version}/{kind}_types.go`.

The defaults function, conversion from un-versioned to versioned, OpenAPI definition and client of the versioned resource will be generated automatically.

The type definition, REST storage, defaults function, conversion from versioned to un-versioned and client of the un-versioned resources will also be generated automatically.

## Foo Example

Let's add a new resource `foos.v1alpha1.tidb.pingcap.com`:

- `kind`: foos
- `version`: v1alpha1
- `group`: tidb.pingcap.com

1. Create packages:

  ```shell
  $ mkdir -p pkg/apiserver/apis/tidb/v1alpha1
  ```
  
  **By convention, we take the sub domain of the group name as the package name.**

2. Write group name and code-generation markers for all resources under `v1alpha.tidb.pingcap.com`:

  `pkg/apiserver/apis/tidb/v1alpha1/doc.go`

  ```
  // +k8s:openapi-gen=true
  // +k8s:deepcopy-gen=package,register
  // +k8s:conversion-gen=github.com/pingcap/tidb-operator/pkg/apiserver/apis/tidb
  // +k8s:defaulter-gen=TypeMeta
  
  // +groupName=tidb.pingcap.com
  package v1alpha1 // import "github.com/pingcap/tidb-operator/pkg/apiserver/apis/tidb/v1alpha1"
  ```
  
3. Write the type definition and code generation markers for Foo:

  `pkg/apiserver/apis/tidb/v1alpha1/foo_types.go`:
  
  ```go
  // +genclient
  // +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
  
  // Foo
  // +k8s:openapi-gen=true
  // +resource:path=foos
  type Foo struct {
  	metav1.TypeMeta   `json:",inline"`
  	metav1.ObjectMeta `json:"metadata,omitempty"`
  
  	Spec   FooSpec   `json:"spec,omitempty"`
  	Status FooStatus `json:"status,omitempty"`
  }
  
  // FooSpec defines the desired state of Foo
  type FooSpec struct {
  	Replicas int `json:"replicas,omitempty"`
  }
  
  // FooStatus defines the observed state of Foo
  type FooStatus struct {
  	CurrentReplicas int `json:"currentReplicas,omitempty"`
  }
  ```

4. Run code generation:

  ```
  ./hack/code-gen.sh
  ```

