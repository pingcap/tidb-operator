variable "region" {
  description = "aws region"
  # supported regions:
  # US: us-east-1, us-east-2, us-west-2
  # Asia Pacific: ap-south-1, ap-northeast-2, ap-southeast-1, ap-southeast-2, ap-northeast-1
  # Europe: eu-central-1, eu-west-1, eu-west-2, eu-west-3, eu-north-1
  default = "us-west-2"
}

# Please note that this is only for manually created VPCs, deploying multiple EKS
# clusters in one VPC is NOT supported now.
variable "create_vpc" {
  description = "Create a new VPC or not, if true the vpc_cidr/private_subnets/public_subnets must be set correctly, otherwise vpc_id/subnet_ids must be set correctly"
  default     = true
}

variable "vpc_cidr" {
  description = "vpc cidr"
  default     = "10.0.0.0/16"
}

variable "private_subnets" {
  description = "vpc private subnets"
  type        = list(string)
  default     = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]
}

variable "public_subnets" {
  description = "vpc public subnets"
  type        = list(string)
  default     = ["10.0.4.0/24", "10.0.5.0/24", "10.0.6.0/24"]
}

variable "vpc_id" {
  description = "VPC id"
  type        = string
  default     = "vpc-c679deae"
}

variable "subnets" {
  description = "subnet id list"
  type        = list(string)
  default     = ["subnet-899e79f3", "subnet-a72d80cf", "subnet-a76d34ea"]
}

variable "eks_name" {
  description = "Name of the EKS cluster. Also used as a prefix in names of related resources."
  default     = "my-cluster"
}

variable "eks_version" {
  description = "Kubernetes version to use for the EKS cluster."
  default     = "1.12"
}

variable "operator_version" {
  description = "tidb operator version"
  default     = "v1.0.0-beta.3"
}
