apiVersion: "jobs.aliyun.com/v1alpha1"
kind: GlobalJob
metadata:
  name: kubelet-customize
spec:
  maxParallel: 100
  terminalStrategy:
    type: Never
  nodeSelector:
    nodeSelectorTerms:
    - matchExpressions:
      # If you need run this global job on all workers[exclude masters], you can use example below:
      #- key: node-role.kubernetes.io/master
      #  operator: DoesNotExist
      # If you need run this global job on some workers who have labels e.g. kubelet-customize=true, you can use example below:
      - key: kubelet-customize
        operator: In
        values:
        - "true"
  template:
    spec:
      containers:
      - name: kubelet-customize
        image: registry.cn-hangzhou.aliyuncs.com/acs/kubelet-customize:v1.1
        #imagePullPolicy: Always
        env:
        # ACTION type:
        # [UPDATE]: UPDATE kubelet with provided KUBELET_CUSTOMIZE_ARGS
        # [ROLLBACK]: ROOLBACK to latest backup version of kubelet if it exists
        # [NONE]: NONE do nothing for the kubelet
        - name: KUBELET_ACTION_TYPE
          value: "${action_type}"
        # each option in KUBELET_CUSTOMIZE_ARGS must be separated with semicolon <;>
              # e.g. --cpu-manager-policy=static;--cluster-domain=myk8s.local;--kube-reserved=cpu=100m,memory=400Mi;--system-reserved=cpu=100m,memory=300Mi
              # currently KUBELET_CUSTOMIZE_ARGS support [--cpu-manager-policy=static;--cluster-domain=<value>;--kube-reserved=cpu=<value>,memory=<value>;--system-reserved=cpu=<value>,memory=<value>]
        - name: KUBELET_CUSTOMIZE_ARGS
          value: "${customize_args}"
        - name: KUBELET_CUSTOMIZE_ARGS_WHITELIST_KEYS
          value: "${whitelist_keys}"
        volumeMounts:
        - mountPath: /alicloud-k8s-host
          name: kubelet-customize
      hostNetwork: true
      hostPID: true
      restartPolicy: Never
      volumes:
      - hostPath:
          path: /
          type: Directory
        name: kubelet-customize

