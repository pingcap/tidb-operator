FROM ghcr.io/pingcap-qe/bases/pingcap-base:v1.9.1

RUN dnf install -y \
    file \
    gdb \
    git \
    iotop \
    mysql \
    net-tools \
    perf \
    perl \
    procps-ng \
    psmisc \
    strace \
    sysstat \
    tree \
    tcpdump \
    unzip \
    vim \
    wget \
    which \
    && dnf clean all

RUN wget -q http://download.pingcap.org/tidb-latest-linux-amd64.tar.gz \
    && tar xzf tidb-latest-linux-amd64.tar.gz \
    && mv tidb-*-linux-amd64/bin/* /usr/local/bin/ \
    && rm -rf tidb-latest-linux-amd64.tar.gz tidb-*-linux-amd64

RUN wget https://github.com/brendangregg/FlameGraph/archive/master.zip \
    && unzip master.zip \
    && mv FlameGraph-master /opt/FlameGraph \
    && rm master.zip
COPY run_flamegraph.sh /run_flamegraph.sh
COPY gdbinit /root/.gdbinit

# used for go pprof
ENV GOLANG_VERSION 1.21.13
ENV GOLANG_DOWNLOAD_URL https://go.dev/dl/go$GOLANG_VERSION.linux-amd64.tar.gz
ENV GOLANG_DOWNLOAD_SHA256 502fc16d5910562461e6a6631fb6377de2322aad7304bf2bcd23500ba9dab4a7
RUN curl -fsSL "$GOLANG_DOWNLOAD_URL" -o golang.tar.gz \
    && echo "$GOLANG_DOWNLOAD_SHA256  golang.tar.gz" | sha256sum -c - \
    && tar -C /usr/local -xzf golang.tar.gz \
    && rm golang.tar.gz
ENV GOPATH /go
ENV GOROOT /usr/local/go
ENV PATH $GOPATH/bin:$GOROOT/bin:$PATH

# build delve
RUN go install github.com/go-delve/delve/cmd/dlv@v1.23.0

ADD banner /etc/banner
ADD profile  /etc/profile

ENTRYPOINT ["/bin/bash", "-l"]
