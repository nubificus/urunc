---
layout: default
title: "Creating rootfs for unikernels"
description: "Packaging and creating unikernel's rootfs"
---

# Packaging and creating unikernel's rootfs for `urunc`

The unikernel and libOS landscape is very diverse and each framework/technology
comes with its own support for storage. The users can easily get lost on the
various storage technologies that each framework supports. In this page we will
explain how users can use our tools to create and package the rootfs for a unikernel.

For the time being, `urunc` supports two ways for passing the rootfs to the
unikernel: a) through initrd and b) as a virtio-block. In the latter case,
`urunc` can either levarage the container's snapshot and pass the whole
container's rootfs as the rootfs for the unikernel, or `urunc` can make use of
a user-created file to pass as a virtio-block device to the unikernel.

Therefore, the users have the following options:
1. Manually create a rootfs (either initrd or block) and package it along with the unikernel.
2. Directly copy all the files to the container's rootfs and use devmapper snapshotter, in order to allow `urunc` to pass the snapshot as a virtio-block to the unikernel.
3. Let `bunny` and ` bimanix` to create the rootfs file.

> **NOTE**: For the time being, `bunny` supports the creation of initrd files and `bimanix` does not provide any support for creating the rootfs.

## Creating an initrd file

Some unikernel frameworks and guests support an in-memory ramfs as a rootfs. In
these cases, we can use `bunny` and instruct it to create the initrd file for
us with all the specified files.
This feature is only supported using a `bunnyfile` and not a Dockerfile-like
syntax file.

Let's take a look at it using a Redis Unikraft unikernel as an example, targeting
Qemu. We will define the `bunnyfile` as:

```
#syntax=harbor.nbfc.io/nubificus/bunny:0.2.0
version: v0.1

platforms:
  framework: unikraft
  monitor: qemu
  architecture: x86

rootfs:
  from: scratch
  type: initrd
  include:
  - redis.conf:/conf/redis.conf

kernel:
  from: local
  path: redis

cmdline: "redis-server /conf/redis.conf"
```

In the above file we specify the followings:
- We want to package a Unikraft unikernel that will execute on top of Qemu over
  x86 architecture.
- We want to create a ^^initrd^^ for its rootfs, specifying `type: initrd`.  In
  particular, we want a initrd file that contains the file `redis.conf` in
  `/data/conf/redis.conf`. In that way, `bunny` creates the initrd file for us
  and set up the respective `urunc` annotations to attach this initrd file
  when we boot the unikernel.
- We want to use the `redis` binary as the unikernel to boot.
- We specify the cmdline for the unikernel as `redis-server /conf/redis.conf`

We can build the OCI image with the following command:

```
docker build -f bunnyfile -t urunc/prebuilt/redis-unikraft-qemu:test .
```

## Preparing an OCI image to be used as a rootfs for the unikernel

As previously mentioned, if we want to simply copy our files in the unikernel
OCI image and directly use the image as the rootfs for the unikernel, we can use
both `bunny` and `bimanix`. In this case, the unikernel container needs to be
created using devmapper as a snapshotter. In that way,`urunc` will use the
snapshot of the container and directly attach it to the unikernel as  a
virtio-block device.

