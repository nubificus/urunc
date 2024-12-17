# Get solo5 tenders
FROM ubuntu:22.04 AS solo5-builder

RUN DEBIAN_FRONTEND=noninteractive apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y build-essential \
    libseccomp-dev git pkg-config

RUN git clone -b v0.6.9 https://github.com/Solo5/solo5.git && \
    cd solo5 && \
    ./configure.sh && \
    make

# Get firecracker binary
FROM ubuntu:22.04 AS firecracker-builder
RUN DEBIAN_FRONTEND=noninteractive apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y curl tar
RUN \
    ARCH=$(uname -m) && \
    VERSION="v1.7.0" && \
    RELEASE_URL="https://github.com/firecracker-microvm/firecracker/releases" && \
    curl -L -o firecracker.tgz ${RELEASE_URL}/download/${VERSION}/firecracker-${VERSION}-${ARCH}.tgz && \
    tar xzf firecracker.tgz && \
    mv release-${VERSION}-${ARCH}/firecracker-${VERSION}-${ARCH} firecracker

# $ # Rename the binary to "firecracker"
# $ sudo mv release-${VERSION}-${ARCH}/firecracker-${VERSION}-${ARCH} /usr/local/bin/firecracker
# $ rm -fr release-${VERSION}-${ARCH}

# Get Qemu binary
FROM quay.io/kata-containers/kata-deploy:latest AS qemu-builder

# Final image with artifacts and urunc-deploy.sh script
FROM alpine:3.18

COPY --from=solo5-builder /solo5/tenders/spt/solo5-spt /urunc-artifacts/solo5-spt
COPY --from=solo5-builder /solo5/tenders/hvt/solo5-hvt /urunc-artifacts/solo5-hvt

COPY --from=firecracker-builder /firecracker /urunc-artifacts/firecracker

COPY --from=qemu-builder /opt/kata-artifacts/opt/kata/bin/qemu-system-x86_64 /urunc-artifacts/qemu-system-x86_64

RUN apk --no-cache add bash curl
WORKDIR /urunc-artifacts
RUN \
    ARCH=$(uname -m) && \
    if [ "${ARCH}" = "x86_64" ]; then ARCH=amd64; fi && \
    if [ "${ARCH}" = "aarch64" ]; then ARCH=arm64; fi && \
    DEBIAN_ARCH=${ARCH} && \
	if [ "${DEBIAN_ARCH}" = "ppc64le" ]; then DEBIAN_ARCH=ppc64el; fi && \
    curl -fL --progress-bar -o /usr/bin/kubectl https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/${ARCH}/kubectl && \
	chmod +x /usr/bin/kubectl && \
    curl -L -o /urunc-artifacts/urunc https://github.com/nubificus/urunc/releases/download/v0.4.0/urunc_${ARCH} && \
    curl -L -o /urunc-artifacts/containerd-shim-v2-urunc \
        https://github.com/nubificus/urunc/releases/download/v0.4.0/containerd-shim-urunc-v2_${ARCH} && \
    curl -fL --progress-bar -o /usr/bin/jq https://github.com/jqlang/jq/releases/download/jq-1.7.1/jq-linux-${DEBIAN_ARCH} && \
    chmod +x /usr/bin/jq && \
    apk --no-cache add py3-pip && \
    pip install --no-cache-dir yq==3.2.3

COPY scripts /urunc-artifacts/scripts
RUN chmod +x /urunc-artifacts/scripts/install.sh

