FROM ghcr.io/pingcap-qe/ci/tidb-operator_backup-manager-base:v2026.6.28-7-g6f649337
ARG TARGETARCH
RUN dnf install -y ca-certificates bind-utils wget nc unzip && dnf clean all

COPY --from=pingcap/tidb-lightning:v8.5.3 /tidb-lightning /tidb-lightning-ctl /
COPY --from=pingcap/dumpling:v8.5.3 /dumpling /dumpling

COPY bin/${TARGETARCH}/tidb-backup-manager /tidb-backup-manager
COPY entrypoint.sh /entrypoint.sh


ENTRYPOINT ["/entrypoint.sh"]
