# Installing urunc with all supported hypervisors

## urunc with solo5-hvt

First, let's install the apt packages required to build solo5:

```bash
sudo apt-get install libseccomp-dev pkg-config gcc -y
```

Next, we can clone, build and install `solo5-hvt`.

```bash
git clone -b v0.6.9 https://github.com/Solo5/solo5.git
cd solo5
./configure.sh  && make -j$(nproc)
sudo cp tenders/hvt/solo5-hvt /usr/local/bin
```

Next, we need to configure the [devmapper snapshotter](https://github.com/nubificus/urunc/blob/main/docs/Installation.md#setup-thinpool-devmapper).

Now we can run a test unikernel:

```bash
sudo nerdctl run --rm -ti --snapshotter devmapper --runtime io.containerd.urunc.v2 harbor.nbfc.io/nubificus/urunc/redis-hvt-rump:latest unikernel
```

## urunc with qemu

Let's install QEMU:

```bash
sudo apt-get install qemu-kvm libvirt-daemon-system -y
sudo systemctl restart libvirtd.service
```

Now we can run a test unikernel:

```bash
sudo nerdctl run --rm -ti --runtime io.containerd.urunc.v2 harbor.nbfc.io/nubificus/urunc/nginx-qemu-unikraft:latest unikernel
```

## urunc with firecracker

```bash
ARCH="$(uname -m)"
release_url="https://github.com/firecracker-microvm/firecracker/releases"
latest=$(basename $(curl -fsSLI -o /dev/null -w  %{url_effective} ${release_url}/latest))
curl -L ${release_url}/download/${latest}/firecracker-${latest}-${ARCH}.tgz \
| tar -xz

# Rename the binary to "firecracker"
sudo mv release-${latest}-$(uname -m)/firecracker-${latest}-${ARCH} /usr/local/bin/firecracker
rm -fr release-${latest}-$(uname -m)
```

Now we can run a test unikernel:

```bash
sudo nerdctl run --rm -ti --runtime io.containerd.urunc.v2 harbor.nbfc.io/nubificus/urunc/nginx-fc-unik:latest unikernel
```