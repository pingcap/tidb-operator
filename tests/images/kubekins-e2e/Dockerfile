FROM debian:bookworm-slim

ARG GO_VERSION=1.23.4

# common util tools
RUN apt-get update && \
    apt-get install -y build-essential ca-certificates curl git openssl default-mysql-client unzip wget && \
    apt-get install -y python3 && \
    apt-get autoremove -y && \
    apt-get clean -y

#
# BEGIN: DOCKER IN DOCKER SETUP
#

# Install Docker deps, some of these are already installed in the image but
# that's fine since they won't re-install and we can reuse the code below
# for another image someday.
RUN apt-get update && apt-get install -y --no-install-recommends \
    apt-transport-https \
    ca-certificates \
    curl \
    gnupg2 \
    software-properties-common \
    lsb-release && \
    rm -rf /var/lib/apt/lists/*

# Add the Docker apt-repository
RUN curl -fsSL https://download.docker.com/linux/$(. /etc/os-release; echo "$ID")/gpg \
    | apt-key add - && \
    add-apt-repository \
    "deb [arch=amd64] https://download.docker.com/linux/$(. /etc/os-release; echo "$ID") \
    $(lsb_release -cs) stable"

# Install Docker
# TODO: the `sed` is a bit of a hack, look into alternatives.
# Why this exists: `docker service start` on debian runs a `cgroupfs_mount` method,
# We're already inside docker though so we can be sure these are already mounted.
# Trying to remount these makes for a very noisy error block in the beginning of
# the pod logs, so we just comment out the call to it... :shrug:
RUN apt-get update && \
    apt-get install -y --no-install-recommends docker-ce=5:20.10.* && \
    apt-get install docker-buildx-plugin -y && \
    rm -rf /var/lib/apt/lists/* && \
    sed -i 's/cgroupfs_mount$/#cgroupfs_mount\n/' /etc/init.d/docker \
    && update-alternatives --set iptables /usr/sbin/iptables-legacy \
    && update-alternatives --set ip6tables /usr/sbin/ip6tables-legacy

# Move Docker's storage location
RUN echo 'DOCKER_OPTS="${DOCKER_OPTS} --data-root=/docker-graph"' | \
    tee --append /etc/default/docker
# NOTE this should be mounted and persisted as a volume ideally (!)
# We will make a fallback one now just in case
RUN mkdir /docker-graph

# configure dockerd to use mirror.gcr.io
ARG DOCKER_REGISTRY_MIRROR_URL=https://registry-mirror.pingcap.net
RUN echo "DOCKER_OPTS=\"\${DOCKER_OPTS} --registry-mirror=${DOCKER_REGISTRY_MIRROR_URL}\"" | \
    tee --append /etc/default/docker

#
# END: DOCKER IN DOCKER SETUP
#

#
# BEGIN: Install GO
#

ENV GOPATH /go
ENV PATH /usr/local/go/bin:$PATH
RUN mkdir -p /go/bin
ENV PATH $GOPATH/bin:$PATH

ARG GO_TARBALL="go${GO_VERSION}.linux-amd64.tar.gz"
RUN wget -q "https://storage.googleapis.com/golang/${GO_TARBALL}" && \
    tar xzf "${GO_TARBALL}" -C /usr/local && \
    rm "${GO_TARBALL}"

#
# END: Install GO
#

COPY ["runner.sh", "/usr/local/bin/"]
