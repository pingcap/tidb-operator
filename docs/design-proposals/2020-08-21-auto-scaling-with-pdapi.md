# Auto-scaling with PD API

## Summary

This document presents a design to support auto-scaling for TiDB and TiKV with PD API.

## Motivation

Users may want to auto-scale TiDB or TiKV during high traffic load and may also want to auto-scale TiKV during high disk usage.

### Goals

* Auto-scale TiKV with CPU and storage metrics
* Auto-scale TiDB with CPU metrics

### Non-Goals

* Auto-scale in TiKV during low disk usage.

## Proposal

### Prerequisites

* The PD API returns the full group information for each API call
* For the same metric, when one group is returned from PD, and if a new group needs to be returned in the following API responses (for example, the user only specifies one resource_type, and then resources in the resource_type are changed), PD should return these two groups in the following responses.

### CR Definition

```
// +k8s:openapi-gen=true
// TidbClusterAutoScaler is the control script's spec
type TidbClusterAutoScaler struct {
    metav1.TypeMeta `json:",inline"`
    // +k8s:openapi-gen=false
    metav1.ObjectMeta `json:"metadata"`
    // Spec describes the state of the TidbClusterAutoScaler
    Spec TidbClusterAutoScalerSpec `json:"spec"`
    // Status describe the status of the TidbClusterAutoScaler
    Status TidbClusterAutoSclaerStatus `json:"status"`
}
// +k8s:openapi-gen=true
// TidbAutoScalerSpec describes the state of the TidbClusterAutoScaler
type TidbClusterAutoScalerSpec struct {
    // TidbClusterRef describe the target TidbCluster
    Cluster TidbClusterRef `json:"cluster"`
    // TiKV represents the auto-scaling spec for tikv
    // +optional
    TiKV *TikvAutoScalerSpec `json:"tikv,omitempty"`
    // TiDB represents the auto-scaling spec for tidb
    // +optional
    TiDB *TidbAutoScalerSpec `json:"tidb,omitempty"`
    // Resources represent the resource type definitions that can be used for TiDB/TiKV
    // +optional
    Resources []AutoResource `json:"resources,omitempty"`
}
// +k8s:openapi-gen=true
// AutoResource describes the resource type definitions
type AutoResource struct {
    // ResourceType identifies a specific resource type
    ResourceType string `json:"resource_type,omitempty"`
    // CPU defines the CPU of this resource type
    CPU resource.Quantity `json:"cpu,omitempty"`
    // Memory defines the memory of this resource type
    Memory resource.Quantity `json:"memory,omitempty"`
    // Storage defines the storage of this resource type
    Storage resource.Quantity `json:"storage,omitempty"`
    // Count defines the max availabel count of this resource type
    Count *int32 `json:"count,omitempty"`
}
// +k8s:openapi-gen=true
// AutoRule describes the rules for auto-scaling with PD API
type AutoRule struct {
    // Max defines the threshold to scale out
    Max *float64 `json:"max,omitempty"`
    // Min defines the threshold to scale in
    Min *float64 `json:"min,omitempty"`
    // ResourceType defines the resource types that can be used for scaling
    ResourceType []string `json:"resource_type,omitempty"`
}
// +k8s:openapi-gen=true
// TikvAutoScalerSpec describes the spec for tikv auto-scaling
type TikvAutoScalerSpec struct {
    BasicAutoScalerSpec `json:",inline"`
}
// +k8s:openapi-gen=true
// TidbAutoScalerSpec describes the spec for tidb auto-scaling
type TidbAutoScalerSpec struct {
    BasicAutoScalerSpec `json:",inline"`
}
// +k8s:openapi-gen=true
// BasicAutoScalerSpec describes the basic spec for auto-scaling
type BasicAutoScalerSpec struct {
    // Rules defines the rules for auto-scaling with PD API
    Rules map[corev1.ResourceName]AutoRule `json:"rules,omitempty"`
    // ScaleInIntervalSeconds represents the duration seconds between each auto-scaling-in
    // If not set, the default ScaleInIntervalSeconds will be set to 500
    // +optional
    ScaleInIntervalSeconds *int32 `json:"scaleInIntervalSeconds,omitempty"`
    // ScaleOutIntervalSeconds represents the duration seconds between each auto-scaling-out
    // If not set, the default ScaleOutIntervalSeconds will be set to 300
    // +optional
    ScaleOutIntervalSeconds *int32 `json:"scaleOutIntervalSeconds,omitempty"`
    // External configures the external endpoint that auto-scaler controller
    // can query to fetch the recommended replicas for TiKV/TiDB and
    // the max replicas that can be scaled
    // +optional
    External *ExternalConfig `json:"external,omitempty"`
}
type ExternalConfig struct {
    // ExternalEndpoint makes the auto-scaler controller able to query the
    // external service to fetch the recommended replicas for TiKV/TiDB
    // +optional
    Endpoint *ExternalEndpoint `json:"endpoint"`
    // maxReplicas is the upper limit for the number of replicas to which the autoscaler can scale out.
    // It cannot be less than minReplicas.
    MaxReplicas int32 `json:"maxReplicas"`
}
```
### Defaulting & Validation

`xxxx` is `tikv` or `tidb` in the following descriptions.

* Defaulting
  * If the user does not configure `spec.resources`, the Operator constructs the `default_xxxx` () resource type according to the resource request in the `spec.cluster` TidbCluster, and the count value is not set
  * If `spec.xxxx.rules[].resource_type` is not configured, default to `spec.resources`(which will be `default_xxxx` if not configured)
* Validation
  * For CPU, `min` < `max` < 1
  * For storage, `max` < 1, `min` is ignored, `1-max` will be used when calling PD API

### Auto-scaling

`TC` represents `TidbCluster` in the following descriptions.
`xxxx` represents `tikv` or `tidb` in the following descriptions.

* If `spec.xxxx.external` is configured, according to the `spec.cluster` TC configuration, create a new TC (heterogeneous TC, only contains xxxx), set the label `specialUse: hotRegion` for TiKV, no special configuration for TiDB
* If `spec.xxxx.external` is not configured, the Operator calls the PD API to get the scaling result and creates heterogeneous TCs based on the response.

    Example:
    `resource_a` is configured as:

    ```
    {
    "resource_type":"resource_a",
    "cpu":1,
    "memory":8589934592,
    "storage":107374182400,
    "count":2,
    }
    ```

    Response from PD API:

    ```
    {
    "component":"tikv",
    "count":1,
    "resource_type":"resource_a", 
    "labels":{
        "group":"a",
        "specialUse":"hotRegion",
        "zone":"A",
    }
    }
    ```

    Construct seed as below:

    ```
    {
    "namespace": <ns>,        
    "tas": <TidbAutoScalerName>,               
    "component":"tikv",
    "cpu":1,
    "memory":8589934592,
    "storage":107374182400,, 
    "labels":{
        "group":"a",
        "specialUse":"hotRegion",
        "zone":"A",
    }
    }
    ```

    Hash the seed, take `auto-<hash>` as the TC name.
    Copy `spec.tikv` from the TC of `spec.cluster` and update the configuration according to the API response.
    Create a new TC and set custom labels:

    ```
    app.kubernetes.io/auto-instance: <TidbAutoScalerName>
    app.kubernetes.io/auto-component: xxxx
    ```
