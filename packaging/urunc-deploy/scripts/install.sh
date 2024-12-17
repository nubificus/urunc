#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

function host_systemctl() {
	nsenter --target 1 --mount systemctl "${@}"
}
function print_usage() {
	echo "Usage: $0 [install/cleanup/reset]"
}
function install_urunc() {
    cp /urunc-artifacts/urunc /host/usr/local/bin/urunc
    chmod +x /host/usr/local/bin/urunc
}

function install_shim() {
    cp /urunc-artifacts/containerd-shim-v2-urunc /host/usr/local/bin/containerd-shim-v2-urunc
    chmod +x /host/usr/local/bin/containerd-shim-v2-urunc
}

function install_qemu() {
    cp /urunc-artifacts/qemu-system-x86_64 /host/usr/local/bin/qemu-system-x86_64
    chmod +x /host/usr/local/bin/qemu-system-x86_64
}

function install_firecracker() {
    cp /urunc-artifacts/firecracker /host/usr/local/bin/firecracker
    chmod +x /host/usr/local/bin/firecracker
}

function uninstall_urunc() {
    rm -f /host/usr/local/bin/urunc
}

function uninstall_shim() {
    rm -f /host/usr/local/bin/containerd-shim-v2-urunc
}

function uninstall_qemu() {
    rm -f /host/usr/local/bin/qemu-system-x86_64
}

function uninstall_firecracker() {
    rm -f /host/usr/local/bin/firecracker
}

function setup() {
    echo "copying urunc artifacts onto host"
    mkdir -p /host/usr/local/bin
    install_urunc
    install_shim
    install_qemu
    install_firecracker
}

function uninstall() {
    uninstall_urunc
    uninstall_shim
    uninstall_qemu
    uninstall_firecracker
}

function print_usage() {
    echo "Please provide a valid action"
    echo "\t install"
    echo "\t uninstall"
}

die() {
    msg="$*"
    echo "ERROR: $msg" >&2
    exit 1
}

function main() {
    action=${1:-}
    if [ -z "$action" ]; then
        print_usage
        die "invalid arguments"
    fi
    case "$action" in
    install)
        setup
        ;;
    cleanup)
        uninstall
        ;;
    *)
        print_usage
        die "invalid arguments"
        ;;
    esac
}

main "$@"
