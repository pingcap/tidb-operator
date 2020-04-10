FROM pingcap/tidb-enterprise-tools:latest
ARG VERSION=v1.51.0
ARG SHUSH_VERSION=v1.4.0
ARG TOOLKIT_VERSION=v3.0.12
RUN apk update && apk add ca-certificates

RUN wget -nv https://github.com/ncw/rclone/releases/download/${VERSION}/rclone-${VERSION}-linux-amd64.zip \
	&& unzip rclone-${VERSION}-linux-amd64.zip \
	&& mv rclone-${VERSION}-linux-amd64/rclone /usr/local/bin \
	&& chmod 755 /usr/local/bin/rclone \
	&& rm -rf rclone-${VERSION}-linux-amd64.zip rclone-${VERSION}-linux-amd64

RUN wget -nv http://download.pingcap.org/br-latest-linux-amd64.tar.gz \
    && tar -xzf br-latest-linux-amd64.tar.gz \
    && mv bin/br /usr/local/bin \
    && chmod 755 /usr/local/bin/br \
    && rm -rf br-latest-linux-amd64.tar.gz

RUN wget -nv https://github.com/realestate-com-au/shush/releases/download/${SHUSH_VERSION}/shush_linux_amd64 \
  && mv shush_linux_amd64 /usr/local/bin/shush \
  && chmod 755 /usr/local/bin/shush

RUN \
  wget -nv https://download.pingcap.org/tidb-toolkit-${TOOLKIT_VERSION}-linux-amd64.tar.gz \
  && tar -xzf tidb-toolkit-${TOOLKIT_VERSION}-linux-amd64.tar.gz \
  && mv tidb-toolkit-${TOOLKIT_VERSION}-linux-amd64/bin/tidb-lightning /tidb-lightning \
  && mv tidb-toolkit-${TOOLKIT_VERSION}-linux-amd64/bin/tidb-lightning-ctl /tidb-lightning-ctl \
  && chmod 755 /tidb-lightning /tidb-lightning-ctl \
  && rm -rf tidb-toolkit-${TOOLKIT_VERSION}-linux-amd64.tar.gz \
  && rm -rf tidb-toolkit-${TOOLKIT_VERSION}-linux-amd64

COPY bin/tidb-backup-manager /tidb-backup-manager
COPY entrypoint.sh /entrypoint.sh


ENTRYPOINT ["/entrypoint.sh"]
