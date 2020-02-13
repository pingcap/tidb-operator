#
# This will create a zonal cluster in zone us-central1-b with one additional zone.
# Work nodes will be created in primary zone us-central1-b and additional zone us-central1-c.
#
gke_name           = "multi-zonal"
vpc_name           = "multi-zonal"
location           = "us-central1-b"
pd_instance_type   = "n1-standard-2"
tikv_instance_type = "n1-highmem-4"
tidb_instance_type = "n1-standard-8"
node_locations = [
  "us-central1-c"
]
