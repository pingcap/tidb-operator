variable "region" {
  description = "Alicloud region"
}

variable "access_key" {
  type = string
}

variable "secret_key" {
  type = string
}

variable "cluster_name" {
  description = "Kubernetes cluster name"
  default     = "my-cluster"
}

variable "k8s_version" {
  description = "Kubernetes cluster version"
  default     = "1.14.8-aliyun.1"
  type        = string
}

variable "cluster_network_type" {
  description = "Kubernetes network plugin, options: [flannel, terway]. Cannot change once created."
  default     = "flannel"
}

variable "operator_version" {
  type    = string
  default = "v1.0.6"
}

variable "operator_helm_values" {
  type    = string
  default = ""
}

variable "public_apiserver" {
  description = "Whether enable apiserver internet access"
  default     = true
}

variable "kubeconfig_file" {
  description = "The path that kubeconfig file write to, default to $$${path.module}/kubeconfig if empty."
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
  description = "The path that new key file write to, defaul to $$${path.module}/$$${cluster_name}-key.pem if empty"
  default     = ""
}

variable "key_pair_name" {
  description = "Key pair for worker instance, specify this variable to use an exsitng key pair. A new key pair will be created by default."
  default     = ""
}

variable "vpc_id" {
  description = "VPC id, specify this variable to use an existing VPC and the vswitches in the VPC. Note that when using existing vpc, it is recommended to use a existing security group too. Otherwise you have to set vpc_cidr according to the existing VPC settings to get correct in-cluster security rule."
  default     = ""
}

variable "group_id" {
  description = "Security group id, specify this variable to use and existing security group"
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
