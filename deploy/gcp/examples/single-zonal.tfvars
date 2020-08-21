#
# This will create a zonal cluster in zone us-central1-b without additional zones.
# Work nodes will be created in a single zone only.
#
gke_name           = "single-zonal"
vpc_name           = "single-zonal"
location           = "us-central1-b"
pd_instance_type   = "n1-standard-2"
tikv_instance_type = "n1-highmem-4"
tidb_instance_type = "n1-standard-8"
pd_count           = 3
tikv_count         = 3
tidb_count         = 3
