FROM ghcr.io/pingcap-qe/ci/tidb-operator_e2e-base:v2026.6.28-7-g6f649337

ARG TARGETARCH
ADD minio /minio

ADD tidb-operator /charts/e2e/tidb-operator
ADD tidb-drainer /charts/e2e/tidb-drainer
ADD manifests /manifests/e2e

ADD bin/ginkgo /usr/local/bin/
ADD bin/e2e.test /usr/local/bin/
ADD bin/webhook /usr/local/bin/
ADD bin/blockwriter /usr/local/bin/

COPY tikv.toml /etc/tikv.toml

ADD entrypoint.sh /usr/local/bin
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
