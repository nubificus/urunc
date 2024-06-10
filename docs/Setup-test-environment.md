# How to setup a testing environment

## Provision a VM

We will be using a clean Ubuntu 22.04 VM to set up our test environment. To provision a VM, `multipass` is used in this example.

Feel free to edit the generated cloud-init.yaml to include your user name and SSH key.

```bash
tee $HOME/cloud-init.yaml > /dev/null << 'EOT'
#cloud-config for dev environment
users:
  - name: gntouts
    ssh-authorized-keys:
      - ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHl6375HRkftGBTgelXCjUmzBfYU1KFOSMJgPdGiARgB ece8441@upnet.gr
    sudo: ALL=(ALL) NOPASSWD:ALL
    shell: /bin/bash

runcmd:
  - apt-get update
  - apt-get upgrade -y
EOT
multipass launch jammy -vv --cpus 4 --disk 30G --memory 4G --name ubuntuvm2 --cloud-init $HOME/cloud-init.yaml
rm $HOME/cloud-init.yaml
```

Once the VM is created, we need to connect via SSH and upgrade all packages and reboot, before proceeding.

```bash
sudo apt-get update
sudo apt-get upgrade -y
sudo reboot
```

## Install required APT packages

Let's connect back to our VM and install all `apt` packages required by `urunc`:

```bash
sudo apt-get install -y git wget bc make gcc build-essential
```

Next, let's install the packages required by `solo5`:

```bash
sudo apt-get install -y libseccomp-dev pkg-config
```

## Install Go

```bash
sudo rm -fr /usr/local/go
wget https://go.dev/dl/go1.20.8.linux-$(dpkg --print-architecture).tar.gz
sudo tar -C /usr/local -xzf go1.20.8.linux-$(dpkg --print-architecture).tar.gz
rm go1.20.8.linux-$(dpkg --print-architecture).tar.gz

sudo tee -a /etc/profile > /dev/null << 'EOT'
export PATH=$PATH:/usr/local/go/bin
EOT
```

## Install runc

```bash
sudo apt-get install -y jq
URUNC_VERSION=$(curl -s https://api.github.com/repos/opencontainers/runc/releases/latest | jq -r '.tag_name')
wget https://github.com/opencontainers/runc/releases/download/$URUNC_VERSION/runc.$(dpkg --print-architecture)
sudo install --mode 0755 runc.$(dpkg --print-architecture) /usr/local/sbin/runc
rm runc.$(dpkg --print-architecture)
```

## Install containerd

First let's install `containerd` binaries:

```bash
CONTAINERD_VERSION=$(curl -s https://api.github.com/repos/containerd/containerd/releases/latest | jq -r '.tag_name')
CONTAINERD_VERSION=${CONTAINERD_VERSION#v}
wget https://github.com/containerd/containerd/releases/download/v$CONTAINERD_VERSION/containerd-$CONTAINERD_VERSION-linux-$(dpkg --print-architecture).tar.gz
sudo tar -C /usr/local -xzf containerd-$CONTAINERD_VERSION-linux-$(dpkg --print-architecture).tar.gz
rm containerd-$CONTAINERD_VERSION-linux-$(dpkg --print-architecture).tar.gz
```

Next, we need to install `containerd` service:

```bash
wget https://raw.githubusercontent.com/containerd/containerd/v$CONTAINERD_VERSION/containerd.service
sudo install --owner root --group root --mode 0644 containerd.service /etc/systemd/system/containerd.service
rm containerd.service
```

Now, let's create a configuration file and enable `containerd` service:

```bash
sudo mkdir -p /etc/containerd
containerd config default > config.toml
sudo install --owner root --group root --mode 0644 config.toml /etc/containerd/config.toml
rm config.toml
sudo systemctl enable --now containerd.service
```

## Install CNI plugins

