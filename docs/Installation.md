# urunc installation guide

This document guides you through the installation of `urunc` and all required components in a fresh Ubuntu 22.04 machine. An example unikernel spawn is also included.

We will be installing and setting up:

- [Go 1.20.6](https://go.dev/doc/install)
- [runc](https://github.com/opencontainers/runc)
- [containerd](https://github.com/containerd/containerd/)
- [CNI plugins](https://github.com/containernetworking/plugins)
- [nerdctl](https://github.com/containerd/nerdctl)
- [devmapper](https://docs.docker.com/storage/storagedriver/device-mapper-driver/)
- [bima](https://github.com/nubificus/bima)
- urunc

## Install urunc

### Update Ubuntu

Before we continue, let's make sure Ubuntu is up to date:

```bash
sudo apt-get update
sudo apt-get upgrade -y
```

### Install Go

To install Go 1.20.6:

```bash
wget -q https://go.dev/dl/go1.20.6.linux-$(dpkg --print-architecture).tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.20.6.linux-$(dpkg --print-architecture).tar.gz
sudo tee -a /etc/profile > /dev/null << 'EOT'
export PATH=$PATH:/usr/local/go/bin
EOT
```


### Install runc

`urunc` requires `runc` to handle any unsupported container images (for example, in k8s pods the pause container is delegated to `runc` and urunc handles only the unikernel container). You can [build runc from source](https://github.com/opencontainers/runc/tree/main#building) or download the latest binary following the commands:

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

For more information, you can read containerd's [Getting Started](https://github.com/containerd/containerd/blob/main/docs/getting-started.md) guide. 

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
git clone git@github.com:nubificus/urunc.git

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

### Install bima

```bash
git clone git@github.com:nubificus/bima.git
cd bima
make && sudo make install
cd ..
```

### Build and install urunc 

```bash
git clone git@github.com:nubificus/urunc.git
cd urunc
make && sudo make install
cd ..
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

Now, we are ready to build and run our unikernel images!

## Run an example unikernel

### Install solo5-hvt 

First, let's download and install `solo5-hvt`.

```bash
curl -L -o solo5-hvt https://s3.nubificus.co.uk/bima/$(uname -m)/solo5
sudo mv solo5-hvt /usr/local/bin
sudo chmod +x /usr/local/bin/solo5-hvt
```

### Run a redis unikernel

Now, let's run a unikernel image:

```bash
sudo nerdctl run --rm -ti --snapshotter devmapper --runtime io.containerd.urunc.v2 harbor.nbfc.io/nubificus/redis-hvt:annotated unikernel
```
