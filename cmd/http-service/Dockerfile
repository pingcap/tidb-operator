FROM ghcr.io/pingcap-qe/bases/pingcap-base:v1.9.1

ARG TARGETARCH
RUN dnf install -y tzdata bind-utils && dnf clean all
ADD bin/${TARGETARCH}/http-service /usr/local/bin/http-service

CMD ["/usr/local/bin/http-service"]
