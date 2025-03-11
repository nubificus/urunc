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

To facilitate the process, we provide various tools that build and package a unikernel
binary, along with the application's necessary files in a container image and
set the respective annotations. In particular, we can produce an OCI image with
all `urunc`'s annotations using:
1. [bunny](https://github.com/nubificus/bunny) a tool that builds and packages unikernels
    using buildkit's LLB and can also act as a frontend for
   [buildkit](https://github.com/moby/buildkit?tab=readme-ov-file#output).
2. [bimanix](https://github.com/nubificus/bimanix) which uses [Nix
   packages](https://github.com/NixOS/nix) to package a unikernel as an OCI image.

In this document, we will first explain all the annotations that `urunc`
expects, in order to handle unikernels and describe how to use the aformentioned
tools and package unikernels as OCI images.

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
- `com.urunc.unikernel.unikernelVersion`: The version of the unikernel framework (e.g.
  0.17.0).

Due to the fact that [Docker](https://www.docker.com/) and some high-level
container runtimes do not pass the image annotations to the underlying container
runtime, `urunc` can also read the above information from a file inside the
container's rootfs. The file should be named `urunc.json`, it should be
placed in the root directory of the container's rootfs and it should have a JSON
format with the above information, where the values are base64 encoded.

## Tools to construct OCI images with `urunc`'s annotations

As previously mentioned we currently provide 2 different tools to build and
package unikernels in OCI images with `urunc`'s annotations.

### bunny

In an effort to simplify the process of building various unikernels, we built
[bunny](https://github.com/nubificus/bunny). Except of building unikernels yunub
can also pack exsiting unikernels (either locally or from OCI images) as OCI images
for `urunc`. In its core `bunny` makes use
of [buildkit's LLB](https://github.com/moby/buildkit?tab=readme-ov-file#exploring-llb),
which allow us to create OCI images from any kind of file. Currently yunub can
process two formats of files: a) the typical Dockerfile-like syntax files and b)
`bunnyfile`, a special yaml-based file.

In the case of Dockerfile-like files, yunub is only able to package pre-built unikernel
images and it is not possible to build them. This file format is kept mostly for
compatibility with pun and bima, which are not maintained anymore. Currently, yunub
can handle the following *instructions*:

- `FROM`: Specify an existing OCI image to use as a base.
- `COPY`: this works as in Dockerfiles. At this moment, only a single copy
  operation per *instruction* (think one copy per line). These files are copied
  inside the container's image rootfs.
- `LABEL`: all LABEL *instructions* are added as annotations to the Container
  image. They are also added to a special `urunc.json` inside the container's image
  rootfs.

To further extend the functionality of yunub and provide information to build
unikernels too, we use `bunnyfile` a yaml-based specia file that yunub transforms
to LLB and can be used to build and package unikernels as OCI images. Except of
building unikernels, bunny can also be used to build or append files in the
unikernel's rootfs.

The current syntax of `bunnyfile` is the following one:

```
#syntax=harbor.nbfc.io/nubificus/bunny:latest   # [1] Set bunnyfile syntax for automatic recogn
ition from docker.
version: v0.1                                   # [2] Bunnyfile version.

platforms:                                      # [3] The target platform for building/packaging.
  framework: unikraft                           # [3a] The unikernel framework.
  version: v0.15.0                              # [3b] The version of the unikernel framework.
  monitor: qemu                                 # [3c] The hypervisor/VMM or any other kind of monitor, where the unikernel will run  on top.
  architecture: x86                             # [3d] The target architecture

rootfs:                                         # [4] (Optional) Specifies the rootfs of the unikernel.
  from: local | OCI image                       # [4a] (Optional) The source of the rootfs
  path: /path/to/file                           # [4b] (Required if from is not scratch) The path in the source, where a prebuilt rootfs file resides.
  type: initrd | raw | block                    # [4c] The type of rootfs, in case the unikernel framework supports more than one (e.g. initrd, raw, block)
  include:                                      # [4d] (Optional) A list of local files to include in the rootfs
    - src:dst

kernel:                                         # [5] Specify a prebuilt kernel to use
  from: local | OCI image                       # [5a] Specify the source of an existing prebuilt kernel.
  path: path/to/file                            # [5b] Specify the path to the kernel

cmdline: hello                                  # [6] The cmdline of the app

```

For more information reagarding the `bunnyfile` please take a look at the respective
section of [bunny's README](https://github.com/nubificus/bunny?tab=readme-ov-file#bunnyfile).

Furthermore, you can find various different examples and use cases for yunub
in the [examples directory of bunny's repository](https://github.com/nubificus/bunny/tree/main/examples).

#### Packaging a Unikraft unikernel with bunny

Since [bunny](https://github.com/nubificus/bunny) uses
[buildkit](https://github.com/moby/buildkit?tab=readme-ov-file#output) it
supports two modes of execution. In the first mode it acts as a [buildkit
frontend](https://docs.docker.com/build/buildkit/frontend/) and in the second
mode it outputs a LLB which can be passed to `buildctl`.Therefore,
[bunny](https://github.com/nubificus/bunny) depends on
[buildkit](https://github.com/moby/buildkit?tab=readme-ov-file#output) which
should be installed. However, if [docker](https://www.docker.com/) is already
installed, the frontend execution mode of [bunny](https://github.com/nubificus/bunny)
can be used directly without building anything.

It is important to note that if we
want to use [bunny](https://github.com/nubificus/bunny) as a frontend for buildkit
we need to start the Containerfile with the following line:

```Dockerfile
#syntax=harbor.nbfc.io/nubificus/bunny:<version>
```

^^**Using a dockerfile-like syntax file**^^

If we want to package a locally built Ngnix Unikraft unikernel, we
can define the a Dockerfile-like syntax file as:

```Dockerfile
#syntax=harbor.nbfc.io/nubificus/bunny:0.0.1
FROM scratch

COPY build/app-nginx_qemu-x86_64 /unikernel/kernel
COPY data.cpio /unikernel/initrd

LABEL "com.urunc.unikernel.binary"=/unikernel/kernel
LABEL "com.urunc.unikernel.initrd"=/unikernel/initrd
LABEL "com.urunc.unikernel.cmdline"='nginx -c /nginx/conf/nginx.conf'
LABEL "com.urunc.unikernel.unikernelType"="unikraft"
LABEL "com.urunc.unikernel.hypervisor"="qemu"
```

^^**Using bunnyfile**^^

If we want to package the same unikernel, using `bunnyfile`, we have to
define it as:

```
#syntax=harbor.nbfc.io/nubificus/bunny:0.0.1
version: v0.1

platforms:
  framework: unikraft
  monitor: qemu
  architecture: x86

rootfs:
  from: local
  path: data.cpio

kernel:
  from: local
  path: build/app-nginx_qemu-x86_64

cmdline: nginx -c /nginx/conf/nginx.conf
```

and we can build it with a docker command:

```bash
docker build -f bunnyfile -t nubificus/urunc/nginx-unikraft-qemu:test .
```

> **NOTE**: We cna use the above command and switch form bunnyfile to the
> Dockerfile-like file and build the same unikernel OCI image.

For more information check [bunny's README](https://github.com/nubificus/bunny).

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
