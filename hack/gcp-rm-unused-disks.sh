#!/usr/bin/env bash
#
# Delete unused (PD) disks.
# It is relatively easy to orphan cloud disk in GKE: just delete the K8s cluster.
#
# Assumption:
# * If you have any inactive (not mounted) PD disks they are represented by a PV in K8s.
# * Your disks are single AZ disks (multi-AZ disk will be ignored)

# Warning:
# * If this script does not work correctly, you will suffer data loss.
# * In a production environment set DRY_RUN=true and inspect disks before deleting them
#
# Extra Safety:
# * If a disk is mounted in an instance, gcloud refuses to delete it.

set -euo pipefail

DRY_RUN="${DRY_RUN:-}"

REGIONS=$(gcloud compute disks list | awk '{print $2}' | grep -v LOCATION | sort | uniq | cut -d "-" -f 1-2 | uniq)
disk_zones=$(gcloud compute disks list | awk '{print $2}' | grep -v LOCATION | sort | uniq | xargs echo -n)

PROJECT="${PROJECT:-$(gcloud config get-value project)}"
echo "using PROJECT: $PROJECT"
echo "looking at REGIONS: $REGIONS"

pv_file="pvs.txt"
node_file=nodes.txt
instance_file=instances.txt
trap "rm -f $pv_file $node_file" "EXIT"

export KUBECONFIG=./kubeconfig
touch "$KUBECONFIG"
chmod 0600 "$KUBECONFIG"
trap "rm -f $KUBECONFIG" "EXIT"

declare -A disks

echo "$REGIONS" | while read -r region ; do
  zones=$(gcloud compute zones list | grep  "$region" | awk '{print $1}')

  # Get the disk listings first. Otherwise we might think the instance using the disk does not exist if it was just created
  echo "$zones" | while read -r zone ; do
    gcloud compute instances list --filter="zone:($zone)" > "$zone-$instance_file"
    trap "rm -f $zone-$instance_file" "EXIT"
    gcloud compute disks list --filter="zone:($zone)" > "$zone-disks"
    trap "rm -f $zone-disks" "EXIT"
  done

  rm -f $pv_file $node_file
  clusters=$(gcloud container clusters list --filter="location:($region)" --format json | jq -r -c '.[] | (.name + " " + .location)')
  if [[ -z "$clusters" ]] ; then
    touch $pv_file $node_file
  else
    echo "$clusters" | while read -r name location ; do
      location_type="--zone"
      if [[ -z "$(echo "$location" | cut -d '-' -f 3)" ]] ; then
        location_type="--region"
      fi
      gcloud container clusters get-credentials "$name" "$location_type" $location
      kubectl describe pv | awk '/PDName:/ {print $2}' | sort >> $pv_file
      kubectl get node --no-headers | awk '{print $1}' | sort >> $node_file
    done
  fi
  rm -f "$KUBECONFIG"

  echo "$zones" | while read -r zone ; do
    if disks=$(cat "$zone-disks" | grep gke | awk '{print $1}') ; then
      echo "  ZONE $zone"
      echo "$disks" | while read -r disk ; do
	  if grep "$disk" $pv_file > /dev/null ; then
	    echo "KEEP disk $disk used by pv"
	  else
	    disk_json=$(gcloud compute disks describe "$disk" --zone "$zone" --format json)
	    node="$(echo "$disk_json" | jq -r '.selfLink')"
            if instance=$(grep "$node" $zone-$instance_file) >/dev/null ; then
	      echo "KEEP disk $disk $zone used by instance $instance"
            else
	      disk_gb="$(echo "$disk_json" | jq -r '.sizeGb')"
	      users_null="$(echo "$disk_json" | jq -r '.users')"
	      if [[ "$users_null" == null ]] ; then
	        if [[ -z $DRY_RUN ]] ; then
	          echo "DELETE disk $disk $zone ${disk_gb}GB"
	          echo "Y" | gcloud compute disks delete "$disk" --zone "$zone"
	        else
	          echo "DRY RUN: would delete disk $disk $zone ${disk_gb}GB"
		fi
	      else
	        users="$(echo "$disk_json" | jq -r '.users[]')"
	        if [[ "$(echo "$users" | wc -l)" -eq 0 ]] ; then
  		  echo "expected null but go an empty users field for disk" >&2
		  exit 1
	        fi

		echo "$users" | while read -r user ; do
		  if ! echo "$user" | grep 'instances' >/dev/null ; then
		    echo "didn't understand user: $user for disk: $disk" >&2
		    exit 1
		  fi
		done

		if [[ "$(echo "$users" | wc -l)" -gt 1 ]] ; then
		  echo "KEEP disk $disk with multiple users $users" >&2
		else
		  user="$(echo "$users" | head -1)"
		  node=$(echo "$user" | awk -F '/' '{print $NF}')
		  if grep "$node" $node_file >/dev/null ; then
		    echo "KEEP disk $disk $zone used by node $node"
		  else
		    if instance=$(grep "$node" $zone-$instance_file) >/dev/null ; then
		      echo "KEEP disk $disk $zone used by instance $instance"
		    else
	              if [[ -z $DRY_RUN ]] ; then
		        echo "DELETE disk $disk $zone ${disk_gb}GB"
		        echo "Y" | gcloud compute disks delete "$disk" --zone "$zone"
		      else
		        echo "DRY RUN: would delete disk $disk $zone ${disk_gb}GB"
		      fi
		    fi
		  fi
		fi
	      fi
	    fi
	  fi
      done
    fi
  done
done

# Just in case the original listing was paged.
other_zones=$(gcloud compute disks list --filter="NOT zone:($disk_zones)" | wc -l)
if [[ $other_zones -gt 0 ]] ; then
  echo "Seeing new regions now. Run this script again"
  exit 1
fi
