This document guides you through the binary installation of `urunc` and all
required components in a vanilla Ubuntu 22.04 machine.

We will be installing and setting up:

- [Go 1.20.6](https://go.dev/doc/install)
- [runc](https://github.com/opencontainers/runc)
- [containerd](https://github.com/containerd/containerd/)
- [CNI plugins](https://github.com/containernetworking/plugins)
- [nerdctl](https://github.com/containerd/nerdctl)
- [devmapper](https://docs.docker.com/storage/storagedriver/device-mapper-driver/)
- [bima](https://github.com/nubificus/bima)
- [urunc](https://github.com/nubificus/urunc)

## Install urunc

### Install required dependencies (through package management)

The following apt packages are required to complete the installation. Depending
on your specific needs, some of them may not be neccessary in your use case.

```bash
sudo apt-get install git wget bc make build-essential -y
```

### Install Go

To install Go 1.20.6:

```bash
wget -q https://go.dev/dl/go1.20.6.linux-$(dpkg --print-architecture).tar.gz
sudo mkdir go1.20.6
sudo tar -C /usr/local/go1.20.6 -xzf go1.20.6.linux-$(dpkg --print-architecture).tar.gz
sudo tee -a /etc/profile > /dev/null << 'EOT'
export PATH=$PATH:/usr/local/go1.20.6/go/bin
EOT
rm -f go1.20.6.linux-$(dpkg --print-architecture).tar.gz
```

### Install runc

`urunc` requires `runc` to handle any unsupported container images (for
example, in k8s pods the pause container is delegated to `runc` and urunc
handles only the unikernel container). You can [build runc from
source](https://github.com/opencontainers/runc/tree/main#building) or download
the latest binary following the commands:

```bash
RUNC_VERSION=$(curl -L -s -o /dev/null -w '%{url_effective}' "https://github.com/opencontainers/runc/releases/latest" | grep -oP "v\d+\.\d+\.\d+" | sed 's/v//')
wget -q https://github.com/opencontainers/runc/releases/download/v$RUNC_VERSION/runc.$(dpkg --print-architecture)
sudo install -m 755 runc.$(dpkg --print-architecture) /usr/local/sbin/runc
rm -f ./runc.$(dpkg --print-architecture)
```

### Install containerd

To install the latest release of `containerd`:

```bash
CONTAINERD_VERSION=$(curl -L -s -o /dev/null -w '%{url_effective}' "https://github.com/containerd/containerd/releases/latest" | grep -oP "v\d+\.\d+\.\d+" | sed 's/v//')
wget -q https://github.com/containerd/containerd/releases/download/v$CONTAINERD_VERSION/containerd-$CONTAINERD_VERSION-linux-$(dpkg --print-architecture).tar.gz
sudo tar Cxzvf /usr/local containerd-$CONTAINERD_VERSION-linux-$(dpkg --print-architecture).tar.gz
sudo rm -f containerd-$CONTAINERD_VERSION-linux-$(dpkg --print-architecture).tar.gz
```

### Install containerd service

```bash
CONTAINERD_VERSION=$(curl -L -s -o /dev/null -w '%{url_effective}' "https://github.com/containerd/containerd/releases/latest" | grep -oP "v\d+\.\d+\.\d+" | sed 's/v//')
wget -q https://raw.githubusercontent.com/containerd/containerd/v$CONTAINERD_VERSION/containerd.service
sudo rm -f /lib/systemd/system/containerd.service
sudo mv containerd.service /lib/systemd/system/containerd.service
sudo systemctl daemon-reload
sudo systemctl enable --now containerd
```

### Configure containerd

```bash
sudo mkdir -p /etc/containerd/
sudo mv /etc/containerd/config.toml /etc/containerd/config.toml.bak
sudo containerd config default | sudo tee /etc/containerd/config.toml
sudo systemctl restart containerd
```

For more information, you can read containerd's [Getting
Started](https://github.com/containerd/containerd/blob/main/docs/getting-started.md)
guide. 

### Install CNI plugins

To install the latest release of CNI plugins:

```bash
CNI_VERSION=$(curl -L -s -o /dev/null -w '%{url_effective}' "https://github.com/containernetworking/plugins/releases/latest" | grep -oP "v\d+\.\d+\.\d+" | sed 's/v//')
wget -q https://github.com/containernetworking/plugins/releases/download/v$CNI_VERSION/cni-plugins-linux-$(dpkg --print-architecture)-v$CNI_VERSION.tgz
sudo mkdir -p /opt/cni/bin
sudo tar Cxzvf /opt/cni/bin cni-plugins-linux-$(dpkg --print-architecture)-v$CNI_VERSION.tgz
sudo rm -f cni-plugins-linux-$(dpkg --print-architecture)-v$CNI_VERSION.tgz
```

### Install nerdctl

To install the latest release of `nerdctl`:

```bash
NERDCTL_VERSION=$(curl -L -s -o /dev/null -w '%{url_effective}' "https://github.com/containerd/nerdctl/releases/latest" | grep -oP "v\d+\.\d+\.\d+" | sed 's/v//')
wget -q https://github.com/containerd/nerdctl/releases/download/v$NERDCTL_VERSION/nerdctl-$NERDCTL_VERSION-linux-$(dpkg --print-architecture).tar.gz
sudo tar Cxzvf /usr/local/bin nerdctl-$NERDCTL_VERSION-linux-$(dpkg --print-architecture).tar.gz
sudo rm -f nerdctl-$NERDCTL_VERSION-linux-$(dpkg --print-architecture).tar.gz
```

### Setup thinpool devmapper

```bash
sudo mkdir -p /usr/local/bin/scripts
git clone https://github.com/nubificus/urunc.git

sudo cp urunc/script/dm_create.sh /usr/local/bin/scripts/dm_create.sh
sudo chmod 755 /usr/local/bin/scripts/dm_create.sh

sudo cp urunc/script/dm_reload.sh /usr/local/bin/scripts/dm_reload.sh
sudo chmod 755 /usr/local/bin/scripts/dm_reload.sh

sudo mkdir -p /usr/local/lib/systemd/system/

sudo cp urunc/script/dm_reload.service /usr/local/lib/systemd/system/dm_reload.service
sudo chmod 644 /usr/local/lib/systemd/system/dm_reload.service
sudo chown root:root /usr/local/lib/systemd/system/dm_reload.service
sudo systemctl daemon-reload
sudo systemctl enable dm_reload.service
```

### Configure containerd for devmapper

```bash
sudo sed -i '/\[plugins\."io\.containerd\.snapshotter\.v1\.devmapper"\]/,/^$/d' /etc/containerd/config.toml
sudo tee -a /etc/containerd/config.toml > /dev/null <<'EOT'

# Customizations for urunc

[plugins."io.containerd.snapshotter.v1.devmapper"]
  pool_name = "containerd-pool"
  root_path = "/var/lib/containerd/io.containerd.snapshotter.v1.devmapper"
  base_image_size = "10GB"
  discard_blocks = true
  fs_type = "ext2"
EOT
sudo systemctl restart containerd
```

## Initialize devmapper

```bash
sudo /usr/local/bin/scripts/dm_create.sh
```

### Install urunc 

```bash
declare -A ARCH_MAP
ARCH_MAP["x86_64"]="amd64"
ARCH_MAP["aarch64"]="aarch64"
SYSTEM_ARCH=$(uname -m)
ARCHITECTURE=${ARCH_MAP[$SYSTEM_ARCH]}
LATEST_RELEASE_URL="https://api.github.com/repos/nubificus/urunc/releases/latest"
RELEASE_JSON=$(curl -s $LATEST_RELEASE_URL)
LATEST_TAG=$(echo $RELEASE_JSON | jq -r '.tag_name')
ASSETS_URL=$(echo $RELEASE_JSON | jq -r '.assets[].browser_download_url')
BINARY_URLS=$(echo "$ASSETS_URL" | grep "$ARCHITECTURE")
for BINARY_URL in $BINARY_URLS; do
  echo "Downloading $BINARY_URL..."
  curl -LO "$BINARY_URL"
  chmod +x `basename $BINARY_URL`
  sudo mv `basename $BINARY_URL` /usr/local/bin
done
```

### Add urunc runtime to containerd

```bash
sudo tee -a /etc/containerd/config.toml > /dev/null <<EOT
[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.urunc]
    runtime_type = "io.containerd.urunc.v2"
    container_annotations = ["com.urunc.unikernel.*"]
    pod_annotations = ["com.urunc.unikernel.*"]
    snapshotter = "devmapper"
EOT
sudo systemctl restart containerd
```

Now, we are ready to run our unikernel images!

## Run an example unikernel

### Install solo5

First, let's install the apt packages required to build solo5:

```bash
sudo apt-get install libseccomp-dev pkg-config gcc -y
```

Next, we can clone, build and install `solo5`.

```bash
git clone -b v0.6.9 https://github.com/Solo5/solo5.git
cd solo5
./configure.sh  && make -j$(nproc)
sudo cp tenders/hvt/solo5-hvt /usr/local/bin
sudo cp tenders/spt/solo5-spt /usr/local/bin
```

### Run a redis rumprun unikernel over solo5

Now, let's run a unikernel image:

```bash
sudo nerdctl run --rm -ti --snapshotter devmapper --runtime io.containerd.urunc.v2 harbor.nbfc.io/nubificus/urunc/redis-hvt-rump:latest unikernel
```
