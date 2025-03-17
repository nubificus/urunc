---
layout: default
title: "Pre-built unikernels"
description: "Packaging pre-built unikernels"
---

# Packaging pre-built unikernels for `urunc`

In this page we will explain the process of packaging an existing / pre-built
unikernel as an OCI image with the necessary annotations for `urunc`. As an
example, we will use a Hello world [Rumprun](https://github.com/cloudkernels/rumprun)
unikernel from
[Rumprun-packages](https://github.com/cloudkernels/rumprun-packages) targeting
[Solo5-hvt](https://github.com/Solo5/solo5).

For simply packaging pre-built unikernel images, we can use both
[bunny](https://github.com/nubificus/bunny) and
[bimanix](https://github.com/nubificus/bimanix).

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

kernel:
  from: local
  path: hello.hvt

cmdline: "hello"
```

In the above file we specify the followings:
- We want to package a Rumprun unikernel that will execute on top o hvt over x86
  architecture.
- We want to use the `hello.hvt` binary as the unikernel to boot.
- We specify the cmdline for the unikernel as `hello`

We can build the OCI image with the following command:

```
docker build -f bunnyfile -t urunc/prebuilt/hello-rumprun-hvt:test .
```

### Using a Dockerfile-like syntax

In order to package an existing pre-built unikernel image with `bunny` and a
Dockerfile-like syntax file, we can define the `Containerfile` as:

```
#syntax=harbor.nbfc.io/nubificus/bunny:0.2.0
FROM scratch

COPY hello.hvt /unikernel/hello.hvt

LABEL com.urunc.unikernel.binary=/unikernel/hello.hvt
LABEL "com.urunc.unikernel.cmdline"="hello"
LABEL "com.urunc.unikernel.unikernelType"="rumprun"
LABEL "com.urunc.unikernel.hypervisor"="hvt"
```

In the above file:
- We directly copy the unikernel binary in the OCI's image rootfs.
- We manually specify through labels the `urunc` annotations.

We can build the OCI image with the following command:

```
docker build -f Containerfile -t urunc/prebuilt/hello-rumprun-hvt:test .
```

## Using `bimanix`


In the case of `bimanix` we need to clone the whole
[repository](https://github.com/nubificus/bimanix).in the same directly as the
unikernel. Then, we simply need to edit the `args.nix` file. For our pre-built
hello Rumprun unikernel we can define the files as:

```Nix
{
  name = "urunc/prebuilt/hello-rumprun-hvt";
  tag = "test";
  files = {
    "./hello.hvt" = "/unikernel/hello.hvt";
  };
  annotations = {
    unikernelType = "rumprun";
    hypervisor = "hvt";
    binary = "/unikernel/hello.hvt";
    cmdline = "hello";
    unikernelVersion = "";
    initrd = "";
    block = "";
    blkMntPoint = "";
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
