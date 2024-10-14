This document acts as a quickstart guide to showcase `urunc` features. Please
refer to the [installation guide](/installation) for more detailed installation
steps, or the [architecture](/design/architecture) document for more details on the
architecture.

We assume a vanilla ubuntu 22.04 environment, although `urunc` is able to be
deployed on a number of distros.

### Install a high-level container runtime

First step is to install a high-level container runtime, such as containerd and
setup basic functionality (`runc`, the `CNI` plugins and a snapshotter). Also,
a nifty tool like `nerdctl` is useful to get a realistic experience.

If these tools are already installed, skip to the next step.

#### Install runc

```bash
RUNC_VERSION=$(curl -L -s -o /dev/null -w '%{url_effective}' "https://github.com/opencontainers/runc/releases/latest" | grep -oP "v\d+\.\d+\.\d+" | sed 's/v//')
wget -q https://github.com/opencontainers/runc/releases/download/v$RUNC_VERSION/runc.$(dpkg --print-architecture)
sudo install -m 755 runc.$(dpkg --print-architecture) /usr/local/sbin/runc
rm -f ./runc.$(dpkg --print-architecture)
```

#### Install containerd

```bash
CONTAINERD_VERSION=$(curl -L -s -o /dev/null -w '%{url_effective}' "https://github.com/containerd/containerd/releases/latest" | grep -oP "v\d+\.\d+\.\d+" | sed 's/v//')
wget -q https://github.com/containerd/containerd/releases/download/v$CONTAINERD_VERSION/containerd-$CONTAINERD_VERSION-linux-$(dpkg --print-architecture).tar.gz
sudo tar Cxzvf /usr/local containerd-$CONTAINERD_VERSION-linux-$(dpkg --print-architecture).tar.gz
sudo rm -f containerd-$CONTAINERD_VERSION-linux-$(dpkg --print-architecture).tar.gz
```
#### Install containerd service

```bash
CONTAINERD_VERSION=$(curl -L -s -o /dev/null -w '%{url_effective}' "https://github.com/containerd/containerd/releases/latest" | grep -oP "v\d+\.\d+\.\d+" | sed 's/v//')
wget -q https://raw.githubusercontent.com/containerd/containerd/v$CONTAINERD_VERSION/containerd.service
sudo rm -f /lib/systemd/system/containerd.service
sudo mv containerd.service /lib/systemd/system/containerd.service
sudo systemctl daemon-reload
sudo systemctl enable --now containerd
```

#### Configure containerd

```bash
sudo mkdir -p /etc/containerd/
sudo mv /etc/containerd/config.toml /etc/containerd/config.toml.bak
sudo containerd config default | sudo tee /etc/containerd/config.toml
sudo systemctl restart containerd
```

#### Install CNI plugins

```bash
CNI_VERSION=$(curl -L -s -o /dev/null -w '%{url_effective}' "https://github.com/containernetworking/plugins/releases/latest" | grep -oP "v\d+\.\d+\.\d+" | sed 's/v//')
wget -q https://github.com/containernetworking/plugins/releases/download/v$CNI_VERSION/cni-plugins-linux-$(dpkg --print-architecture)-v$CNI_VERSION.tgz
sudo mkdir -p /opt/cni/bin
sudo tar Cxzvf /opt/cni/bin cni-plugins-linux-$(dpkg --print-architecture)-v$CNI_VERSION.tgz
sudo rm -f cni-plugins-linux-$(dpkg --print-architecture)-v$CNI_VERSION.tgz
```

#### Install nerdctl

```bash
NERDCTL_VERSION=$(curl -L -s -o /dev/null -w '%{url_effective}' "https://github.com/containerd/nerdctl/releases/latest" | grep -oP "v\d+\.\d+\.\d+" | sed 's/v//')
wget -q https://github.com/containerd/nerdctl/releases/download/v$NERDCTL_VERSION/nerdctl-$NERDCTL_VERSION-linux-$(dpkg --print-architecture).tar.gz
sudo tar Cxzvf /usr/local/bin nerdctl-$NERDCTL_VERSION-linux-$(dpkg --print-architecture).tar.gz
sudo rm -f nerdctl-$NERDCTL_VERSION-linux-$(dpkg --print-architecture).tar.gz
```

#### Setup thinpool devmapper

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

