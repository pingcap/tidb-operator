FROM ghcr.io/pingcap-qe/bases/pingcap-base:v1.12.5@sha256:715150b855957b5a007b006240b2735b3e645c5a471d2391ad9ffa4aafdb93fd

ARG TARGETARCH
RUN dnf install -y tzdata bind-utils && dnf clean all
ADD bin/${TARGETARCH}/http-service /usr/local/bin/http-service

CMD ["/usr/local/bin/http-service"]
