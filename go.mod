//
// Run hack/pin-deps.sh to change pinned dependency versions.
//

module github.com/pingcap/tidb-operator

go 1.13

require (
	cloud.google.com/go/storage v1.0.0
	github.com/Azure/azure-storage-blob-go v0.8.0
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.2
	github.com/MakeNowJust/heredoc v0.0.0-20171113091838-e9091a26100e // indirect
	github.com/Masterminds/semver v1.4.2
	github.com/NYTimes/gziphandler v1.1.1 // indirect
	github.com/agiledragon/gomonkey/v2 v2.3.0
	github.com/aws/aws-sdk-go v1.30.9
	github.com/chai2010/gettext-go v0.0.0-20170215093142-bf70f2a70fb1 // indirect
	github.com/codahale/hdrhistogram v0.0.0-20161010025455-3a0bb77429bd // indirect
	github.com/docker/docker v1.4.2-0.20200309214505-aa6a9891b09c
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/dustin/go-humanize v1.0.0
	github.com/elazarl/goproxy v0.0.0-20190421051319-9d40249d3c2f // indirect; indirectload
	github.com/elazarl/goproxy/ext v0.0.0-20190421051319-9d40249d3c2f // indirect
	github.com/emicklei/go-restful v2.9.5+incompatible
	github.com/fatih/color v1.7.0
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-sql-driver/mysql v1.5.0
	github.com/gogo/protobuf v1.3.2
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/golang/snappy v0.0.1 // indirect
	github.com/google/go-cmp v0.5.5
	github.com/google/gofuzz v1.1.0
	github.com/gorilla/websocket v1.4.1 // indirect
	github.com/gregjones/httpcache v0.0.0-20190212212710-3befbb6ad0cc // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.13.0 // indirect
	github.com/juju/errors v0.0.0-20180806074554-22422dad46e1
	github.com/juju/loggo v0.0.0-20180524022052-584905176618 // indirect
	github.com/juju/testing v0.0.0-20180920084828-472a3e8b2073 // indirect
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/minio/minio-go/v6 v6.0.55
	github.com/nwaples/rardecode v1.0.0 // indirect
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/openshift/generic-admission-server v1.14.1-0.20210422140326-da96454c926d
	github.com/opentracing/opentracing-go v1.1.0 // indirect
	github.com/pierrec/lz4 v2.0.5+incompatible // indirect
	github.com/pingcap/advanced-statefulset/client v1.17.1-0.20210831081013-d54ef54b2938
	github.com/pingcap/check v0.0.0-20190102082844-67f458068fc8 // indirect
	github.com/pingcap/errors v0.11.0
	github.com/pingcap/kvproto v0.0.0-20200927054727-1290113160f0
	github.com/pingcap/tidb v2.1.0-beta+incompatible
	github.com/pingcap/tidb-operator/pkg/apis v1.3.6
	github.com/pingcap/tidb-operator/pkg/client v1.3.6
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.26.0
	github.com/prometheus/prom2json v1.3.0
	github.com/robfig/cron v1.1.0
	github.com/sethvargo/go-password v0.2.0
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/tikv/pd v2.1.17+incompatible
	github.com/uber-go/atomic v0.0.0-00010101000000-000000000000 // indirect
	github.com/uber/jaeger-client-go v2.19.0+incompatible // indirect
	github.com/uber/jaeger-lib v2.2.0+incompatible // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	github.com/yisaer/crd-validation v0.0.3
	go.etcd.io/etcd v0.5.0-alpha.5.0.20200819165624-17cef6e3e9d5
	gocloud.dev v0.18.0
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e
	gomodules.xyz/jsonpatch/v2 v2.1.0
	google.golang.org/grpc v1.27.0
	gopkg.in/mgo.v2 v2.0.0-20180705113604-9856a29383ce // indirect
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.19.16
	k8s.io/apiextensions-apiserver v0.19.16
	k8s.io/apimachinery v0.19.16
	k8s.io/apiserver v0.19.16
	k8s.io/cli-runtime v0.19.16
	k8s.io/client-go v0.19.16
	k8s.io/code-generator v0.19.16
	k8s.io/component-base v0.19.16
	k8s.io/klog/v2 v2.3.0
	k8s.io/kube-aggregator v0.19.16
	k8s.io/kube-scheduler v0.19.16
	k8s.io/kubectl v0.19.16
	k8s.io/kubernetes v1.19.16
	k8s.io/utils v0.0.0-20200912215256-4140de9c8800
	sigs.k8s.io/apiserver-builder-alpha/cmd v0.0.0-20191113095113-4493943d2568
	sigs.k8s.io/controller-runtime v0.7.2
)

