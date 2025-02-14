#!/usr/bin/env bash

IFS=' ' read -a hypervisors <<< "$HYPERVISORS"
# for hypervisor in "${hypervisors[@]}" ; do
#     echo "Testing on $hypervisor"
# done
for hypervisor in "${hypervisors[@]}" ; do
        echo "Testing on $hypervisor"
        case "$hypervisor" in
        qemu)
            echo "Installing qemu"
            # install_artifact /urunc-artifacts/hypervisors/qemu-system-x86_64 /host/usr/local/bin/qemu-system-x86_64
            ;;
        *) 
            echo "Unsupported hypervisor: $hypervisor"
            ;;
	    esac
done