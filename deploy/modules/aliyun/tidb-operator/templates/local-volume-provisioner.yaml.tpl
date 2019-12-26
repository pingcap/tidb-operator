apiVersion: v1
kind: ConfigMap
metadata:
  name: local-provisioner-config
  namespace: kube-system
data:
  nodeLabelsForPV: |
    - kubernetes.io/hostname
  storageClassMap: |
    local-volume:
       vendor: alibabacloud
       hostDir: /mnt/disks
       mountDir: /mnt/disks
       blockCleanerCommand:
         - "/scripts/shred.sh"
         - "2"
       volumeMode: Filesystem
       fsType: ext4
       deviceStartWith: vdb
       mkFSOptions: ""
       mountOptions: "nodelalloc"
---
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: local-volume
provisioner: kubernetes.io/no-provisioner
reclaimPolicy: Retain
volumeBindingMode: WaitForFirstConsumer
---
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: local-volume-provisioner
  namespace: kube-system
  labels:
    app: local-volume-provisioner
spec:
  selector:
    matchLabels:
      app: local-volume-provisioner
  template:
    metadata:
      labels:
        app: local-volume-provisioner
    spec:
      tolerations:
        - key: dedicated
          operator: Exists
          effect: "NoSchedule"
      nodeSelector:
        pingcap.com/aliyun-local-ssd: "true"
      hostPID: true
      hostNetwork: true
      serviceAccountName: admin
      containers:
      - image: registry.cn-hangzhou.aliyuncs.com/plugins/local-volume-provisioner:v1.12-7802d35-aliyun
        imagePullPolicy: "Always"
        name: provisioner
        securityContext:
          privileged: true
        env:
        - name: MY_NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        - name: ACCESS_KEY_ID
          value: "${access_key_id}"
        - name: ACCESS_KEY_SECRET
          value: "${access_key_secret}"
        resources:
          requests:
            cpu: 100m
            memory: 100Mi
          limits:
            cpu: 100m
            memory: 100Mi
        volumeMounts:
        - mountPath: /etc/provisioner/config
          name: provisioner-config
          readOnly: true
        - mountPath:  /mnt/disks
          name: local
          mountPropagation: "HostToContainer"
        - mountPath: /etc/kubernetes
          name: etc
      volumes:
      - name: provisioner-config
        configMap:
          name: local-provisioner-config
      - name: local
        hostPath:
          path: /mnt/disks
      - name: etc
        hostPath:
          path: /etc/kubernetes