sudo /usr/local/bin/scripts/dm_create.sh
```

#### Configure containerd for devmapper

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

### Build a simple unikernel

As an example, we will be using `rumprun/solo5` running on top of `solo5-hvt`.
To facilitate the building of the unikernel we provide a container image with
the toolchain. On an `amd64` machine with docker installed, clone
https://github.com/cloudkernels/rumprun-packages and, assuming you're at $HOME,
share this folder with the container:

```
git clone https://github.com/cloudkernels/rumprun-packages -b feat_update_docker
cd rumprun-packages
docker run --rm -it -v /home/ubuntu:/home/ubuntu -w $PWD harbor.nbfc.io/nubificus/rumprun-toolchain-release:generic
```

In the container:

```
mv config.mk.dist config.mk
cd nginx
make
```

After a short while, this will produce the binary file `bin/nginx`. We now need
to "bake" it as a `solo5-hvt` unikernel:

```console
rumprun-bake solo5-hvt ./bin/nginx.hvt ./bin/nginx
```

We can now exit the container with the rumprun toolchain and continue building
the container image.

### Package the unikernel binary into a container image

To package the binary we just built into a container image, we use `bima`. For
more information on how to use this tool, refer to the relevant
[instructions](/image-building).

Go to the nginx directory and create a Containerfile:

```console
cd nginx
cat << EOF > Containerfile
FROM scratch
# the FROM instruction will not be parsed
FROM scratch

COPY nginx.hvt /unikernel/nginx.hvt
COPY data/ /

LABEL "com.urunc.unikernel.binary"=/unikernel/nginx.hvt
LABEL "com.urunc.unikernel.cmdline"="nginx -c /data/conf/nginx.conf"
LABEL "com.urunc.unikernel.unikernelType"="rumprun"
LABEL "com.urunc.unikernel.hypervisor"="hvt"
LABEL "com.urunc.unikernel.version"="0.6.6"
EOF
```

Build the container image:

```bash
wget https://s3.nbfc.io/nbfc-assets/github/bima/dist/main/x86_64/bima_x86_64
mv bima_x86_64 bima && chmod +x bima
./bima build -t nubificus/nginx-hvt-test:latest --tar .
docker load < nginx-hvt-test\:latest
```

if we inspect available images, we can see the newly created image:

```console
# docker image ls
REPOSITORY                                           TAG       IMAGE ID       CREATED         SIZE
nubificus/nginx-hvt-test                             latest    bf746aa8f8ad   N/A             40.6MB
```

One option is to push the image to dockerhub. Another option is to import it in
the snapshotter we will use for `urunc`.

```console
# ctr image import --snapshotter devmapper --base-name nubificus/nginx-hvt-test:latest nginx-hvt-test\:latest 
unpacking docker.io/nubificus/nginx-hvt-test:latest (sha256:7e410ba0559a91d1120b5d8495bc04356ef606111d54c6c9893d7d337a15c1d3)...done
```

Now, we're ready to install `urunc` and run our fist unikernel!

### Install `urunc`

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
  FILENAME=$(basename $BINARY_URL | sed s/_$ARCHITECTURE//)
  sudo mv `basename $BINARY_URL` /usr/local/bin/$FILENAME
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

### Run the unikernel

#### Install solo5

Lets install `solo5-hvt`:

```bash
apt install make gcc pkg-config libseccomp-dev
git clone -b v0.6.6 https://github.com/Solo5/solo5.git
cd solo5
./configure.sh  && make -j$(nproc)
sudo cp tenders/hvt/solo5-hvt /usr/local/bin
```

#### Run the Unikernel!

Now, let's run the unikernel image we built:

```bash
sudo nerdctl run --rm -ti --snapshotter devmapper --runtime io.containerd.urunc.v2 nubificus/nginx-hvt-test:latest
```

We can inspect the running container to check it's IP address:

```console
# nerdctl ps 
CONTAINER ID    IMAGE                                        COMMAND        CREATED          STATUS    PORTS    NAMES
10801c856b73    docker.io/nubificus/nginx-hvt-test:latest    "unikernel"    4 minutes ago    Up                 nginx-hvt-test-10801
# nerdctl inspect 10801c856b73 | grep eth0 -A 5 |grep IPAddr | awk -F\: '{print $2}'
 "10.4.0.9",
```

and we can interact with the nginx unikernel:

```bash
# curl 10.4.0.9
<html>
<body style="font-size: 14pt;">
    <img src="logo150.png"/>
    Served to you by <a href="http://nginx.org/">nginx</a>, running on a
    <a href="http://rumpkernel.org">rump kernel</a>...
</body>
</html>
```

