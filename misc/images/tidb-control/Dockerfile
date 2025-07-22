FROM bash:5.2.37

ARG TIDB_VERSION=v8.5.2
RUN OS=linux; ARCH=$([ "$(arch)" = "x86_64" ] && echo amd64 || echo arm64); \
      wget -O- https://tiup-mirrors.pingcap.com/ctl-${TIDB_VERSION}-${OS}-${ARCH}.tar.gz | tar -C /usr/local/bin -xzvf - pd-ctl tidb-ctl
ADD banner profile /etc/

CMD ["/usr/local/bin/bash", "-l"]
