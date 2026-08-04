FROM ghcr.io/pingcap-qe/bases/pingcap-base:v1.12.1@sha256:0afd78a822c3ac0d0d572d4f43b434981870a7a79d577fe20238536659605b51

ARG TARGETARCH
RUN dnf install -y tzdata bind-utils && dnf clean all
ADD bin/${TARGETARCH}/http-service /usr/local/bin/http-service

CMD ["/usr/local/bin/http-service"]
