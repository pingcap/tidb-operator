resource "aws_security_group" "ssh" {
  name        = "${var.eks_name}-bastion"
  description = "Allow SSH access for bastion instance"
  vpc_id      = var.create_vpc ? module.vpc.vpc_id : var.vpc_id
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

module "ec2" {
  source = "terraform-aws-modules/ec2-instance/aws"

  version                     = "2.3.0"
  name                        = "${var.eks_name}-bastion"
  instance_count              = var.create_bastion ? 1 : 0
  ami                         = data.aws_ami.amazon-linux-2.id
  instance_type               = var.bastion_instance_type
  key_name                    = module.key-pair.key_name
  associate_public_ip_address = true
  monitoring                  = false
  user_data                   = file("bastion-userdata")
  vpc_security_group_ids      = [aws_security_group.ssh.id]
  subnet_ids = split(
    ",",
    var.create_vpc ? join(",", module.vpc.public_subnets) : join(",", var.subnets),
  )
}