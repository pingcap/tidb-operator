variable "region" {
  description = "AWS region"
}

variable "bastion_name" {
  description = "Name of the EKS cluster. Also used as a prefix in names of related resources."
  type        = string
}

variable "enable_ssh_to_workers" {
  description = "Whether enable ssh from bastion to workers, if true, the worker_security_group_id must be provided"
  default     = false
}

variable "worker_security_group_id" {
  description = "The security group that bastion allowed to ssh to"
  type        = string
  default     = ""
}

variable "key_name" {
  type = string
}

variable "vpc_id" {
  type = string
}

variable "public_subnets" {
  description = "VPC public subnets, must be set correctly if create_vpc is true"
  type        = list(string)
}

variable "bastion_ingress_cidr" {
  description = "IP cidr that allowed to access bastion ec2 instance"
  default     = ["0.0.0.0/0"] # Note: Please restrict your ingress to only necessary IPs. Opening to 0.0.0.0/0 can lead to security vulnerabilities.
}

variable "bastion_instance_type" {
  description = "bastion ec2 instance type"
  default     = "t2.micro"
}
