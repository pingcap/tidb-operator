#
# This will create a zonal cluster in zone us-central1-b without additional zones.
# Work nodes will be created in a single zone only.
#
aks_name           = "single-zonal"
vpc_name           = "single-zonal"
location           = "westus2"
pd_instance_type   = "Standard_B2s"
tikv_instance_type = "Standard_B2s"
tidb_instance_type = "Standard_B2s"
pd_count           = 3
tikv_count         = 3
tidb_count         = 3
