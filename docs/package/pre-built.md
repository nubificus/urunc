---
layout: default
title: "Pre-built unikernels"
description: "Packaging pre-built unikernels"
---

# Packaging pre-built unikernels for `urunc`

In this page we will expalin the process of packaging an existing / pre-built
unikernel as an OCI image with the necessary annotations for `urunc`. As an
example, we will use a Redis [Rumprun](https://github.com/cloudkernels/rumprun)
unikernel from
[Rumprun-packages](https://github.com/cloudkernels/rumprun-packages) targetting
[Solo5-hvt](https://github.com/Solo5/solo5).

For simply packaging pre-built unikernel images, we can use both yunub and bnix.

Assumptions:
- We assume that we execute the commands in the same path where the unikernel
  resides
- We assume that all the files we want to copy inside the OCI image reside also
  in the same path as the unikernel.

> **NOTE**: The below steps can be easily adjusted to any pre-built unikernel image.

## Using `bunny`

In the case of `bunny` and pre-built unikernel images, we can use both supported
file syntaxes: a) `bunnyfile` and b) the Dockerfile-like syntax.

### Using a `bunnyfile`

In order to package an existing pre-built unikernel image with `bunny` and a
`bunnyfile` we can define the `bunnyfile` as:

```
#syntax=harbor.nbfc.io/nubificus/bunny:0.2.0
version: v0.1

platforms:
  framework: rumprun
  monitor: hvt
  architecture: x86

rootfs:
  from: scratch
  type: raw
  include:
  - redis.conf:/data/conf/redis.conf

kernel:
  from: local
  path: redis.hvt

cmdline: "redis-server /data/conf/redis.conf"
```

In the above file we specify the followings:
- We want to package a rumprun unikernel that will execute on top o hvt over x86
  architecture.
- We want to create a raw rootfs which includes the file `redis.conf` and placed
  in `/data/conf/redis.conf`. A raw rootfs means that we will simply copy the
  files we specify directly in the OCI image (similar to `COPY` in Dockerfile).
  Because of this type selction, `bunny` will also set up the respective
  annotations to mount the OCI images rootfs directly to the unikernel. The way
  `urunc` passes the rootfs will depend on the storage support of the respective
  unikernel framework (e.g. through shared-fs, virtio-blk).
- We want to use the `redis.hvt` binary as the unikernel to boot.
- We specify the cmdline for the unikernel as `redis-server /data/conf/redis.conf`

We can build the OCI image with the following command:

```
docker build -f bunnyfile -t urunc/prebuilt/redis-rumprun-hvt:test .
```

### Using a Dockerfile-like syntax

In order to package an existing pre-built unikernel image with `bunny` and a
Dockerfile-like syntax file, we can define the `Containerfile` as:

```
#syntax=harbor.nbfc.io/nubificus/bunny:0.2.0
FROM scratch

COPY redis.hvt /unikernel/redis.hvt
COPY redis.conf /conf/redis.conf

LABEL com.urunc.unikernel.binary=/unikernel/redis.hvt
LABEL "com.urunc.unikernel.cmdline"="redis-server /data/conf/redis.conf"
LABEL "com.urunc.unikernel.unikernelType"="rumprun"
LABEL "com.urunc.unikernel.hypervisor"="hvt"
LABEL "com.urunc.unikernel.useDMBlock"="true"

```

In the above file:
- We directly copy the unikernel binary and any files that we want to have in
  the OCI's image rootfs.
- We manually specify through labels the `urunc` annotations.

We can build the OCI image with the following command:

```
docker build -f Containerfile -t urunc/prebuilt/redis-rumprun-hvt:test .
```

## Using `bimanix`

In the case of `bimanix` we need the whole repository in the same directly as
the unikernel. Then, we simply need to edit the `args.nix` file. For our
pre-built Redis rumprun unikernel we can define the files as:

```Nix
{
  name = "urunc/prebuilt/redis-rumprun-hvt";
  tag = "test";
  files = {
    "./redis.hvt" = "/unikernel/redis.hvt";
    "./redis.conf" = "/conf/redis.conf";
  };
  annotations = {
    unikernelType = "rumprun";
    hypervisor = "hvt";
    binary = "/unikernel/redis.hvt";
    cmdline = "hello";
    unikernelVersion = "";
    initrd = "";
    block = "";
    blkMntPoint = "";
    useDMBlock = "true";
  };
}
```

In the above file:
- We directly specify the files to copy inside the OCI's image rootfs.
- We manually specify all `urunc` annotations.

We can build the OCI image by simply running the following command:

```bash
nix-build default.nix
```

The above command will create a container image in a tar inside Nix's store. For
easier access of the tar, Nix creates a symlink of the tar file in the CWD. The
symlink will be named as `result`. Therefore, we can load the container image with:

```bash
docker load < result
```
