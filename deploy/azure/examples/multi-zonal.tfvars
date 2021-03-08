#
# This will create a zonal cluster in zone us-central1-b with one additional zone.
# Work nodes will be created in primary zone us-central1-b and additional zone us-central1-c.
#
aks_name           = "multi-zonal"
vpc_name           = "multi-zonal"
location           = "westus2"
pd_instance_type   = "Standard_B2s"
tikv_instance_type = "Standard_B2s"
tidb_instance_type = "Standard_B2s"
node_locations = [1,2,3]
