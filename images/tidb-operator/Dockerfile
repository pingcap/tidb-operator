FROM alpine:3.14

ARG TARGETARCH
ARG RCLONE_VERSION=v1.51.0
RUN apk add tzdata bind-tools --no-cache
ADD bin/${TARGETARCH}/tidb-scheduler /usr/local/bin/tidb-scheduler
ADD bin/${TARGETARCH}/tidb-discovery /usr/local/bin/tidb-discovery
ADD bin/${TARGETARCH}/tidb-controller-manager /usr/local/bin/tidb-controller-manager
ADD bin/${TARGETARCH}/tidb-admission-webhook /usr/local/bin/tidb-admission-webhook

RUN wget -nv https://github.com/ncw/rclone/releases/download/${RCLONE_VERSION}/rclone-${RCLONE_VERSION}-linux-${TARGETARCH}.zip \
  && unzip rclone-${RCLONE_VERSION}-linux-${TARGETARCH}.zip \
  && mv rclone-${RCLONE_VERSION}-linux-${TARGETARCH}/rclone /usr/local/bin \
  && chmod 755 /usr/local/bin/rclone \
  && rm -rf rclone-${RCLONE_VERSION}-linux-${TARGETARCH}.zip rclone-${RCLONE_VERSION}-linux-${TARGETARCH}