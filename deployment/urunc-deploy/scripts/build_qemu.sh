#!/usr/bin/env bash
#
# Copyright (c) 2022 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

ARCH=${ARCH:-$(uname -m)}

QEMU_REPO=https://github.com/qemu/qemu
QEMU_VERSION_NUM=v9.1.2
PREFIX=/opt/urunc

git clone --depth=1 "${QEMU_REPO}" qemu
pushd qemu
git fetch --depth=1 origin "${QEMU_VERSION_NUM}"
git checkout FETCH_HEAD
scripts/git-submodule.sh update meson capstone
apt install -y ninja-build libglib2.0-dev libaio-dev liburing-dev libseccomp-dev libcap-ng-dev librados-dev librbd-dev
if [ $ARCH == 'x86_64' ];
then
	./configure --disable-brlapi --disable-docs --disable-curses --disable-gtk --disable-opengl --disable-sdl --disable-spice --disable-vte --disable-vnc --disable-vnc-jpeg --disable-png --disable-vnc-sasl --disable-auth-pam --disable-glusterfs --disable-libiscsi --disable-libnfs --disable-libssh --disable-bzip2 --disable-lzo --disable-snappy --disable-slirp --disable-libusb --disable-usb-redir --disable-tcg --static --disable-debug-tcg --disable-tcg-interpreter --disable-qom-cast-debug --disable-libudev --disable-curl --disable-rdma --disable-tools --enable-virtfs --disable-bsd-user --disable-linux-user --disable-sparse --disable-vde --disable-nettle --disable-xen --disable-capstone --disable-virglrenderer --disable-replication --disable-smartcard --disable-guest-agent --disable-guest-agent-msi --disable-vvfat --disable-vdi --disable-qed --disable-qcow1 --disable-bochs --disable-cloop --disable-dmg --disable-parallels --disable-colo-proxy --disable-debug-graph-lock --disable-hexagon-idef-parser --disable-libdw --disable-pipewire --disable-pixman --disable-relocatable --disable-rutabaga-gfx --disable-vmdk --disable-avx512bw --disable-vpc --disable-vhdx --disable-hv-balloon --disable-qpl --disable-uadk --disable-debug-remap --disable-gio --disable-libdaxctl --disable-oss --enable-kvm --enable-vhost-net --enable-linux-aio --enable-linux-io-uring --enable-virtfs --enable-attr --enable-cap-ng --enable-seccomp --enable-avx2 --enable-avx512bw --disable-libpmem --disable-rbd --enable-malloc-trim --target-list=x86_64-softmmu --extra-cflags=" -O2 -fno-semantic-interposition -falign-functions=32 -D_FORTIFY_SOURCE=2" --extra-ldflags=" -z noexecstack -z relro -z now" --prefix=$PREFIX --libdir=$PREFIX/lib/qemu --libexecdir=$PREFIX/libexec/qemu --datadir=$PREFIX/share/qemu
else
	./configure --disable-brlapi --disable-docs --disable-curses --disable-gtk --disable-opengl --disable-sdl --disable-spice --disable-vte --disable-vnc --disable-vnc-jpeg --disable-png --disable-vnc-sasl --disable-auth-pam --disable-glusterfs --disable-libiscsi --disable-libnfs --disable-libssh --disable-bzip2 --disable-lzo --disable-snappy --disable-slirp --disable-libusb --disable-usb-redir --static --disable-qom-cast-debug --disable-libudev --disable-curl --disable-rdma --disable-tools --enable-virtfs --disable-bsd-user --disable-linux-user --disable-sparse --disable-vde --disable-nettle --disable-xen --disable-capstone --disable-virglrenderer --disable-replication --disable-smartcard --disable-guest-agent --disable-guest-agent-msi --disable-vvfat --disable-vdi --disable-qed --disable-qcow1 --disable-bochs --disable-cloop --disable-dmg --disable-parallels --disable-colo-proxy --disable-debug-graph-lock --disable-hexagon-idef-parser --disable-libdw --disable-pipewire --disable-pixman --disable-relocatable --disable-rutabaga-gfx --disable-vmdk --disable-avx512bw --disable-vpc --disable-vhdx --disable-hv-balloon --disable-qpl --disable-uadk --disable-debug-remap --disable-gio --disable-libdaxctl --disable-oss --disable-pie --enable-kvm --enable-vhost-net --enable-linux-aio --enable-linux-io-uring --enable-virtfs --enable-attr --enable-cap-ng --enable-seccomp --disable-avx2 --disable-libpmem --disable-rbd --enable-malloc-trim --target-list=aarch64-softmmu --extra-cflags=" -O2 -fno-semantic-interposition -falign-functions=32 -D_FORTIFY_SOURCE=2" --prefix=$PREFIX --libdir=$PREFIX/lib/qemu --libexecdir=$PREFIX/libexec/qemu --datadir=$PREFIX/share/qemu
fi
make -j"$(nproc +--ignore 1)"
make install
popd
