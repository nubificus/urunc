# Hypervisor Support

`urunc` supports the execution of pre-packaged applications in pre-defined
sandboxes. These sandboxes can be software-based, like
[seccomp](https://en.wikipedia.org/wiki/Seccomp)-based sandboxes (`solo5-spt`)
or hardware-assisted enclaves, such as traditional or lightweight hypervisors,
eg.  [QEMU](https://qemu.org), [AWS
Firecracker](https://github.com/firecracker-microvm/firecracker) etc.).

In this document, we will go through the installation process of the various
sandboxes currently supported by `urunc`.

> Note: In general, `urunc` expects all supported hypervisors to be available somewhere in the `$PATH`.

## urunc with `solo5-hvt` / `solo5-spt`

First, let's install the apt packages required to build solo5:

```bash
sudo apt-get install libseccomp-dev pkg-config gcc -y
```

Next, we can clone and build `solo5-hvt`.

```bash
git clone -b v0.6.9 https://github.com/Solo5/solo5.git
cd solo5
./configure.sh && make -j$(nproc)
```

`urunc` expects to find the `solo5-hvt` binary located in the `$PATH` and named `solo5-hvt`. To install it:

```bash
sudo cp tenders/hvt/solo5-hvt /usr/local/bin
```

Next, we need to configure the [devmapper snapshotter](/installation/#setup-thinpool-devmapper).

Now we can run a test unikernel:

```bash
sudo nerdctl run --rm -ti --snapshotter devmapper --runtime io.containerd.urunc.v2 harbor.nbfc.io/nubificus/urunc/redis-hvt-rumprun:latest unikernel
```

> Note: as `solo5-hvt` and `solo5-spt` share command-line options and features,
> replacing `hvt` with `spt` in the above instructions installs the `solo5-spt`
> sandbox.

## urunc with qemu

`urunc` expects to find the `qemu` binary located in the `$PATH` and named
`qemu-system-{ARCH}`. You can ensure this by executing the following commands:

```bash
sudo apt-get install qemu-kvm -y
```

Now we can run a test unikernel:

```bash
sudo nerdctl run --rm -ti --runtime io.containerd.urunc.v2 harbor.nbfc.io/nubificus/urunc/nginx-qemu-unikraft-initrd:latest unikernel
```

## urunc with firecracker

`urunc` expects to find the `firecracker` binary located in the `$PATH` and
named `firecracker`. You can ensure this by executing the following commands:

```bash
ARCH="$(uname -m)"
release_url="https://github.com/firecracker-microvm/firecracker/releases"
latest=$(basename $(curl -fsSLI -o /dev/null -w %{url_effective} ${release_url}/latest))
curl -L ${release_url}/download/${latest}/firecracker-${latest}-${ARCH}.tgz \
| tar -xz

# Rename the binary to "firecracker"
sudo mv release-${latest}-$(uname -m)/firecracker-${latest}-${ARCH} /usr/local/bin/firecracker
rm -fr release-${latest}-$(uname -m)
```

Now we can run a test unikernel:

```bash
sudo nerdctl run --rm -ti --runtime io.containerd.urunc.v2 harbor.nbfc.io/nubificus/urunc/nginx-firecracker-unikraft-initrd:latest unikernel
```
