FROM docker.io/library/debian:bullseye AS solo5-builder

# This resolves a segmentation fault.
# See more: https://stackoverflow.com/questions/78105004/docker-build-fails-because-unable-to-install-libc-bins
RUN rm /var/lib/dpkg/info/libc-bin.*
RUN DEBIAN_FRONTEND=noninteractive apt-get clean
RUN DEBIAN_FRONTEND=noninteractive apt-get update
RUN DEBIAN_FRONTEND=noninteractive apt-get install libc-bin

RUN DEBIAN_FRONTEND=noninteractive apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y \
    build-essential libseccomp-dev git pkg-config
WORKDIR /app
RUN git -c http.sslVerify=false clone --branch v0.6.9 https://github.com/Solo5/solo5.git . && \
    sed -i '66 a\HOSTLDFLAGS += -static' tenders/GNUmakefile
RUN ./configure.sh && \
    make || true
RUN test -f tenders/spt/solo5-spt && test -f tenders/hvt/solo5-hvt
WORKDIR /artifacts
RUN cp /app/tenders/hvt/solo5-hvt .
RUN cp /app/tenders/spt/solo5-spt .


FROM docker.io/library/golang:alpine3.21 AS urunc-builder
RUN apk update && \
apk add --no-cache git make
WORKDIR /app
ARG BRANCH=main
RUN git clone --branch ${BRANCH} https://github.com/nubificus/urunc.git .
RUN make
WORKDIR /artifacts
RUN cp -r /app/dist .


FROM docker.io/library/alpine:3.21 AS firecracker-builder
RUN apk update && \
apk add --no-cache curl tar
RUN ARCH=$(uname -m) && \
    VERSION="v1.7.0" && \
    RELEASE_URL="https://github.com/firecracker-microvm/firecracker/releases" && \
    curl -L -o firecracker.tgz ${RELEASE_URL}/download/${VERSION}/firecracker-${VERSION}-${ARCH}.tgz && \
    tar xzf firecracker.tgz && \
    mv release-${VERSION}-${ARCH}/firecracker-${VERSION}-${ARCH} firecracker
WORKDIR /artifacts
RUN cp /firecracker .



FROM quay.io/kata-containers/kata-deploy:latest AS qemu-builder
WORKDIR /artifacts
RUN ARCH=$(uname -m) && \
    cp /opt/kata-artifacts/opt/kata/bin/qemu-system-${ARCH} ./qemu-system


FROM docker.io/library/alpine:3.21
RUN apk update && \
apk add --no-cache bash curl && \
    DEBIAN_ARCH=$(uname -m) && \
    if [ "${DEBIAN_ARCH}" = "x86_64" ]; then DEBIAN_ARCH=amd64; fi && \
    if [ "${DEBIAN_ARCH}" = "aarch64" ]; then DEBIAN_ARCH=arm64; fi && \
    curl -fL --progress-bar -o /usr/bin/jq https://github.com/jqlang/jq/releases/download/jq-1.7.1/jq-linux-${DEBIAN_ARCH} && \
    chmod +x /usr/bin/jq && \
    apk --no-cache add py3-pip && \
    pip install --no-cache-dir --break-system-packages yq==3.2.3
COPY scripts /urunc-artifacts/scripts
COPY runtimeclasses /urunc-artifacts/runtimeclasses
RUN chmod +x /urunc-artifacts/scripts/install.sh

COPY --from=solo5-builder /artifacts /urunc-artifacts/hypervisors
COPY --from=qemu-builder /artifacts /urunc-artifacts/hypervisors
COPY --from=firecracker-builder /artifacts /urunc-artifacts/hypervisors
COPY --from=urunc-builder /artifacts/dist /urunc-artifacts

RUN ARCH=$(uname -m) && \
    mv /urunc-artifacts/hypervisors/qemu-system /urunc-artifacts/hypervisors/qemu-system-${ARCH}

RUN DEBIAN_ARCH=$(uname -m) && \
if [ "${DEBIAN_ARCH}" = "x86_64" ]; then DEBIAN_ARCH=amd64; fi && \
if [ "${DEBIAN_ARCH}" = "aarch64" ]; then DEBIAN_ARCH=arm64; fi && \
mv /urunc-artifacts/containerd-shim-urunc-v2_${DEBIAN_ARCH} /urunc-artifacts/containerd-shim-urunc-v2 && \
mv /urunc-artifacts/urunc_static_${DEBIAN_ARCH} /urunc-artifacts/urunc
