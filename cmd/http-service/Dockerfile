FROM ghcr.io/pingcap-qe/bases/pingcap-base:v1.11.0@sha256:7b3a5c5df7a323a35d2a99ea574ca1477c113ebc669bc4f054cf49da47615845

ARG TARGETARCH
RUN dnf install -y tzdata bind-utils && dnf clean all
ADD bin/${TARGETARCH}/http-service /usr/local/bin/http-service

CMD ["/usr/local/bin/http-service"]
