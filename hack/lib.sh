#!/usr/bin/env bash

if [ -z "$ROOT" ]; then
    echo "error: ROOT should be initialized"
    exit 1
fi

OS=$(go env GOOS)
ARCH=$(go env GOARCH)
OUTPUT=${ROOT}/output
OUTPUT_BIN=${OUTPUT}/bin
TERRAFORM_BIN=${OUTPUT_BIN}/terraform
TERRAFORM_VERSION=0.12.12
KUBECTL_VERSION=1.12.10
KUBECTL_BIN=$OUTPUT_BIN/kubectl
HELM_BIN=$OUTPUT_BIN/helm
HELM_VERSION=2.9.1

test -d "$OUTPUT_BIN" || mkdir -p "$OUTPUT_BIN"

function hack::verify_terraform() {
    if test -x "$TERRAFORM_BIN"; then
        local v=$($TERRAFORM_BIN version | awk '/^Terraform v.*/ { print $2 }' | sed 's#^v##')
        [[ "$v" == "$TERRAFORM_VERSION" ]]
        return
    fi
    return 1
}

function hack::install_terraform() {
    echo "Installing terraform v$TERRAFORM_VERSION..."
    local tmpdir=$(mktemp -d)
    trap "test -d $tmpdir && rm -r $tmpdir" RETURN
    pushd $tmpdir > /dev/null
    wget https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_${OS}_${ARCH}.zip
    unzip terraform_${TERRAFORM_VERSION}_${OS}_${ARCH}.zip
    mv terraform $TERRAFORM_BIN
    popd > /dev/null
    chmod +x $TERRAFORM_BIN
}

function hack::ensure_terraform() {
    if ! hack::verify_terraform; then
        hack::install_terraform
    fi
}

function hack::verify_kubectl() {
    if test -x "$KUBECTL_BIN"; then
        [[ "$($KUBECTL_BIN version --client --short | grep -o -P '\d+\.\d+\.\d+')" == "$KUBECTL_VERSION" ]]
        return
    fi
    return 1
}

function hack::ensure_kubectl() {
    if hack::verify_kubectl; then
        return 0
    fi
    echo "Installing kubectl v$KUBECTL_VERSION..."
    tmpfile=$(mktemp)
    trap "test -f $tmpfile && rm $tmpfile" RETURN
    curl -Lo $tmpfile https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VERSION}/bin/${OS}/${ARCH}/kubectl
    mv $tmpfile $KUBECTL_BIN
    chmod +x $KUBECTL_BIN
}

function hack::verify_helm() {
    if test -x "$HELM_BIN"; then
        local v=$($HELM_BIN version --short --client | grep -o -P '\d+\.\d+\.\d+')
        [[ "$v" == "$HELM_VERSION" ]]
        return
    fi
    return 1
}

function hack::ensure_helm() {
    if hack::verify_helm; then
        return 0
    fi
    local HELM_URL=http://storage.googleapis.com/kubernetes-helm/helm-v${HELM_VERSION}-${OS}-${ARCH}.tar.gz
    curl -s "$HELM_URL" | tar --strip-components 1 -C $OUTPUT_BIN -zxf - ${OS}-${ARCH}/helm
}