As an example, we will use a Redis
[Rumprun](https://github.com/cloudkernels/rumprun) unikernel from
[Rumprun-packages](https://github.com/cloudkernels/rumprun-packages) targeting
[Solo5-hvt](https://github.com/Solo5/solo5).

Assumptions:
- We assume that we execute the commands in the same path where the unikernel
  resides
- We assume that all the files we want to copy inside the OCI image reside also
  in the same path as the unikernel.
- We assume that the target unikernel supports virtio-block.

> **NOTE**: The below steps can be easily adjusted to any pre-built unikernel image.

### Using `bunny`

In the case of `bunny`, we can use both supported
file syntaxes: a) `bunnyfile` and b) the Dockerfile-like syntax.

#### Using a `bunnyfile`

In order to package an existing pre-built unikernel image and any other files
with `bunny` and a `bunnyfile` we can define the `bunnyfile` as:

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
- We want to package a Rumprun unikernel that will execute on top o hvt over x86
  architecture.
- We want to create a ^^raw^^ rootfs, meaning that we will just copy the
  specified files directly to the OCI image's rootfs. In particular, we copy the
  file `redis.conf` and place in `/data/conf/redis.conf`.This is similar to
  `COPY` in Dockerfile.  Because of this type selction, `bunny` will also set up
  the respective annotations to mount the OCI images rootfs directly to the
  unikernel.
- We want to use the `redis.hvt` binary as the unikernel to boot.
- We specify the cmdline for the unikernel as `redis-server
  /data/conf/redis.conf`

We can build the OCI image with the following command:

```
docker build -f bunnyfile -t urunc/prebuilt/redis-rumprun-hvt:test .
```

#### Using a Dockerfile-like syntax

In order to package an existing pre-built unikernel image and any other files
with `bunny` using a Dockerfile-like syntax file,
we can define the `Containerfile` as:

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
- We manually specify through labels the necessary `urunc` annotations.

We can build the OCI image with the following command:

```
docker build -f Containerfile -t urunc/prebuilt/redis-rumprun-hvt:test .
```

### Using `bimanix`

In the case of `bimanix` we need the whole repository in the same directly as
the unikernel. Then, we simply need to edit the `args.nix` file. For our
pre-built Redis Rumprun unikernel we can define the files as:

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
## Packaging a pre-built rootfs, along with the unikernel

At last, there is always the option to manually create the rootfs file for the
unikernel and then package the unikernel binary and the rootfs file setting up
the respective annotations.

As an example, we will use a simple  [C HTTP Web
Server](https://github.com/unikraft/catalog/tree/main/examples/http-c) from
[Unikraft's catalog](https://github.com/unikraft/catalog).

Assumptions:
- We assume that we execute the commands in the same path where the unikernel
  resides
- We assume that all the files we want to copy inside the OCI image reside also
  in the same path as the unikernel.

> **NOTE**: The below steps can be easily adjusted to any pre-built unikernel image.

### Using `bunny`

In the case of `bunny`, we can use both supported
file syntaxes: a) `bunnyfile` and b) the Dockerfile-like syntax.

#### Using a `bunnyfile`

In order to package an existing pre-built unikernel image and any other files
with `bunny` and a `bunnyfile` we can define the `bunnyfile` as:

```
#syntax=harbor.nbfc.io/nubificus/bunny:0.2.0
version: v0.1

platforms:
  framework: unikraft
  monitor: qemu
  architecture: x86

rootfs:
  from: local
  path: rootfs.cpio

kernel:
  from: local
  path: app-elfloader

cmdline: "/chttp"
```

In the above file we specify the followings:
- We want to package a Unikraft unikernel that will execute on top of qemu over x86
  architecture.
- We want to copy a local file as a rootfs, specifically the `rootfs.cpio` file
  in the local build context.
- We want to use the `app-elfloader` binary as the unikernel to boot.
- We specify the cmdline for the unikernel as `/chttp`

We can build the OCI image with the following command:

```
docker build -f bunnyfile -t urunc/prebuilt/chttp-unikraft-qemu:test .
```

#### Using a Dockerfile-like syntax

We can do all the above using a Dockerfile-like syntax file as:

```
#syntax=harbor.nbfc.io/nubificus/bunny:0.2.0
FROM scratch

COPY app-elfloader /unikernel/app-elfloader
COPY rootfs.cpio /unikernel/rootfs.cpio

LABEL "com.urunc.unikernel.binary"=/unikernel/app-elfloader
LABEL "com.urunc.unikernel.initrd"="/unikernel/rootfs.cpio"
LABEL "com.urunc.unikernel.cmdline"="/chttp"
LABEL "com.urunc.unikernel.unikernelType"="unikraft"
LABEL "com.urunc.unikernel.hypervisor"="qemu"

```

In the above file:
- We directly copy the unikernel binary and the cpio file in
  the OCI's image rootfs.
- We manually specify through labels the necessary `urunc` annotations to use
  the previously copied cpio file as initrd.

We can build the OCI image with the following command:

```
docker build -f Containerfile -t urunc/prebuilt/chttp-unikraft-qemu:test .
```

### Using `bimanix`

In the case of `bimanix` we need the whole repository in the same directly as
the unikernel and the cpio file. Then, we simply need to edit the `args.nix` file as:

```Nix
{
  name = "urunc/prebuilt/chttp-unikraft-qemu";
  tag = "test";
  files = {
    "./app-elfloader" = "/unikernel/app-elfloader";
    "./rootfs.cpio" = "/unikernel/rootfs.cpio";
  };
  annotations = {
    unikernelType = "unikraft";
    hypervisor = "qemu";
    binary = "/unikernel/app-elfloader";
    cmdline = "/chttp";
    unikernelVersion = "";
    initrd = "/unikernel/rootfs.cpio";
    block = "";
    blkMntPoint = "";
    useDMBlock = "";
  };
}
```

In the above file:
- We directly specify the files to copy inside the OCI's image rootfs.
- We manually specify all `urunc` annotations, including the initrd one to
  specify the file to use as initrd for the unikernel.

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
