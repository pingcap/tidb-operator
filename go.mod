//
// Run hack/pin-deps.sh to change pinned dependency versions.
//

module github.com/pingcap/tidb-operator

go 1.21

require (
	cloud.google.com/go/storage v1.30.1
	github.com/Azure/azure-storage-blob-go v0.8.0
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.2
	github.com/Masterminds/semver v1.4.2
	github.com/agiledragon/gomonkey/v2 v2.7.0
	github.com/aws/aws-sdk-go v1.44.72
	github.com/aws/aws-sdk-go-v2 v1.16.11
	github.com/aws/aws-sdk-go-v2/config v1.17.1
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.53.0
	github.com/aws/smithy-go v1.12.1
	github.com/docker/docker v17.12.0-ce-rc1.0.20200916142827-bd33bbf0497b+incompatible
	github.com/dustin/go-humanize v1.0.0
	github.com/emicklei/go-restful v2.16.0+incompatible
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-sql-driver/mysql v1.5.0
	github.com/gogo/protobuf v1.3.2
	github.com/google/go-cmp v0.5.9
	github.com/google/gofuzz v1.1.0
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/minio/minio-go/v6 v6.0.55
	github.com/ncw/directio v1.0.5
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/openshift/generic-admission-server v1.14.1-0.20210422140326-da96454c926d
	github.com/pingcap/TiProxy/lib v0.0.0-20230201020701-df06ec482c69
	github.com/pingcap/advanced-statefulset/client v1.17.1-0.20230403114412-d141a788a127
	github.com/pingcap/errors v0.11.4
	github.com/pingcap/kvproto v0.0.0-20231122054644-fb0f5c2a0a10
	github.com/pingcap/tidb-operator/pkg/apis v1.5.3
	github.com/pingcap/tidb-operator/pkg/client v1.5.3
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.26.0
	github.com/prometheus/prom2json v1.3.0
	github.com/r3labs/diff/v2 v2.15.1
	github.com/robfig/cron v1.1.0
	github.com/sethvargo/go-password v0.2.0
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cobra v1.5.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.8.1
	github.com/tikv/pd v2.1.17+incompatible
	github.com/yisaer/crd-validation v0.0.3
	go.etcd.io/etcd v0.5.0-alpha.5.0.20200910180754-dd1b699fc489
	gocloud.dev v0.18.0
	golang.org/x/sync v0.1.0
	golang.org/x/time v0.3.0
	gomodules.xyz/jsonpatch/v2 v2.1.0
	google.golang.org/grpc v1.59.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.20.15
	k8s.io/apiextensions-apiserver v0.20.15
	k8s.io/apimachinery v0.20.15
	k8s.io/apiserver v0.20.15
	k8s.io/cli-runtime v0.20.15
	k8s.io/client-go v0.20.15
	k8s.io/component-base v0.20.15
	k8s.io/klog/v2 v2.4.0
	k8s.io/kube-aggregator v0.20.15
	k8s.io/kube-scheduler v0.20.15
	k8s.io/kubectl v0.20.15
	k8s.io/kubernetes v1.20.15
	k8s.io/utils v0.0.0-20201110183641-67b214c5f920
	mvdan.cc/sh/v3 v3.4.3
	sigs.k8s.io/controller-runtime v0.7.2
)

require (
	cloud.google.com/go/compute v1.23.0 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	cloud.google.com/go/iam v1.1.2 // indirect
	github.com/form3tech-oss/jwt-go v3.2.2+incompatible // indirect
	go.uber.org/atomic v1.9.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20231002182017-d307bd883b97 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231016165738-49dd2c1f3d0b // indirect
	k8s.io/controller-manager v0.20.15 // indirect
	k8s.io/kubelet v0.0.0 // indirect
)

