{
  "kind" : "Policy",
  "apiVersion" : "v1",
  "predicates": [
    {"name": "NoVolumeZoneConflict"},
    {"name": "MaxEBSVolumeCount"},
    {"name": "MaxAzureDiskVolumeCount"},
    {"name": "NoDiskConflict"},
    {"name": "GeneralPredicates"},
    {"name": "PodToleratesNodeTaints"},
    {"name": "CheckVolumeBinding"},
    {"name": "MaxGCEPDVolumeCount"},
    {"name": "MatchInterPodAffinity"},
{{- if semverCompare "~1.11.0" .Capabilities.KubeVersion.GitVersion }}
    {"name": "CheckNodePIDPressure"},
{{- end }}
{{- if semverCompare "<1.12-0" .Capabilities.KubeVersion.GitVersion }}
    {"name": "CheckNodeCondition"},
    {"name": "CheckNodeMemoryPressure"},
    {"name": "CheckNodeDiskPressure"},
{{- end }}
    {"name": "CheckVolumeBinding"}
  ],
  "priorities": [
    {"name": "SelectorSpreadPriority", "weight": 1},
    {"name": "InterPodAffinityPriority", "weight": 1},
    {"name": "LeastRequestedPriority", "weight": 1},
    {"name": "BalancedResourceAllocation", "weight": 1},
    {"name": "NodePreferAvoidPodsPriority", "weight": 1},
    {"name": "NodeAffinityPriority", "weight": 1},
    {"name": "TaintTolerationPriority", "weight": 1}
  ],
  "extenders": [
    {
      "urlPrefix": "http://127.0.0.1:10262/scheduler",
      "filterVerb": "filter",
      "weight": 1,
      "httpTimeout": 30000000000,
      "enableHttps": false
    }
  ]
}
