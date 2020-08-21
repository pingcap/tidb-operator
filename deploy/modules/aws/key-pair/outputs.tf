output "key_name" {
  value = aws_key_pair.generated.key_name
}

output "public_key_openssh" {
  value = tls_private_key.generated.public_key_openssh
}

output "private_key_pem" {
  value = tls_private_key.generated.private_key_pem
}

output "public_key_filepath" {
  value = local.public_key_filename
}

output "private_key_filepath" {
  value = local.private_key_filename
}