require (
	cloud.google.com/go v0.110.8 // indirect
	github.com/Azure/azure-pipeline-go v0.2.1 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.6 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.5 // indirect
	github.com/Azure/go-autorest/autorest/azure/cli v0.4.1 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/logger v0.2.0 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/BurntSushi/toml v0.3.1 // indirect
	github.com/GoogleCloudPlatform/k8s-cloud-provider v0.0.0-20200415212048-7901bc822317 // indirect
	github.com/MakeNowJust/heredoc v0.0.0-20171113091838-e9091a26100e // indirect
	github.com/Microsoft/go-winio v0.4.15 // indirect
	github.com/NYTimes/gziphandler v1.1.1 // indirect
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.12.14 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.12.12 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.18 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.12 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.19 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.11.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.16.13 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/chai2010/gettext-go v0.0.0-20170215093142-bf70f2a70fb1 // indirect
	github.com/containerd/containerd v1.4.1 // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20190321100706-95778dfbb74e // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dimchansky/utfbom v1.1.0 // indirect
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/elazarl/goproxy v0.0.0-20190421051319-9d40249d3c2f // indirect; indirectload
	github.com/evanphx/json-patch v4.9.0+incompatible // indirect
	github.com/exponent-io/jsonpath v0.0.0-20151013193312-d6023ce2651d // indirect
	github.com/fatih/camelcase v1.0.0 // indirect
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/go-logr/logr v0.3.0 // indirect
	github.com/go-openapi/jsonpointer v0.19.3 // indirect
	github.com/go-openapi/jsonreference v0.19.3 // indirect
	github.com/go-openapi/spec v0.19.3 // indirect
	github.com/go-openapi/swag v0.19.5 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/btree v1.0.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/google/wire v0.3.0 // indirect
	github.com/googleapis/gax-go v2.0.2+incompatible // indirect
	github.com/googleapis/gax-go/v2 v2.12.0 // indirect
	github.com/googleapis/gnostic v0.5.1 // indirect
	github.com/gregjones/httpcache v0.0.0-20190212212710-3befbb6ad0cc // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/imdario/mergo v0.3.10 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/json-iterator/go v1.1.11 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.3 // indirect
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de // indirect
	github.com/mailru/easyjson v0.7.0 // indirect
	github.com/mattn/go-ieproxy v0.0.0-20190610004146-91bb50d98149 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/minio/sha256-simd v0.1.1 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.0 // indirect
	github.com/moby/term v0.0.0-20200312100748-672ec06f55cd // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/mohae/deepcopy v0.0.0-20170603005431-491d3605edfb // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nwaples/rardecode v1.0.0 // indirect
	github.com/nxadm/tail v1.4.4 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pierrec/lz4 v2.0.5+incompatible // indirect
	github.com/pingcap/check v0.0.0-20190102082844-67f458068fc8 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/procfs v0.6.0 // indirect
	github.com/prometheus/prometheus v1.8.2 // indirect
	github.com/russross/blackfriday v1.5.2 // indirect
	github.com/ulikunitz/xz v0.5.6 // indirect
	github.com/vmihailenco/msgpack v4.0.4+incompatible // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.5 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	go.uber.org/zap v1.23.0 // indirect
	golang.org/x/crypto v0.14.0 // indirect
	golang.org/x/net v0.17.0 // indirect
	golang.org/x/oauth2 v0.11.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/term v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	google.golang.org/api v0.128.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20231012201019-e917dd12ba7a // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/gcfg.v1 v1.2.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.51.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/warnings.v0 v0.1.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/cloud-provider v0.20.15 // indirect
	k8s.io/csi-translation-lib v0.20.15 // indirect
	k8s.io/klog v1.0.0 // indirect
	k8s.io/kube-openapi v0.0.0-20211110013926-83f114cd0513 // indirect
	k8s.io/legacy-cloud-providers v0.0.0 // indirect
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.0.22 // indirect
	sigs.k8s.io/kustomize v2.0.3+incompatible // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.1.2 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)

replace github.com/pingcap/tidb-operator/pkg/apis => ./pkg/apis

replace github.com/pingcap/tidb-operator/pkg/client => ./pkg/client

replace github.com/renstrom/dedent => github.com/lithammer/dedent v1.1.0

replace k8s.io/api => k8s.io/api v0.20.15

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.20.15

replace k8s.io/apimachinery => k8s.io/apimachinery v0.20.15

replace k8s.io/apiserver => k8s.io/apiserver v0.20.15

replace k8s.io/cli-runtime => k8s.io/cli-runtime v0.20.15

replace k8s.io/client-go => k8s.io/client-go v0.20.15

replace k8s.io/code-generator => k8s.io/code-generator v0.20.15

replace k8s.io/csi-api => k8s.io/csi-api v0.0.0-20190118125032-c4557c74373f

replace k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.20.15

replace k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.20.15

replace k8s.io/kube-proxy => k8s.io/kube-proxy v0.20.15

replace k8s.io/kubelet => k8s.io/kubelet v0.20.15

replace k8s.io/metrics => k8s.io/metrics v0.20.15

replace k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.20.15

replace k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.20.15

replace k8s.io/sample-controller => k8s.io/sample-controller v0.20.15

replace k8s.io/cloud-provider => k8s.io/cloud-provider v0.20.15

replace k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.20.15

replace k8s.io/component-base => k8s.io/component-base v0.20.15

replace k8s.io/cri-api => k8s.io/cri-api v0.20.15

replace k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.20.15

replace k8s.io/kubectl => k8s.io/kubectl v0.20.15

replace k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.20.15

replace k8s.io/node-api => k8s.io/node-api v0.0.0-20190918163711-2299658ad911

replace github.com/uber-go/atomic => go.uber.org/atomic v1.5.0

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v14.2.0+incompatible

replace github.com/prometheus/client_golang => github.com/prometheus/client_golang v1.11.1

replace k8s.io/controller-manager => k8s.io/controller-manager v0.20.15

replace k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.20.15

replace k8s.io/component-helpers => k8s.io/component-helpers v0.20.15

replace k8s.io/mount-utils => k8s.io/mount-utils v0.20.15

// workaround for https://github.com/kubernetes/apiserver/issues/65
// controller-rutime v0.7.2 use github.com/googleapis/gnostic v0.5.0, kube-apiserver v1.19 use github.com/googleapis/gnostic v0.4.1
// so downgrade github.com/googleapis/gnostic to v0.4.1
// TODO: remove it after upgrading kubernetes dependency to v1.22
replace github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.1

// workaround for github.com/advisories/GHSA-25xm-hr59-7c27
// TODO: remove it after upgrading github.com/mholt/archiver greater than v3.5.0
replace github.com/ulikunitz/xz => github.com/ulikunitz/xz v0.5.8

// workaround for github.com/advisories/GHSA-w73w-5m7g-f7qc
// TODO: remove it after upgrading k8s.io/client-go equal or greater than v0.20.0
replace github.com/dgrijalva/jwt-go => github.com/golang-jwt/jwt v3.2.1+incompatible

// workaround for "but does not contain package google.golang.org/grpc/naming"

replace google.golang.org/grpc => google.golang.org/grpc v1.27.1

replace cloud.google.com/go/storage => cloud.google.com/go/storage v1.6.0

replace google.golang.org/api => google.golang.org/api v0.46.0
