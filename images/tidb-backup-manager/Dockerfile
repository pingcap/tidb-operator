FROM pingcap/tidb-enterprise-tools:latest

ARG VERSION=v1.48.0
RUN apk update && apk add curl ca-certificates

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

RUN curl -sL -o /usr/local/bin/shush \
  https://github.com/realestate-com-au/shush/releases/download/v1.4.0/shush_linux_amd64 \
  && chmod +x /usr/local/bin/shush

COPY bin/tidb-backup-manager /tidb-backup-manager
COPY entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
