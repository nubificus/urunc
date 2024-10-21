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

### Using Docker

Docker is probably the easiest way to get started with `urunc` locally.

Install Docker:

```bash
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
rm get-docker.sh
sudo groupadd docker
sudo usermod -aG docker $USER
```

Install `urunc`:

```bash
sudo apt-get install -y git
git clone https://github.com/nubificus/urunc.git
docker run --rm -ti -v $PWD/urunc:/urunc -w /urunc golang:latest bash -c "git config --global --add safe.directory /urunc && make"
sudo install -D -m0755 $PWD/urunc/dist/urunc_static_$(dpkg --print-architecture) /usr/local/bin/urunc
sudo install -D -m0755 $PWD/urunc/dist/containerd-shim-urunc-v2_$(dpkg --print-architecture) /usr/local/bin/containerd-shim-urunc-v2
```

Install QEMU:

```bash
sudo apt install -y qemu-kvm
```

Now we are ready to run a Unikernel using Docker with `urunc`:

```bash
docker run --rm -d --runtime io.containerd.urunc.v2 harbor.nbfc.io/nubificus/urunc/nginx-qemu-unikraft:latest unikernel
```

We can see the QEMU process:

```bash
root@dck02:~$ ps -ef | grep qemu
root       11302   11287  7 19:17 ?        00:00:02 /usr/bin/qemu-system-x86_64 -m 256M -cpu host -enable-kvm -nographic -vga none --sandbox on,obsolete=deny,elevateprivileges=deny,spawn=deny,resourcecontrol=deny -kernel /var/lib/docker/overlay2/4e1943bb06c1a4d4bd72f990628c3ab5696859339288d5a87315179a29a04e98/merged/unikernel/app-nginx_kvm-x86_64 -net nic,model=virtio -net tap,script=no,ifname=tap0_urunc -initrd /var/lib/docker/overlay2/4e1943bb06c1a4d4bd72f990628c3ab5696859339288d5a87315179a29a04e98/merged/unikernel/initrd -append nginx netdev.ipv4_addr=172.17.0.2 netdev.ipv4_gw_addr=172.17.0.1 netdev.ipv4_subnet_mask=255.255.0.0 vfs.rootfs=initrd --  -c /nginx/conf/nginx.conf
```

We are also able to extract the IP and ping the nginx unikernel:

```bash
root@dck02:~$ IP_ADDR=$(ps -ef | grep qemu | grep 'ipv4_addr' | awk -F"netdev.ipv4_addr=" '{print $2}' | awk '{print $1}')
root@dck02:~$ curl $IP_ADDR 
<!DOCTYPE html>
<html>
<head>
  <title>Hello, world!</title>
</head>
<body>
  <h1>Hello, world!</h1>
  <p>Powered by <a href="http://unikraft.org">Unikraft</a>.</p>
</body>
</html>
```

### Using containerd & nerdctl

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
