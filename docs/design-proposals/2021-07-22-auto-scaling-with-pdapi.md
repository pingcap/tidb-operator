# Auto-scaling with PD API

## Summary

This document presents a design to support auto-scaling for TiDB and TiKV with PD API.

## Motivation

Users may want to auto-scale TiDB or TiKV during high traffic load and may also want to auto-scale TiKV during high disk usage.

### Terms

* homogeneous instance
  
  homogeneous instances are instances which were specified in the TidbCluster CR when creating the tidb cluster, resource requests of all homogeneous instances are as same as original instances. homogeneous instances represent permanent instances, this means that homogeneous instances will not be scaled in even if resource usage is very low.
  

* heterogeneous instance
  
  homogeneous instances are instances which were specified in the TidbClusterAutoScaler CR and created automatically during the run time, resource requests of heterogeneous instances may be different. heterogeneous instances represent temporary instances, this means that heterogeneous instances will be scaled in if resource usage is low.

### Goals

* scale out homogeneous tikv instances when average storage usage is high 
* scale out homogeneous tikv/tidb instances when average CPU usage is high
* scale out heterogeneous tikv/tidb instances when average CPU usage is not high but partial instances are high
* scale in heterogeneous tikv/tidb instances when CPU usage of all instances are low

### Non-Goals

* Auto-scale in homogeneous tikv/tidb instances

## Proposal

### Prerequisites

* The PD API returns the full group information for each API call
* For the same metric, when one group is returned from PD, and if a new group needs to be returned in the following API responses (for example, the user only specifies one resource_type, and then resources in the resource_type are changed), PD should return these two groups in the following responses.

### CR Definition

```
// +k8s:openapi-gen=true
// TidbClusterAutoScaler is the control script's spec
type TidbClusterAutoScalerSpec struct {
	// TidbClusterRef describe the target TidbCluster
	Cluster TidbClusterRef `json:"cluster"`

	// TiKV represents the auto-scaling spec for tikv
	// +optional
	TiKV *TikvAutoScalerSpec `json:"tikv,omitempty"`

	// TiDB represents the auto-scaling spec for tidb
	// +optional
	TiDB *TidbAutoScalerSpec `json:"tidb,omitempty"`
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
}

// +k8s:openapi-gen=true
// AutoResource describes the resource type definitions
type AutoResource struct {
	// CPU defines the CPU of this resource type
	CPU resource.Quantity `json:"cpu"`
	// Memory defines the memory of this resource type
	Memory resource.Quantity `json:"memory"`
	// Storage defines the storage of this resource type
	Storage resource.Quantity `json:"storage,omitempty"`
	// Count defines the max availabel count of this resource type
	Count *int32 `json:"count,omitempty"`
}

// +k8s:openapi-gen=true
// AutoRule describes the rules for auto-scaling with PD API
type AutoRule struct {
	// MaxThreshold defines the threshold to scale out
	MaxThreshold float64 `json:"max_threshold"`
	// MinThreshold defines the threshold to scale in
	MinThreshold *float64 `json:"min_threshold,omitempty"`
	// ResourceTypes defines the resource types that can be used for scaling
	ResourceTypes []string `json:"resource_types,omitempty"`
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
    Endpoint ExternalEndpoint `json:"endpoint"`
    // maxReplicas is the upper limit for the number of replicas to which the autoscaler can scale out.
    // It cannot be less than minReplicas.
    MaxReplicas int32 `json:"maxReplicas"`
}
```
### Defaulting & Validation

`<component>` is `tikv` or `tidb` in the following descriptions.

* Defaulting
  * If the user does not configure `spec.resources`, the Operator constructs the `default_<component>` resource type according to the resource request in the `spec.cluster` TidbCluster, and the count value is not set
  * If `spec.<component>.rules[].resource_type` is not configured, default to `spec.resources`(which will be `default_<component>` if not configured)
* Validation
  * For CPU, `min` < `max` < 1
  * For storage, `min` < `max` < 1

### Algorithm

* storage based scaling
  * if average storage usage is larger than MaxThreshold defined in the AutoRule, homogeneous scaling out plan will be generated, the goal is that new average storage usage is lower than (MaxThreshold + MinThreshold)/2
  * storage based scaling is always homogeneous, homogeneous instances will not be scaled in by auto scaling, even if the all storage usages are very low
  * users can specify maximum homogeneous auto scale count in the TidbClusterAutoScaler CR, but can not specify the resource requests
  * if there are not enough k8s nodes left for the scaling, heterogeneous instances will be scaled in
  * only tikv component performs storage based scaling
  
* cpu based scaling
  * if average cpu usage is larger than MaxThreshold defined in the AutoRule, homogeneous scaling out plan will be generated, the goal is that new average cpu usage is lower than (MaxThreshold + MinThreshold)/2
  * homogeneous instances will not be scaled in by auto scaling, even if all cpu usages are very low
  * if there are not enough k8s nodes left for the scaling, heterogeneous instances will be scaled in
  * if average cpu usage is not larger than MaxThreshold but cpu usages of partial instances are larger than MaxThreshold, heterogeneous scaling out plans will be generated, the goal is that cpu usages of those partial instances are lower than (MaxThreshold + MinThreshold)/2
  * if there are more than one resource defined in the TidbClusterAutoScaler CR, resources will be sorted by the cpu core number in descending order, the larger one will be used preferentially
  * if one resource count is not enough for this scaling purpose, the next resource will be used, this process will continue until all resources are used or there is no free k8s node left
  * if cpu usages of all instances are smaller than MinThreshold, heterogeneous scale in plan will be generated, for now, only one heterogeneous instance will be scaled in at one time
  * both tikv and tidb components perform cpu based scaling

### Auto-scaling

`TC` represents `TidbCluster` in the following descriptions.
`<component>` represents `tikv` or `tidb` in the following descriptions.

* If `spec.<component>.external` is configured, according to the `spec.cluster` TC configuration, create a new TC (heterogeneous TC, only contains <component>), set the label `specialUse: hotRegion` for TiKV, no special configuration for TiDB
* If `spec.<component>.external` is not configured, the Operator calls the PD API to get the scaling result and perform scaling operation based on the response.

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
    "storage":107374182400,
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
    app.kubernetes.io/auto-component: <component>
    ```
