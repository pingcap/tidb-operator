
terraform {
  required_version = ">= 0.12"
  required_providers {
    # TODO: remove the restriction of < 2.19 once the `ip_allocation_policy.0.use_ip_aliases` error fixes
    # https://github.com/terraform-providers/terraform-provider-google/blob/master/CHANGELOG.md#2190-november-05-2019
    google      = ">= 2.16, < 2.19"
    google-beta = ">= 2.16, < 2.19"
    external    = "~> 1.2"
    helm        = "~> 0.10"
    null        = "~> 2.1"
  }
}
