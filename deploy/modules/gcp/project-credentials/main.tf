resource "null_resource" "prepare-dir" {
  provisioner "local-exec" {
    command = "mkdir -p ${var.path}"
  }
}

resource "null_resource" "set-gcloud-project" {
  provisioner "local-exec" {
    command = "gcloud config set project ${var.gcloud_project}"
  }
}