replace github.com/pingcap/tidb-operator/pkg/apis => ./pkg/apis

replace github.com/pingcap/tidb-operator/pkg/client => ./pkg/client

replace github.com/renstrom/dedent => github.com/lithammer/dedent v1.1.0

replace k8s.io/api => k8s.io/api v0.19.16

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.16

replace k8s.io/apimachinery => k8s.io/apimachinery v0.19.16

replace k8s.io/apiserver => k8s.io/apiserver v0.19.16

replace k8s.io/cli-runtime => k8s.io/cli-runtime v0.19.16

replace k8s.io/client-go => k8s.io/client-go v0.19.16

replace k8s.io/code-generator => k8s.io/code-generator v0.19.16

replace k8s.io/csi-api => k8s.io/csi-api v0.0.0-20190118125032-c4557c74373f

replace k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.19.16

replace k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.19.16

replace k8s.io/kube-proxy => k8s.io/kube-proxy v0.19.16

replace k8s.io/kubelet => k8s.io/kubelet v0.19.16

replace k8s.io/metrics => k8s.io/metrics v0.19.16

replace k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.19.16

replace k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.19.16

replace k8s.io/sample-controller => k8s.io/sample-controller v0.19.16

replace k8s.io/cloud-provider => k8s.io/cloud-provider v0.19.16

replace k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.19.16

replace k8s.io/component-base => k8s.io/component-base v0.19.16

replace k8s.io/cri-api => k8s.io/cri-api v0.19.16

replace k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.19.16

replace k8s.io/kubectl => k8s.io/kubectl v0.19.16

replace k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.19.16

replace k8s.io/node-api => k8s.io/node-api v0.0.0-20190918163711-2299658ad911

replace github.com/uber-go/atomic => go.uber.org/atomic v1.5.0

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v14.2.0+incompatible

replace github.com/prometheus/client_golang => github.com/prometheus/client_golang v1.11.0

replace k8s.io/controller-manager => k8s.io/controller-manager v0.19.16

replace k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.19.16

// workaround for https://github.com/kubernetes/apiserver/issues/65
// controller-rutime v0.7.2 use github.com/googleapis/gnostic v0.5.0, kube-apiserver v1.19 use github.com/googleapis/gnostic v0.4.1
// so downgrade github.com/googleapis/gnostic to v0.4.1
// TODO: remove it after upgrading kubernetes dependency to v1.22
replace github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.1

// workaround for avd.aquasec.com/nvd/cve-2020-29652
// TODO: remove it after upgrading
replace golang.org/x/crypto => golang.org/x/crypto v0.0.0-20201216223049-8b5274cf687f

// workaround for github.com/advisories/GHSA-25xm-hr59-7c27
// TODO: remove it after upgrading github.com/mholt/archiver greater than v3.5.0
replace github.com/ulikunitz/xz => github.com/ulikunitz/xz v0.5.8

// workaround for github.com/advisories/GHSA-w73w-5m7g-f7qc
// TODO: remove it after upgrading k8s.io/client-go equal or greater than v0.20.0
replace github.com/dgrijalva/jwt-go => github.com/golang-jwt/jwt v3.2.1+incompatible
