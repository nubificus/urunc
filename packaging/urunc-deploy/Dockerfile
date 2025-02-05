FROM docker.io/library/alpine:3.21 AS solo5-builder
RUN apk add --no-cache build-base libseccomp-dev git pkgconfig
WORKDIR /app
RUN git clone --branch v0.6.9 https://github.com/Solo5/solo5.git .
RUN sed -i '66 a\HOSTLDFLAGS += -static' tenders/GNUmakefile
RUN ./configure.sh
RUN  make
WORKDIR /artifacts
RUN cp /app/tenders/hvt/solo5-hvt .
RUN cp /app/tenders/spt/solo5-spt .

FROM docker.io/libray/golang:alpine3.21 as urunc-builder
RUN apk add --no-cache git make
WORKDIR /app
ARG BRANCH=main
RUN git clone --branch ${BRANCH} https://github.com/nubificus/urunc.git .
RUN make
WORKDIR /artifacts
RUN cp -r /app/dist .

FROM docker.io/library/alpine:3.21 AS firecracker-builder
RUN apk add --no-cache curl tar
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
RUN apk --no-cache add bash curl && \
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

RUN ARCH=$(uname -m) && \
    mv /urunc-artifacts/hypervisors/qemu-system /urunc-artifacts/hypervisors/qemu-system-${ARCH}