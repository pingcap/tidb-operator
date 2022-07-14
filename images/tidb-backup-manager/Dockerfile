FROM pingcap/alpine-glibc:alpine-3.14.3-gcompat
ARG TARGETARCH
ARG RCLONE_VERSION=v1.51.0
ARG SHUSH_VERSION=v1.4.0
ARG TOOLKIT_VERSION=v6.1.0
RUN apk update && apk add ca-certificates wget unzip bind-tools --no-cache && rm -rf /var/cache/apk/*

RUN wget -nv https://github.com/ncw/rclone/releases/download/${RCLONE_VERSION}/rclone-${RCLONE_VERSION}-linux-${TARGETARCH}.zip \
  && unzip rclone-${RCLONE_VERSION}-linux-${TARGETARCH}.zip \
  && mv rclone-${RCLONE_VERSION}-linux-${TARGETARCH}/rclone /usr/local/bin \
  && chmod 755 /usr/local/bin/rclone \
  && rm -rf rclone-${RCLONE_VERSION}-linux-${TARGETARCH}.zip rclone-${RCLONE_VERSION}-linux-${TARGETARCH}

RUN wget -nv https://github.com/realestate-com-au/shush/releases/download/${SHUSH_VERSION}/shush_linux_${TARGETARCH} \
  && mv shush_linux_${TARGETARCH} /usr/local/bin/shush \
  && chmod 755 /usr/local/bin/shush

RUN \
  wget -nv https://download.pingcap.org/tidb-community-toolkit-${TOOLKIT_VERSION}-linux-${TARGETARCH}.tar.gz -O tidb-toolkit-${TOOLKIT_V40}-linux-amd64.tar.gz \
  && tar -xzf tidb-toolkit-${TOOLKIT_VERSION}-linux-${TARGETARCH}.tar.gz \
  && mv tidb-toolkit-${TOOLKIT_VERSION}-linux-${TARGETARCH}/bin/tidb-lightning /tidb-lightning \
  && mv tidb-toolkit-${TOOLKIT_VERSION}-linux-${TARGETARCH}/bin/tidb-lightning-ctl /tidb-lightning-ctl \
  && mv tidb-toolkit-${TOOLKIT_VERSION}-linux-${TARGETARCH}/bin/dumpling /dumpling \
  && chmod 755 /dumpling /tidb-lightning /tidb-lightning-ctl \
  && rm -rf tidb-toolkit-${TOOLKIT_VERSION}-linux-${TARGETARCH}.tar.gz \
  && rm -rf tidb-toolkit-${TOOLKIT_VERSION}-linux-${TARGETARCH}

COPY bin/${TARGETARCH}/tidb-backup-manager /tidb-backup-manager
COPY entrypoint.sh /entrypoint.sh


ENTRYPOINT ["/entrypoint.sh"]