```bash
CNI_VERSION=$(curl -s https://api.github.com/repos/containernetworking/plugins/releases/latest | jq -r '.tag_name')
CNI_VERSION=${CNI_VERSION#v}
sudo mkdir -p /opt/cni/bin
wget https://github.com/containernetworking/plugins/releases/download/v$CNI_VERSION/cni-plugins-linux-$(dpkg --print-architecture)-v$CNI_VERSION.tgz
sudo tar -C /opt/cni/bin -xzf cni-plugins-linux-$(dpkg --print-architecture)-v$CNI_VERSION.tgz
sudo chmod 0755 /opt/cni/bin
sudo tee -a /etc/profile > /dev/null << 'EOT'
export PATH=$PATH:/opt/cni/bin
EOT
rm cni-plugins-linux-$(dpkg --print-architecture)-v$CNI_VERSION.tgz
```

## Install nerdctl

```bash
NERDCTL_VERSION=$(curl -s https://api.github.com/repos/containerd/nerdctl/releases/latest | jq -r '.tag_name')
NERDCTL_VERSION=${NERDCTL_VERSION#v}
wget https://github.com/containerd/nerdctl/releases/download/v$NERDCTL_VERSION/nerdctl-$NERDCTL_VERSION-linux-$(dpkg --print-architecture).tar.gz
sudo tar -C /usr/local/bin -xzf nerdctl-$NERDCTL_VERSION-linux-$(dpkg --print-architecture).tar.gz
rm nerdctl-$NERDCTL_VERSION-linux-$(dpkg --print-architecture).tar.gz
```

## Install crictl

```bash
CRICTL_VERSION=$(curl -s https://api.github.com/repos/kubernetes-sigs/cri-tools/releases/latest | jq -r '.tag_name')
CRICTL_VERSION=${CRICTL_VERSION#v}
wget https://github.com/kubernetes-sigs/cri-tools/releases/download/v$CRICTL_VERSION/crictl-v$CRICTL_VERSION-linux-$(dpkg --print-architecture).tar.gz
sudo tar -C /usr/local/bin -xzf crictl-v$CRICTL_VERSION-linux-$(dpkg --print-architecture).tar.gz
rm crictl-v$CRICTL_VERSION-linux-$(dpkg --print-architecture).tar.gz
sudo tee /etc/crictl.yaml > /dev/null << 'EOT'
runtime-endpoint: unix:///run/containerd/containerd.sock
image-endpoint: unix:///run/containerd/containerd.sock
timeout: 10
debug: false
EOT
```

## Configure devmapper snapshotter

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

## Configure containerd for devmapper

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
sudo systemctl restart containerd.service
```

## Build and install bima

```bash
git clone git@github.com:nubificus/bima.git
cd bima
make && sudo make install
cd ..
```

## Build and install urunc

```bash
git clone git@github.com:nubificus/urunc.git
cd urunc
make && sudo make install
cd ..
```

## Add urunc runtime to containerd

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

## Install solo5

```bash
git clone -b v0.6.9 https://github.com/Solo5/solo5.git
cd solo5
./configure.sh && make -j$(nproc)
sudo cp tenders/hvt/solo5-hvt /usr/local/bin
sudo cp tenders/hvt/solo5-spt /usr/local/bin
```

## Install qemu

```bash
sudo apt-get install qemu-kvm -y
```

## Install firecracker

```bash
FC_VERSION=$(curl -s https://api.github.com/repos/firecracker-microvm/firecracker/releases/latest | jq -r '.tag_name')
wget https://github.com/firecracker-microvm/firecracker/releases/download/$FC_VERSION/firecracker-$FC_VERSION-$(uname -m).tgz
sudo tar -C $HOME -xzf firecracker-$FC_VERSION-$(uname -m).tgz
sudo mv $HOME/release-$FC_VERSION-$(uname -m)/firecracker-$FC_VERSION-$(uname -m) /usr/local/bin/firecracker
sudo mv $HOME/release-$FC_VERSION-$(uname -m)/jailer-$FC_VERSION-$(uname -m) /usr/local/bin/jailer
sudo rm -fr release-$FC_VERSION-$(uname -m)
rm firecracker-$FC_VERSION-$(uname -m).tgz
```

## Run the tests

```bash
cd urunc
make test
```
