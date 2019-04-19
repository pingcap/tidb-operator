variable "region" {
  description = "aws region"
  default = "us-east-2"
}

variable "ingress_cidr" {
  description = "IP cidr that allowed to access bastion ec2 instance"
  default = ["0.0.0.0/0"]	# Note: Please restrict your ingress to only necessary IPs. Opening to 0.0.0.0/0 can lead to security vulnerabilities.
}

variable "create_vpc" {
  description = "Create a new VPC or not, if true the vpc_cidr/private_subnets/public_subnets must be set correctly, otherwise vpc_id/subnet_ids must be set correctly"
  default = true
}

variable "vpc_cidr" {
  description = "vpc cidr"
  default = "10.0.0.0/16"
}

variable "private_subnets" {
  description = "vpc private subnets"
  type = "list"
  default = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]
}

variable "public_subnets" {
  description = "vpc public subnets"
  type = "list"
  default = ["10.0.4.0/24", "10.0.5.0/24", "10.0.6.0/24"]
}

variable "vpc_id" {
  description = "VPC id"
  type = "string"
  default = "vpc-c679deae"
}

variable "subnets" {
  description = "subnet id list"
  type = "list"
  default = ["subnet-899e79f3", "subnet-a72d80cf", "subnet-a76d34ea"]
}

variable "create_bastion" {
  description = "Create bastion ec2 instance to access TiDB cluster"
  default = true
}

variable "bastion_ami" {
  description = "bastion ami id"
  default = "ami-0cd3dfa4e37921605"
}

variable "bastion_instance_type" {
  description = "bastion ec2 instance type"
  default = "t2.micro"
}

variable "cluster_name" {
  description = "eks cluster name"
  default = "my-cluster"
}

variable "k8s_version" {
  description = "eks cluster version"
  default = "1.12"
}

variable "tidb_version" {
  description = "tidb cluster version"
  default = "v2.1.8"
}

variable "pd_count" {
  default = 3
}

variable "tikv_count" {
  default = 3
}

variable "tidb_count" {
  default = 2
}
