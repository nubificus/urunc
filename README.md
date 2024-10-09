# urunc

![Build workflow](https://github.com/nubificus/urunc/actions/workflows/build.yml/badge.svg)
![Lint workflow](https://github.com/nubificus/urunc/actions/workflows/lint.yml/badge.svg)

To bridge the gap between traditional unikernels and containerized environments, enabling seamless integration with cloud-native architectures, we introduce `urunc`. Designed to fully leverage the container semantics and benefit from the OCI tools and methodology, `urunc` aims to become “runc for unikernels”, while offering compatibility with the Container Runtime Interface (CRI). By relying on underlying hypervisors, `urunc` launches unikernels provided by OCI-compatible images, allowing developers and administrators to package, deliver, deploy, and manage their software using familiar cloud-native practices.

## How urunc works

To delve into the inner workings of urunc, the process of starting a new unikernel "container" via containerd involves the following steps:

- Containerd unpacks the image onto a devmapper block device and invokes urunc.
- urunc parses the image's rootfs and annotations, initiating the required setup procedures. These include creating essential pipes for stdio and verifying the availability of the specified vmm.
- Subsequently, urunc spawns a new process within a distinct network namespace and awaits the completion of the setup phase.
- Once the setup is finished, urunc executes the vmm process, replacing the container's init process with the vmm process. The parameters for the vmm process are derived from the unikernel binary and options provided within the "unikernel" image.
- Finally, urunc returns the process ID (PID) of the vmm process to containerd, effectively enabling it to handle the container's lifecycle management.

## Installing from source

At the moment, urunc is available on x86_64 and arm64 architectures.

### Build requirements

To build and install urunc binaries, you need:

- make
- [Go](https://go.dev/doc/install) version 1.18 or greater

### Building

A urunc installation requires two binary files: `containerd-shim-urunc-v2` and `urunc`. To build and install those:

```sh
make
sudo make install
```

## Installing from prebuilt binaries

You can download the binaries from the [latest release](https://github.com/nubificus/urunc/releases/latest) and install in your PATH.

## Quick start

To run a simple `urunc` example locally, you need to address a few dependencies:

- [containerd](https://github.com/containerd/containerd) version 1.7 or higher (for installation instructions, see [here](docs/Installation.md#install-containerd), [here](docs/Installation.md#install-containerd-service) and [here](docs/Installation.md#configure-containerd))
- [devmapper snapshotter](https://docs.docker.com/storage/storagedriver/device-mapper-driver/) (for setup and configuration instructions, see [here](docs/Installation.md#setup-thinpool-devmapper), [here](docs/Installation.md#configure-containerd-for-devmapper) and [here](docs/Installation.md#initialize-devmapper))
- [runc](https://github.com/opencontainers/runc/) (installation instructions can be found [here](docs/Installation.md#install-runc))
- [nerdctl](https://github.com/containerd/nerdctl/) (installation instructions can be found [here](docs/Installation.md#install-nerdctl))
- `solo5-hvt` as the backend (installation instructions can be found [here](docs/Installation.md#install-solo5-hvt))
- `urunc` and `containerd-shim-urunc-v2` binaries
- `containerd` needs to be configured to [use devmapper](docs/Installation.md#configure-containerd-for-devmapper) and [register urunc as a runtime](docs/Installation.md#add-urunc-runtime-to-containerd)

If you already have these requirements, you can run a test container using `nerdctl`:

```bash
sudo nerdctl run --rm -ti --snapshotter devmapper --runtime io.containerd.urunc.v2 harbor.nbfc.io/nubificus/urunc/redis-hvt-rumprun:latest unikernel
```

![demo](docs/img/urunc-nerdctl-example.gif)

## Setup guide

The setup process may differ depending on your system and requirements. A full setup process for Ubuntu 22.04 can be found at [docs/Installation.md](docs/Installation.md).

Additional instructions on how to setup the various supported hypervisors can be found at [docs/Urunc-Hypervisors.md](docs/Urunc-Hypervisors.md).

## Supported hypervisors and unikernels

The following table provides an overview of the currently supported hypervisors and unikernels:

| Unikernel  | VMMs               | Arch         | Storage    |
|----------- |------------------- |------------- |----------- |
| Rumprun    | Solo5-hvt          | x86,aarch64  | Devmapper  |
| Unikraft   | QEMU, Firecracker  | x86          | Initrd     |

## Running on k8s

To use `urunc` with an existing Kubernetes cluster, you can follow the [instructions in the docs](docs/How-to-urunc-on-k8s.md).

## Linting

To locally lint the source code using Docker, run:

```bash
git clone https://github.com/nubificus/urunc.git
cd urunc
docker run --rm -v $(pwd):/app -w /app golangci/golangci-lint:v1.53.3 golangci-lint run -v --timeout=5m
# OR
sudo nerdctl run --rm -v $(pwd):/app -w /app golangci/golangci-lint:v1.53.3 golangci-lint run -v --timeout=5m
```

## License

[Apache License 2.0](LICENSE)
