resource "null_resource" "prepare-dir" {
  provisioner "local-exec" {
    command = "mkdir -p ${var.path}"
  }
}
