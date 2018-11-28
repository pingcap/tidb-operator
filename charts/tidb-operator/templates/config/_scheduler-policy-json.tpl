{
  "kind" : "Policy",
  "apiVersion" : "v1",
  "predicates": [
    {"name": "MatchInterPodAffinity"},
    {"name": "CheckVolumeBinding"},
{{- if semverCompare "<1.12-0" .Values.kubeVersion }}
    {"name": "CheckNodeCondition"},
    {"name": "CheckNodeMemoryPressure"},
    {"name": "CheckNodeDiskPressure"},
{{- end }}
    {"name": "GeneralPredicates"},
    {"name": "HostName"},
    {"name": "PodFitsHostPorts"},
    {"name": "MatchNodeSelector"},
    {"name": "PodFitsResources"},
    {"name": "NoDiskConflict"},
    {"name": "PodToleratesNodeTaints"}
  ],
  "priorities": [
    {"name": "EqualPriority", "weight": 1},
    {"name": "ImageLocalityPriority", "weight": 1},
    {"name": "LeastRequestedPriority", "weight": 1},
    {"name": "BalancedResourceAllocation", "weight": 1},
    {"name": "SelectorSpreadPriority", "weight": 1},
    {"name": "NodePreferAvoidPodsPriority", "weight": 1},
    {"name": "NodeAffinityPriority", "weight": 1},
    {"name": "TaintTolerationPriority", "weight": 1},
    {"name": "MostRequestedPriority", "weight": 1}
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
