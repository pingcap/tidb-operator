variable "region" {
  description = "AWS region"
  # supported regions:
  # US: us-east-1, us-east-2, us-west-2
  # Asia Pacific: ap-south-1, ap-northeast-2, ap-southeast-1, ap-southeast-2, ap-northeast-1
  # Europe: eu-central-1, eu-west-1, eu-west-2, eu-west-3, eu-north-1
  default = "us-west-2"
}

variable "eks_name" {
  description = "Name of the EKS cluster. Also used as a prefix in names of related resources."
  default     = "my-cluster"
}

variable "eks_version" {
  description = "Kubernetes version to use for the EKS cluster."
  default     = "1.15"
}

variable "operator_version" {
  description = "TiDB operator version"
  default     = "v1.1.0-rc.3"
}

variable "operator_values" {
  description = "The helm values file for TiDB Operator, path is relative to current working dir"
  default     = ""
}

# Please note that this is only for manually created VPCs, deploying multiple EKS
# clusters in one VPC is NOT supported now.
variable "create_vpc" {
  description = "Create a new VPC or not, if true the vpc_id/subnet_ids must be set correctly, otherwise the vpc_cidr/private_subnets/public_subnets must be set correctly"
  default     = true
}

variable "vpc_cidr" {
  description = "VPC cidr, must be set correctly if create_vpc is true"
  default     = "10.0.0.0/16"
}

variable "private_subnets" {
  description = "VPC private subnets, must be set correctly if create_vpc is true"
  type        = list(string)
  default     = ["10.0.16.0/20", "10.0.32.0/20", "10.0.48.0/20"]
}

variable "public_subnets" {
  description = "VPC public subnets, must be set correctly if create_vpc is true"
  type        = list(string)
  default     = ["10.0.64.0/20", "10.0.80.0/20", "10.0.96.0/20"]
}

variable "vpc_id" {
  description = "VPC id, must be set correctly if create_vpc is false"
  type        = string
  default     = ""
}

variable "subnets" {
  description = "subnet id list, must be set correctly if create_vpc is false"
  type        = list(string)
  default     = []
}

variable "bastion_ingress_cidr" {
  description = "IP cidr that allowed to access bastion ec2 instance"
  default     = ["0.0.0.0/0"] # Note: Please restrict your ingress to only necessary IPs. Opening to 0.0.0.0/0 can lead to security vulnerabilities.
}

variable "create_bastion" {
  description = "Create bastion ec2 instance to access TiDB cluster"
  default     = true
}

variable "bastion_instance_type" {
  description = "bastion ec2 instance type"
  default     = "t2.micro"
}

# For aws tutorials compatiablity
variable "default_cluster_version" {
  default = "v3.0.13"
}

variable "default_cluster_pd_count" {
  default = 3
}

variable "default_cluster_tikv_count" {
  default = 3
}

variable "default_cluster_tidb_count" {
  default = 2
}

variable "default_cluster_pd_instance_type" {
  default = "m5.xlarge"
}

variable "default_cluster_tikv_instance_type" {
  default = "c5d.4xlarge"
}

variable "default_cluster_tidb_instance_type" {
  default = "c5.4xlarge"
}

variable "default_cluster_monitor_instance_type" {
  default = "c5.2xlarge"
}

variable "default_cluster_name" {
  default = "my-cluster"
}

variable "create_tidb_cluster_release" {
  description = "whether creating tidb-cluster helm release"
  default     = false
}

variable "create_tiflash_node_pool" {
  description = "whether creating node pool for tiflash"
  default     = false
}

variable "create_cdc_node_pool" {
  description = "whether creating node pool for cdc"
  default     = false
}

variable "cluster_tiflash_count" {
  default = 2
}

variable "cluster_cdc_count" {
  default = 3
}

variable "cluster_cdc_instance_type" {
  default = "c5.2xlarge"
}

variable "cluster_tiflash_instance_type" {
  default = "i3.4xlarge"
}
