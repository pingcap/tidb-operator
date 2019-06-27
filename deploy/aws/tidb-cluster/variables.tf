variable "subnets" {
  description = "A list of subnets to place the EKS cluster and workers within."
  type        = list(string)
}

variable "tags" {
  description = "A map of tags to add to all resources."
  type        = map(string)
  default     = {}
}

variable "worker_groups" {
  description = "A list of maps defining worker group configurations to be defined using AWS Launch Configurations. See workers_group_defaults for valid keys."
  type        = list(map(string))

  default = [
    {
      "name" = "default"
    },
  ]
}

variable "worker_group_count" {
  description = "The number of maps contained within the worker_groups list."
  type        = string
  default     = "1"
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
      "name" = "default"
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
  description = "Additional name filter for AWS EKS worker AMI. Default behaviour will get latest for the cluster_version but could be set to a release from amazon-eks-ami, e.g. \"v20190220\""
  default     = "v*"
}

variable "worker_additional_security_group_ids" {
  description = "A list of additional security group ids to attach to worker instances"
  type        = list(string)
  default     = []
}

variable "worker_sg_ingress_from_port" {
  description = "Minimum port number from which pods will accept communication. Must be changed to a lower value if some pods in your cluster will expose a port lower than 1025 (e.g. 22, 80, or 443)."
  default     = "1025"
}

variable "workers_additional_policies" {
  description = "Additional policies to be added to workers"
  type        = list(string)
  default     = []
}

variable "workers_additional_policies_count" {
  default = 0
}

variable "kubeconfig_aws_authenticator_command" {
  description = "Command to use to fetch AWS EKS credentials."
  default     = "aws-iam-authenticator"
}

variable "kubeconfig_aws_authenticator_command_args" {
  description = "Default arguments passed to the authenticator command. Defaults to [token -i $cluster_name]."
  type        = list(string)
  default     = []
}

variable "kubeconfig_aws_authenticator_additional_args" {
  description = "Any additional arguments to pass to the authenticator such as the role to assume. e.g. [\"-r\", \"MyEksRole\"]."
  type        = list(string)
  default     = []
}

variable "kubeconfig_aws_authenticator_env_variables" {
  description = "Environment variables that should be used when executing the authenticator. e.g. { AWS_PROFILE = \"eks\"}."
  type        = map(string)
  default     = {}
}

variable "cluster_create_timeout" {
  description = "Timeout value when creating the EKS cluster."
  default     = "15m"
}

variable "cluster_delete_timeout" {
  description = "Timeout value when deleting the EKS cluster."
  default     = "15m"
}

variable "local_exec_interpreter" {
  description = "Command to run for local-exec resources. Must be a shell-style interpreter. If you are on Windows Git Bash is a good choice."
  type        = list(string)
  default     = ["/bin/sh", "-c"]
}

variable "cluster_create_security_group" {
  description = "Whether to create a security group for the cluster or attach the cluster to `cluster_security_group_id`."
  default     = true
}

variable "worker_create_security_group" {
  description = "Whether to create a security group for the workers or attach the workers to `worker_security_group_id`."
  default     = true
}

variable "permissions_boundary" {
  description = "If provided, all IAM roles will be created with this permissions boundary attached."
  default     = ""
}

variable "iam_path" {
  description = "If provided, all IAM roles will be created on this path."
  default     = "/"
}

variable "cluster_endpoint_private_access" {
  description = "Indicates whether or not the Amazon EKS private API server endpoint is enabled."
  default     = false
}

variable "cluster_endpoint_public_access" {
  description = "Indicates whether or not the Amazon EKS public API server endpoint is enabled."
  default     = true
}




variable "operator_version" {
  description = "tidb operator version"
  default = "v1.0.0-beta.3"
}

variable "cluster_name" {
  type = string
  description = "tidb cluster name"
}

variable "cluster_version" {
  type = string
  default = "v3.0.0-rc.2"
}

variable "ssh_key_name" {
  type = string
}

variable "pd_count" {
  type = number
  default = 1
}

variable "tikv_count" {
  type = number
  default = 1
}

variable "tidb_count" {
  type = number
  default = 1
}

variable "pd_instance_type" {
  type = string
  default = "c5d.large"
}

variable "tikv_instance_type" {
  type = string
  default = "c5d.large"
}

variable "tidb_instance_type" {
  type = string
  default = "c5d.large"
}

variable "monitor_instance_type" {
  type = string
  default = "c5d.large"
}

variable "monitor_storage_size" {
  type = string
  default = "100Gi"
}

variable "monitor_enable_anonymous_user" {
  type = bool
  default = false
}

variable "override_values" {
  type = string
}

variable "eks_info" {
  description = "eks info"
}
