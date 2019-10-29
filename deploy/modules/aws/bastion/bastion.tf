provider "aws" {
  region = var.region
}

resource "aws_security_group" "accept_ssh_from_local" {
  name        = var.bastion_name
  description = "Allow SSH access for bastion instance"
  vpc_id      = var.vpc_id
  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = var.bastion_ingress_cidr
  }
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_security_group_rule" "enable_ssh_to_workers" {
  count                    = var.enable_ssh_to_workers ? 1 : 0
  security_group_id        = var.worker_security_group_id
  source_security_group_id = aws_security_group.accept_ssh_from_local.id
  from_port                = 22
  to_port                  = 22
  protocol                 = "tcp"
  type                     = "ingress"
}

module "ec2" {
  source = "terraform-aws-modules/ec2-instance/aws"

  version                     = "2.3.0"
  name                        = var.bastion_name
  instance_count              = 1
  ami                         = data.aws_ami.centos.id
  instance_type               = var.bastion_instance_type
  key_name                    = var.key_name
  associate_public_ip_address = true
  monitoring                  = false
  user_data                   = file("${path.module}/bastion-userdata")
  vpc_security_group_ids      = [aws_security_group.accept_ssh_from_local.id]
  subnet_ids                  = var.public_subnets
}
