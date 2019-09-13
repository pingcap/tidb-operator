FROM pingcap/tidb-enterprise-tools:latest

ARG VERSION=v1.48.0
RUN apk update && apk add ca-certificates

RUN wget -nv https://github.com/ncw/rclone/releases/download/${VERSION}/rclone-${VERSION}-linux-amd64.zip \
	&& unzip rclone-${VERSION}-linux-amd64.zip \
	&& mv rclone-${VERSION}-linux-amd64/rclone /usr/local/bin \
	&& chmod 755 /usr/local/bin/rclone \
	&& rm -rf rclone-${VERSION}-linux-amd64.zip rclone-${VERSION}-linux-amd64

COPY bin/tidb-backup-manager /tidb-backup-manager
COPY entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
