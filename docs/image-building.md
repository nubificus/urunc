# Packaging unikernels in OCI images for `urunc`

The [OCI (Open Container Initiative) image
format](https://github.com/opencontainers/image-spec) is a standardized
specification for packaging and distributing containerized applications across
different platforms and container runtimes. It defines a common structure for
container images, including their metadata, layers, and filesystem content.
Since `urunc` is an OCI-compatible container runtime, it expects the unikernel
to be placed inside an OCI container image.

Nevertheless, in order to differentiate between traditional container images
and unikernel OCI images, `urunc` makes use of annotations or a metadata file
(`urunc.json`) inside the container's rootfs.

To facilitate the process, we provide various tools that package a unikernel
binary, along with the application's necessary files in a container image and
set the respective annotations. In particular, we can produce an OCI image with
all `urunc`'s annotations using:
1. [bima](https://github.com/nubificus/bima) a standalone tool that communicates
   with [containerd](https://github.com/containerd/containerd),
2. [pun](https://github.com/nubificus/pun) a tool that constructs a LLB or acts
   as a frontend for
   [buildkit](https://github.com/moby/buildkit?tab=readme-ov-file#output) and
3. [bimanix](https://github.com/nubificus/bimanix) which uses [Nix
   packages](https://github.com/NixOS/nix) to build the image.

In this document, we will first explain all the annotations that `urunc`
expects, in order to handle unikernels and describe each one of the three ways
to build such OCI images for unikernels.

## Annotations

[OCI
annotations](https://github.com/opencontainers/image-spec/blob/main/annotations.md)
are key-value metadata used to describe and provide additional
context for container images and runtime configurations within the OCI
specification. Using these annotations developers can
embed non-essential information about containers, such as version details,
licensing, build information, or custom runtime parameters, without affecting
the core functionality of the container itself.
The annotations can be placed in several components of the specification.
However, in the case of `urunc` we are interested about annotations which can
reach the container runtime.

Using these annotations `urunc` receives information regarding the type of the
unikernel, the VMM or sandbox mechanism to use and more. For the time being, the
required annotations are the following:

- `com.urunc.unikernel.unikernelType`: The type of the unikernel. Currently
  supported values: a) unikraft, b) rumprun.
- `com.urunc.unikernel.hypervisor`: The VMM or sandbox monitor to run the
  unikernel Currently supported values: a) `qemu`, b) `firecracker`, c) `spt`,
  d) `hvt`.
- `com.urunc.unikernel.binary`: The path to the unikernel binary inside the
  container's rootfs
- `com.urunc.unikernel.cmdline`: The application's cmdline to pass to the
  unikernel.

Except of the above, `urunc` accepts the following optional annotations:

- `com.urunc.unikernel.initrd`: The path to the initrd of the unikernel inside
  the container's rootfs.
- `com.urunc.unikernel.block`: The path to a block image, inside container's
  rootfs, which will get attached to the unikernel.
- `com.urunc.unikernel.blkMntPoint`: The mount point of the block image to
  attach in the unikernel.
- `com.urunc.unikernel.version`: The version of the unikernel framework (e.g.
  0.17.0).

Due to the fact that [Docker](https://www.docker.com/) and some high-level
container runtimes do not pass the image annotations to the underlying container
runtime, `urunc` can also read the above information from a file inside the
container's rootfs. The file should be named `urunc.json`, it should be
placed in the root directory of the container's rootfs and it should have a JSON
format with the above information, where the values are base64 encoded.

## Tools to construct OCI images with `urunc`'s annotations

As previously mentioned we currently provide 3 different tools to package
unikernels in OCI images with `urunc`'s annotations.

### Bima

[bima](https://github.com/nubificus/bima) uses
[containerd](https://github.com/containerd/containerd) to build OCI images. In
particular, [bima](https://github.com/nubificus/bima) reads the contents of a
file with a Dockerfile-like syntax. This file can contain a set of
*instructions* that specify how to package an existing unikernel binary as an
OCI image. The currently supported *instructions* are:

- `FROM`: this is not taken into account at the current implementation, but we
  plan to add support for.
- `COPY`: this works as in Dockerfiles. At this moment, only a single copy
  operation per *instruction* (think one copy per line). These files are copied
  inside the container's image rootfs.
- `LABEL`: all LABEL *instructions* are added as annotations to the Container
  image. They are also added to a special `urunc.json` inside the container's image
  rootfs.

#### Packaging a rumprun unikernel with bima

The main benefit of [bima](https://github.com/nubificus/bima) is that it is a
standalone tool which uses
[containerd](https://github.com/containerd/containerd). Therefore, there are no
dependencies on using it. For instructions to build and install
[bima](https://github.com/nubificus/bima) please check [its
README](https://github.com/nubificus/bima?tab=readme-ov-file#build-from-source)

As a result, to package a unikernel inside an OCI image and setting `urunc`'s
annotations with [bima](https://github.com/nubificus/bima), we simply need to
construct the Containerfile with all the necessary *instructions*. For instance,
to package a Redis unikernel that uses Rumprun and runs on top of Solo5, we can
construct the Containerfile as:

```Dockerfile
# the FROM instruction will not be parsed
FROM scratch

COPY test-redis.hvt /unikernel/test-redis.hvt
COPY redis.conf /conf/redis.conf

LABEL "com.urunc.unikernel.binary"=/unikernel/test-redis.hvt
LABEL "com.urunc.unikernel.cmdline"='redis-server /data/conf/redis.conf'
LABEL "com.urunc.unikernel.unikernelType"="rumprun"
LABEL "com.urunc.unikernel.hypervisor"="hvt"
```

> Note: For labels, you can use single quotes, double quotes or no quotes at
all. Defining multiple label key-value pairs in a single LABEL instruction is
not supported.

As soon as we create the Containerfile we can build the container with:

```bash
bima build -t nubificus/image:tag .
```

Please check the [README
file](https://github.com/nubificus/bima?tab=readme-ov-file#usage) of bima for
more information.

### Pun

As an alternative to [bima](https://github.com/nubificus/bima), we built
[pun](https://github.com/nubificus/pun) a tool based on
[buildkit](https://github.com/moby/buildkit?tab=readme-ov-file#output). The main
differentiate between [bima](https://github.com/nubificus/bima) and
[pun](https://github.com/nubificus/pun) is that
[pun](https://github.com/nubificus/pun) supports using existing OCI images with
a unikernel inside. An example of such images is [Unikraft's application
catalog](https://github.com/unikraft/catalog). Therefore, with
[pun](https://github.com/nubificus/pun) a user can simply define an existing
image to use and [pun](https://github.com/nubificus/pun) will add any other
files inside the container image and of course the necessary annotations.
Both [pun](https://github.com/nubificus/pun) and
[bima](https://github.com/nubificus/bima) support the same set of *instructions*
with the difference that `FROM` is handled properly from
[pun](https://github.com/nubificus/pun).

#### Packaging a Unikraft unikernel with pun

Since [pun](https://github.com/nubificus/pun) uses
[buildkit](https://github.com/moby/buildkit?tab=readme-ov-file#output) it
supports two modes of execution. In the first mode it acts as a [buildkit
frontend](https://docs.docker.com/build/buildkit/frontend/) and in the second
mode it outputs a LLB which can be passed to `buildctl`.Therefore,
[pun](https://github.com/nubificus/pun) depends on
[buildkit](https://github.com/moby/buildkit?tab=readme-ov-file#output) which
should be installed. However, if [docker](https://www.docker.com/) is already
installed, the frontend execution mode of [pun](https://github.com/nubificus/pun)
can be used directly without building anything.

Similarly to [bima](https://github.com/nubificus/bima) the first step to build
the container is to define the Containerfile. It is important to note that if we
want to use [pun](https://github.com/nubificus/pun) as a frontend for buildkit
we need to start the Containerfile with the following line:

```Dockerfile
#syntax=harbor.nbfc.io/nubificus/urunc/pun/llb:latest
```

Therefore, if we want to package a locally built Ngnix Unikraft unikernel, we
can define the Containerfile as:

```Dockerfile
#syntax=harbor.nbfc.io/nubificus/pun:0.1.0
FROM scratch

COPY build/app-nginx_qemu-x86_64 /unikernel/kernel
COPY data.cpio /unikernel/initrd

LABEL com.urunc.unikernel.binary=/unikernel/kernel
LABEL "com.urunc.unikernel.initrd"=/unikernel/initrd
LABEL "com.urunc.unikernel.cmdline"='nginx -c /nginx/conf/nginx.conf'
LABEL "com.urunc.unikernel.unikernelType"="unikraft"
LABEL "com.urunc.unikernel.hypervisor"="qemu"
```

and we can build it with a docker command:

```bash
docker build -f Containerfile -t nubificus/urunc/nginx-unikraft-qemu:test .
```

In a similar way, if we want to package an existing Nginx Unikraft unikernel
form [unikraft's catalog](https://github.com/unikraft/catalog), we should define
the Containerfile as:

```Dockerfile
#syntax=harbor.nbfc.io/nubificus/pun:0.1.0
FROM unikraft.org/nginx:1.15

LABEL com.urunc.unikernel.binary="/unikraft/bin/kernel"
LABEL "com.urunc.unikernel.cmdline"="nginx -c /nginx/conf/nginx.conf"
LABEL "com.urunc.unikernel.unikernelType"="unikraft"
LABEL "com.urunc.unikernel.hypervisor"="qemu"
```

and we can build it with the same docker command:

```bash
docker build -f Containerfile -t nubificus/urunc/nginx-unikraft-qemu:test .
```

> Note: For labels, you can use single quotes, double quotes or no quotes at
all. Defining multiple label key-value pairs in a single LABEL instruction is
not supported.

For more information check [pun's README](https://github.com/nubificus/pun).

### Bimanix

For Nix users, we have created a set of Nix scripts that we maintain in the
[bimanix](https://github.com/nubificus/bimanix) repository to build container
images for `urunc`. In contrast to the previous tools,
[bimanix](https://github.com/nubificus/bimanix) uses a nix file to define the
files to package as a container image, along with the `urunc` annotations. In
particular, this file is the `args.nix` file, which expects the same fields:

- name: the name of the container image that Nix will build
- tag: the tag of the container image that Nix will build
- files: a list of key-value pairs with all the files to copy inside the
  container image. The key-value pairs have the following format:
  `"<path-based-on-cwd>" = "<path-inside-container>"`.
- annotations: a list will all the `urunc` annotations.

#### Packaging a unikernel with bimanix

A necessary requirement to use [bimanix](https://github.com/nubificus/bimanix)
is the presence of [Nix package manager](https://github.com/NixOS/nix). Then
using [bimanix](https://github.com/nubificus/bimanix) is as simple as completing
the `args.nix` file.

For example to package a locally built Rumprun Hello world unikernel running on
top of Solo5-hvt, we should set the `args.nix` file as:

```Nix
{
  name = "hello-rumprun";
  tag = "latest";
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

Then we can build the image by simply running the following command
inside the repository:

```bash
nix-build default.nix
```

The above command will create a container image in a tar inside Nix's store. For
easier access of the tar, Nix creates a symlink of the tar file in the CWD. The
symlink will be named as `result`. Therefore, we can load the container image with:

```bash
docker load < result
```

Please check [bimanix's README](https://github.com/nubificus/bimanix) for more information.
