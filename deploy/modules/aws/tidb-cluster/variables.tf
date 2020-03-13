variable "region" {
  description = "AWS region"
}

variable "tidb_cluster_chart_version" {
  description = "tidb-cluster chart version"
  default     = "v1.0.6"
}

variable "create_tidb_cluster_release" {
  description = "Whether create tidb-cluster release in the node pools automatically"
  default     = true
}

variable "cluster_name" {
  type        = string
  description = "tidb cluster name"
}

variable "cluster_version" {
  type    = string
  default = "v3.0.8"
}

variable "ssh_key_name" {
  type = string
}

variable "pd_count" {
  type    = number
  default = 3
}

variable "tikv_count" {
  type    = number
  default = 3
}

variable "tidb_count" {
  type    = number
  default = 2
}

variable "pd_instance_type" {
  type    = string
  default = "m5.xlarge"
}

variable "tikv_instance_type" {
  type    = string
  default = "c5d.4xlarge"
}

variable "tidb_instance_type" {
  type    = string
  default = "c5.4xlarge"
}

variable "monitor_instance_type" {
  type    = string
  default = "c5.2xlarge"
}

variable "override_values" {
  description = "The helm values of TiDB cluster, it is recommended to use the 'file()' function call to read the content from a local file, e.g. 'file(\"my-cluster.yaml\")'"
  type        = string
  default     = ""
}

variable "eks" {
  description = "eks info"
}

# Advanced customization below
variable "subnets" {
  description = "A list of subnets to place the EKS cluster and workers within."
  type        = list(string)
}

variable "tags" {
  description = "A map of tags to add to all resources."
  type        = map(string)
  default     = {}
}

variable "workers_group_defaults" {
  description = "Override default values for target groups. See workers_group_defaults_defaults in local.tf for valid keys."
  type        = map(string)
  default     = {}
}

variable "worker_group_tags" {
  description = "A map defining extra tags to be applied to the worker group ASG."
  type        = map(list(string))

  default = {
    default = []
  }
}

variable "worker_groups_launch_template" {
  description = "A list of maps defining worker group configurations to be defined using AWS Launch Templates. See workers_group_defaults for valid keys."
  type        = list(map(string))

  default = [
    {
      name = "default"
    },
  ]
}

variable "worker_group_launch_template_count" {
  description = "The number of maps contained within the worker_groups_launch_template list."
  type        = string
  default     = "0"
}

variable "workers_group_launch_template_defaults" {
  description = "Override default values for target groups. See workers_group_defaults_defaults in local.tf for valid keys."
  type        = map(string)
  default     = {}
}

variable "worker_group_launch_template_tags" {
  description = "A map defining extra tags to be applied to the worker group template ASG."
  type        = map(list(string))

  default = {
    default = []
  }
}

variable "worker_ami_name_filter" {
  description = "Additional name filter for AWS EKS worker AMI. Default behaviour will get the latest AMI for the cluster_version, but it could be set to a specific version, e.g. \"v20190220\", please check the `Packer version` in https://docs.aws.amazon.com/eks/latest/userguide/eks-linux-ami-versions.html for the supported AMI versions."
  default     = "v*"
}

variable "worker_additional_security_group_ids" {
  description = "A list of additional security group ids to attach to worker instances"
  type        = list(string)
  default     = []
}

variable "local_exec_interpreter" {
  description = "Command to run for local-exec resources. Must be a shell-style interpreter. If you are on Windows Git Bash is a good choice."
  type        = list(string)
  default     = ["/bin/sh", "-c"]
}

variable "iam_path" {
  description = "If provided, all IAM roles will be created on this path."
  default     = "/"
}

variable "kubelet_extra_args" {
  description = "Extra arguments passed to kubelet"
  default     = "--kube-reserved memory=0.3Gi,ephemeral-storage=1Gi --system-reserved memory=0.2Gi,ephemeral-storage=1Gi"
}

variable "group_kubelet_extra_args" {
  description = "If provided, override the kubelet_extra_args for a specific node group which matches the key of map (e.g. tidb, tikv, pd, monitor)"
  type        = map(string)
  default     = {}
}
