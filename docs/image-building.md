# bima: Package a unikernel into an OCI artifact

bima builds an OCI-compatible Container Image from a special type of containerfile. This special containerfile supports
a minimal set of "instructions", namely FROM, COPY and LABEL. The images built by bima are intended to be run by urunc,
so there is no compatibility with other container runtimes.

- `FROM`: this is not taken into account at the current implementation, but we plan to add support for.
- `COPY`: this works as in Dockerfiles. At this moment, only a single copy operation per "instruction" (think one copy per line). These files are copied inside the image's `rootfs`, which is then passed to the unikernel as a block device and mounted under `/data` directory.
- `LABEL`: all LABEL "instructions" are added as annotations to the Container image. They are also added to a special `urunc.json` inside the container's rootfs.

Due to the tight coupling between bima and urunc, the few annotations that are required for urunc to work, are also required by bima.

The required annotations are the following:

- `com.urunc.unikernel.unikernelType`: The type of the unikernel (can be rumprun, unikraft, etc)
- `com.urunc.unikernel.hypervisor`: The desired hypervisor to run the unikernel (eg qemu, hedge, hvt)
- `com.urunc.unikernel.binary`: The unikernel binary to run
- `com.urunc.unikernel.cmdline`: The cmdline used to run the unikernel

The produced image's platform OS is always Linux, while the platform architecture is automatically extracted from the ELF headers of the file defined in `com.urunc.unikernel.binary` annotation.

A sample Containerfile should look like this:

```Dockerfile
# the FROM instruction will not be parsed
FROM scratch

COPY test-redis.hvt /unikernel/test-redis.hvt
COPY redis.conf /conf/redis.conf

LABEL com.urunc.unikernel.binary=/unikernel/test-redis.hvt
LABEL "com.urunc.unikernel.cmdline"='redis-server /data/conf/redis.conf'
LABEL "com.urunc.unikernel.unikernelType"="rumprun"
LABEL "com.urunc.unikernel.hypervisor"="qemu"
```

> Note: For labels, you can use single quotes, double quotes or no quotes at all. Defining multiple label key-value pairs in a single LABEL instruction is not supported.

## Usage

Bima mostly follows the Docker build CLI interface, as you can see:

```
NAME:
   bima build - build a container image

USAGE:
   bima build [command options] [arguments...]

OPTIONS:
   --namespace NAMESPACE, -n NAMESPACE       NAMESPACE to use when importing image to containerd (default: "default") [$CONTAINERD_NAMESPACE]
   --address ADDRESS, -a ADDRESS             ADDRESS for containerd's GRPC server to use when importing image to containerd (default: "/run/containerd/containerd.sock") [$CONTAINERD_ADDRESS]
   --snapshotter SNAPSHOTTER                 [Optional] SNAPSHOTTER name. Empty value stands for the default value. Used when importing the produced image to containerd [$CONTAINERD_SNAPSHOTTER]
   --output OUTPUT, --out OUTPUT, -o OUTPUT  [Optional] OUTPUT format for the produced images. Possible values: ["ctr", "tar"] (default: "ctr")
   --tar                                     [Optional] Shorthand version of --output=tar (default: false)
   --tag NAME, -t NAME                       Image NAME and optionally a tag (format: "name:tag")
   --file CONTAINERFILE, -f CONTAINERFILE    Name of the CONTAINERFILE  (default: "./Containerfile")
   --help, -h                                show help
```

Apart from the command options, `bima build` only accepts a single argument: the context directory for the build.

In addition to the usual options, there are a few more (non Docker) options, namely `namespace`, `address`, `snapshotter` and `output`. 

By default, bima will import the image to containerd. Namespace, address and snapshotter are passed directly to containerd, when importing the produced image.

If you want to inspect the image instead, you can set `--output=tar` or `--tar` flag to create a local tarball of the container image.

For example, to create an image based on Containerfile (or Dockerfile) found in the current directory:

```bash
bima build -t harbor.nbfc.io/nubificus/image:tag .
```
To create an image tarball:

```bash
bima build -t harbor.nbfc.io/nubificus/image:tag --tar .
# or
bima build -t harbor.nbfc.io/nubificus/image:tag --output tar .
```

To create an image from a different Containerfile:

```bash
bima build -t harbor.nbfc.io/nubificus/redis:latest -f Containerfile.redis .
```

You can verify that images were properly created:

```bash
sudo ctr image ls
```

To push the image:

```bash
sudo ctr image push harbor.nbfc.io/nubificus/redis:latest
```

## Build from source

To build from source, you can use the Makefile:

```bash
make
```

This will build a single binary under `./dist` directory for your CPU's architecture.

To install the binary:

```bash
sudo make install
```

If you want to compile for both `aarch64` and `amd64`:

```bash
make all
```



