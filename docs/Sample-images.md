# Sample Unikernel OCI images

In this document, you can find the images used to perform `urunc`'s end-to-end tests.
This might be helpful for anyone looking to spawn some example unikernels using `urunc`.

The naming convention used for these images is $APPLICATION-$HYPERVISOR-$UNIKERNEL-$ADDITIONAL_INFO:tag
We plan to create and maintain multi-platform images soon, as well as enrich this list with new images.

- harbor.nbfc.io/nubificus/urunc/hello-hvt-rumprun-nonet:latest
- harbor.nbfc.io/nubificus/urunc/hello-spt-rumprun-nonet:latest
- harbor.nbfc.io/nubificus/urunc/nginx-qemu-unikraft-initrd:latest
- harbor.nbfc.io/nubificus/urunc/nginx-hvt-rumprun:latest
- harbor.nbfc.io/nubificus/urunc/nginx-firecracker-unikraft-initrd:latest
- harbor.nbfc.io/nubificus/urunc/hello-hvt-rumprun:latest
- harbor.nbfc.io/nubificus/urunc/redis-hvt-rumprun:latest
- harbor.nbfc.io/nubificus/urunc/redis-hvt-rumprun-block:latest
- harbor.nbfc.io/nubificus/urunc/redis-spt-rumprun:latest
- harbor.nbfc.io/nubificus/urunc/redis-qemu-unikraft-initrd:latest
- harbor.nbfc.io/nubificus/urunc/nginx-firecracker-unikraft-initrd:latest
- harbor.nbfc.io/nubificus/urunc/httpreply-firecracker-unikraft:latest

