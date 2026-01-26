FROM ghcr.io/pingcap-qe/bases/pingcap-base:v1.11.0@sha256:7b3a5c5df7a323a35d2a99ea574ca1477c113ebc669bc4f054cf49da47615845
ARG TARGETARCH
ARG RCLONE_VERSION=v1.71.2
ARG SHUSH_VERSION=v1.5.5
RUN dnf install -y ca-certificates bind-utils wget nc unzip && dnf clean all

RUN wget -nv https://github.com/ncw/rclone/releases/download/${RCLONE_VERSION}/rclone-${RCLONE_VERSION}-linux-${TARGETARCH}.zip \
    && unzip rclone-${RCLONE_VERSION}-linux-${TARGETARCH}.zip \
    && mv rclone-${RCLONE_VERSION}-linux-${TARGETARCH}/rclone /usr/local/bin \
    && chmod 755 /usr/local/bin/rclone \
    && rm -rf rclone-${RCLONE_VERSION}-linux-${TARGETARCH}.zip rclone-${RCLONE_VERSION}-linux-${TARGETARCH}

RUN wget -nv https://github.com/realestate-com-au/shush/releases/download/${SHUSH_VERSION}/shush_linux_${TARGETARCH} \
    && mv shush_linux_${TARGETARCH} /usr/local/bin/shush \
    && chmod 755 /usr/local/bin/shush

COPY --from=pingcap/tidb-lightning:v8.5.5 /tidb-lightning /tidb-lightning-ctl /
COPY --from=pingcap/dumpling:v8.5.5 /dumpling /dumpling

COPY bin/${TARGETARCH}/tidb-backup-manager /tidb-backup-manager
COPY entrypoint.sh /entrypoint.sh


ENTRYPOINT ["/entrypoint.sh"]
