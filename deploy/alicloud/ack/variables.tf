variable "cluster_name" {
  description = "Kubernetes cluster name"
  default     = "ack-cluster"
}

variable "cluster_network_type" {
  description = "Kubernetes network plugin, options: [flannel, terway]. Cannot change once created."
  default     = "flannel"
}

variable "span_all_zones" {
  description = "Whether span worker nodes in all avaiable zones, worker_zones will be ignored if span_all_zones=true"
  default     = true
}

variable "worker_zones" {
  description = "Available zones of worker nodes, used when span_all_zones=false. It is highly recommended to guarantee the instance type of workers is available in at least two zones in favor of HA."
  type        = "list"
  default     = []
}

variable "public_apiserver" {
  description = "Whether enable apiserver internet access"
  default     = false
}

variable "kubeconfig_file" {
  description = "The path that kubeconfig file write to, default to $${path.module}/kubeconfig if empty."
  default     = ""
}

variable "k8s_pod_cidr" {
  description = "The kubernetes pod cidr block. It cannot be equals to vpc's or vswitch's and cannot be in them. Cannot change once the cluster created."
  default     = "172.20.0.0/16"
}

variable "k8s_service_cidr" {
  description = "The kubernetes service cidr block. It cannot be equals to vpc's or vswitch's or pod's and cannot be in them. Cannot change once the cluster created."
  default     = "172.21.0.0/20"
}

variable "vpc_cidr" {
  description = "VPC cidr_block, options: [192.168.0.0.0/16, 172.16.0.0/16, 10.0.0.0/8], cannot collidate with kubernetes service cidr and pod cidr. Cannot change once the vpc created."
  default     = "192.168.0.0/16"
}

variable "key_file" {
  description = "The path that new key file write to, defaul to $${path.module}/$${cluster_name}-key.pem if empty"
  default     = ""
}

variable "key_pair_name" {
  description = "Key pair for worker instance, specify this variable to use an exsitng key pair. A new key pair will be created by default."
  default     = ""
}

variable "vpc_id" {
  description = "VPC id, specify this variable to use an exsiting VPC and the vswitches in the VPC. Note that when using existing vpc, it is recommended to use a existing security group too. Otherwise you have to set vpc_cidr according to the existing VPC settings to get correct in-cluster security rule."
  default     = ""
}

variable "group_id" {
  description = "Security group id, specify this variable to use and exising security group"
  default     = ""
}

variable "vpc_cidr_newbits" {
  description = "VPC cidr newbits, it's better to be set as 16 if you use 10.0.0.0/8 cidr block"
  default     = "8"
}

variable "create_nat_gateway" {
  description = "If create nat gateway in VPC"
  default     = true
}

variable "default_worker_count" {
  description = "The number of kubernetes default worker nodes, value: [2,50]. See module README for detail."
  default     = 2
}

variable "default_worker_cpu_core_count" {
  description = "The instance cpu core count of kubernetes default worker nodes, this variable will be ignroed if default_worker_type set"
  default     = 1
}

variable "default_worker_type" {
  description = "The instance type of kubernets default worker nodes, it is recommend to use default_worker_cpu_core_count to select flexible instance type"
  default     = ""
}

variable "worker_groups" {
  description = "A list of maps defining worker group configurations to be defined using alicloud ESS. See group_default for validate keys."
  type        = "list"

  default = [
    {
      "name" = "default"
    },
  ]
}

variable "group_default" {
  description = <<EOS
The default values for all worker groups, overriden by per group arguments:
  - min_size: min size of scaling group
  - max_size: max size of scaling group
  - default_cooldown: cooldown peroid between scaling operation
  - multi_az_policy: options [BALANCE, PRIORITY]
  - image_id: OS image of worker instance
  - instance_type:
  - system_disk_category: self-explained
  - systen_disk_size: system disk size in GB
  - pre_userdata: customized instance initialization actions before register instance to k8s cluster
  - post_userdata: customized instance initialization actions after register instance to k8s cluster
  - autoscaling_enables: whether enable autoscaling using cluster scaler
  - internet_charge_type: network billing type, Values: [PayByBandwidth, PayByTraffic]
  - internet_max_bandwith_in: self-explained, range [1,200]
  - internet_max_bandwith_out: self-explained, range [1,200]
  - node_taints: kubernetes node tains, format: k1=v1,k2=v2
  - node_labels: kubernetes node labels, format: k1=v1,k2=v2
EOS

  type = "map"

  default = {
    "min_size"                   = 0
    "max_size"                   = 100
    "default_cooldown"           = 300
    "multi_az_policy"            = "BALANCE"
    "image_id"                   = "centos_7_06_64_20G_alibase_20190218.vhd"
    "instance_type"              = "ecs.g5.large"
    "system_disk_category"       = "cloud_efficiency"
    "system_disk_size"           = 50
    "pre_userdata"               = ""
    "post_userdata"              = ""
    "autoscaling_enabled"        = false
    "internet_charge_type"       = "PayByTraffic"
    "internet_max_bandwidth_in"  = 10
    "internet_max_bandwidth_out" = 10
    "node_taints" = ""
    "node_labels" = ""
  }
}

variable "default_group_tags" {
  description = "A map of tags to add to all ecs instances."
  type        = "map"
  default     = {}
}

variable "worker_group_tags" {
  description = "A list of tags to add to ecs instances of the spcific group, mapping by index with mod."
  type        = "list"
  default     = [{}]
}
