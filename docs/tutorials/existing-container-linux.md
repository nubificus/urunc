# Running existing containers in `urunc` with Linux

While Linux is not a unikernel framework, it is the most widely used kernel in
the servers that power the cloud. As a result, the vast majority of existing
applications and services target Linux. Furthermore, Linux has a very highly
configurable build system and as proven by Lupine, we can create tailored Linux
kernel configurations for a single application.

With that goal in mind, i.e. create single application Linux kernels, this page
will provide the required to steps to get exisitng containers and execute them
on top of `urunc` as a Linux VM.

Overall, we need to do the followings:

1. Build/Fetch a Linux kernel.
2. Build/Fetch an init (optional).
3. Prepare the final image by appending the Linux kernel (and init) and set up
   `urunc` annotations.

## Linux kernel

The main requirement for running existing containers on top of `urunc` is a
Linux kernel. From `urunc`;s side there ar eno required Linux kernel
configuration options required. However, since Linux will boot over Qemu or
Firecracker, the Linux kernel should get configured accordingly for these
monitors (e.g. virtio drivers).

To provide a template, we have uploaded a Linux
kernel configuration for the v6.15.0-rc5 of the LInux kernel. This configuration
will produce a small Linux kernel of just 13MiB. On the other hand, some
features (e.g. cgroups, some system calls) are not included and further
configuration might be required.

Alternatively, the container images
`harbor.nbfc.io/nubificus/urunc/linux-kernel-qemu:v6.15.0-rc5` and
`harbor.nbfc.io/nubificus/urunc/linux-kernel-firecracker:v6.15.0-rc5`
contain a Linux kernel
built for Qemu and Firecracker with the above configuration at `/kernel`.

## Init process

When the LInux kernel boots, ti gives the control to the init process. The init
process is a long running process which serves as the parent of all other
user-space processes on the system. Since, we target single application kernels,
our application could have the role of the init process. However, there are
cases that such a scenario would not be possible.

In case the main process of an application exits, then the Linux kernel will
panic, since the init process exited.
Furthermore, there might be an issue with the application's cli arguments.
In the Linux case, `urunc` uses the Linux boot parameters to
pass cli arguments to the application. However, the Linux kernel can not
distinguish multi-word cli arguments and will treat each word as a separate argument.
To tackle this, `urunc` follows a simple convention. All multi-word cli arguments
are wrapped in single quotes and the init or the application should
handle them correctly.

As a result, we recommend using an init process before spawning an application,
or modifying the application accordingly. For that reason we developed
[urunit](https://github.com/nubificus/urunit#) a simple init specifically for
`urunc`. It serves two purposes: 1) it handles multi-word cli arguments and 2)
acts as a reaper. We can easily fetch a static binary of `urunit` from its
releases or from `harbor.nbfc.io/nubificus/urunc/urunit:latest` at
`/urunit`.

## Preparing the image

In order to distinguish normal containers from unikernels, `urunc` makes use of
specific [annotations](../image-building#annotations). Therefore, to spawn a
container over Linux with `urunc` we need to set up these annotations and of
course include the Linux kernel in the container's image rootfs. To perform the
above, we will use [bunny](https://github.com/nubificus/bunny).

Another thing that we need to take care is the rootfs. Since we boot a Linux VM,
we need to set up its rootfs. Currently there are three options for that:

1. Using directly the rootfs of the container's image (devmapper is required).
2. Creating a block image out of a container's image rootfs.
3. Creating a initrd.

### Using directly the container's rootfs

The most effortless way boot an existing container over `urunc` using a Linux
kernel is using the container's rootfs directly. However, since `urunc` does not
support shared-fs between the guest and the host yet, the only option is to use
devmapper as a snapshotter. In that way, containerd's devmapper snapshotter will
create a block image out of the container's rootfs and `urunc` can easily attach
the block image to the VM.

To set up devmapper as a snapshotter please take alook at the [installation
guide](../installation#setup-thinpool-devmapper).

#### Preparing the container image.

In this case preparing the contianer image is as easy as appending the Linux
kernel binary in the container's image and setting the respective annotations.
We can easily do all these steps with `bunny`.

Let's use as an example the `redis:alpine` container image and use the kernel
from
`harbor.nbfc.io/nubificus/urunc/linux-kernel-qemu:v6.15.0-rc5`. The respective
`bunnyfile` will look like:

```
#syntax=harbor.nbfc.io/nubificus/bunny:latest
version: v0.1

platforms:
  framework: linux
  monitor: qemu
  architecture: x86

rootfs:
  from: redis:alpine
  type: raw

kernel:
  from: local
  path: bzImage

cmdline: "/usr/local/bin/redis-server"
```

We can build the container with:

```
$ docker build -f bunnyfile -t redis/apline/linux/qemu:latest .
```

Alternatively, if we built the kernel locally, we can modify the kernel section
in the bunnyfile as:

```
kernel:
  from: local
  path: bzImage
```

It is important to note that we will execute `redis-server` as the init.
Therefore, if we want to include `urunit`, we will need to append it to the
`redis:alpine` container image with the following Dockerfile:

```
FROM harbor.nbfc.io/nubificus/urunc/urunit:latest AS init

FROM redis:alpine

COPY --from=init /urunit /urunit
```

At last we need to modify the `cmdline` section of `bunnyfile` to execute `urunit`:

```
cmdline: "/urunit /usr/local/bin/redis-server"
```

#### Running the container

Unfortunately, `docker` needs extra configuration to use devmapper. To get over
this restriction we will use nerdctl. First, we need to transfer the container
image from docker's image store to containerd's one.

```
$ docker save redis/apline/linux/qemu:latest | nerdctl load
```

Now we are finally ready to run the container and we can do that with:

```
$ nerdctl run --rm -it --snapshotter devmapper --runtime "io.containerd.urunc.v2" redis/apline/linux/qemu:latest
```

Let's find the IP of the container:
```
$ nerdctl inspect <CONTAINER ID> | grep IPAddress
```

and we should be able to ping it:

```
$ ping -c 3 10.0.4.2
```

### Using a block image

If we are not able to set up devmapper or we have a block image that can be used
as a rootfs, we can instruct `urunc` to do so.

#### Preparing the container image.

To prepare the container image we will need to first create block image. For
that purpose, we will use `nginx:alpine` image and we can perform the following
steps:

```
$ dd if=/dev/zero of=rootfs.ext2 bs=1 count=0 seek=60M
$ mkfs.ext2 rootfs.ext2
$ mkdir tmp_mnt
$ mount rootfs.ext2 tmp_mnt
$ docker export $(docker create nginx:alpine) -o nginx_alpine.tar
$ tar -xf nginx_alpine.tar -C tmp_mnt
$ cp urunit tmp_mnt # If we want urunit as init
$ umount tmp_mnt
```

Now we have a block image `rootfs.ext2` generated from `nginx:alpine` and with
`urunit` that we can get from its release page. In order to pack everything
together, we will use a Dockerfile-like systax file, just to showcase the
definition of annotations:

```
#syntax=harbor.nbfc.io/nubificus/bunny:latest
FROM scratch

COPY bzImage /kernel
COPY rootfs.ext2 /rootfs.ext2

LABEL "com.urunc.unikernel.binary"="/kernel"
LABEL "com.urunc.unikernel.cmdline"="/urunit /usr/sbin/nginx `daemon off;error_log stderr debug;`"
LABEL "com.urunc.unikernel.unikernelType"="linux"
LABEL "com.urunc.unikernel.block"="/rootfs.ext2"
LABEL "com.urunc.unikernel.blkMntPoint"="/"
LABEL "com.urunc.unikernel.hypervisor"="qemu"
```

We can build the container with:

```
$ docker build -f Containerfile -t nginx/apline/linux/qemu:latest .
```

#### Running the container

In this case, we can directly use docker to run the container, since there is no
need for devmapper.

```
$ docker run --rm -it --runtime "io.containerd.urunc.v2" nginx/apline/linux/qemu:latest
```

Let's find the IP of the container:
```
$ docker inspect <CONTAINER ID> | grep IPAddress
            "SecondaryIPAddresses": null,
            "IPAddress": "172.17.0.2",
                    "IPAddress": "172.17.0.2",
```

and we should be able to curl it:

```
$ curl 172.17.0.2
```

### Using initrd as a rootfs

In a similar way as above, we can create an initrd instead of a block image to
use as rootfs. To showcase that, we will use `traefik/whoami` image as an
exmaple.

#### Preparing the container image.

First let's create the initrd:

```
$ mkdir tmp_rootfs
$ docker export $(docker create traefik/whoami) | tar -C tmp_irootfs/ -xvf -
$ cp urunit tmp_rootfs # If we want urunit as init
$ cd tmp_rootfs
$ find . | cpio -H newc -o > ../rootfs.initrd
```

Now we have an initrd `rootfs.initrd` generated from `traefik/whoami` and with
`urunit` that we can get from its release page. In order to pack everything
together, we can use the following `bunnyfile`:

```
#syntax=harbor.nbfc.io/nubificus/bunny:latest
version: v0.1

platforms:
  framework: linux
  monitor: qemu
  architecture: x86

rootfs:
  from: local
  type: initrd
  path: rootfs.initrd

kernel:
  from: local
  path: bzImage

cmdline: "/urunit /whoami"
```

We can build the container with:

```
$ docker build -f bunnyfile -t traefik/whoami/linux/qemu:latest .
```

#### Running the container

In this case, we can directly use docker to run the container, since there is no
need for devmapper.

```
$ docker run --rm -it --runtime "io.containerd.urunc.v2" traefik/whoami/linux/qemu:latest
```

Let's find the IP of the container:
```
$ docker inspect <CONTAINER ID> | grep IPAddress
            "SecondaryIPAddresses": null,
            "IPAddress": "172.17.0.2",
                    "IPAddress": "172.17.0.2",
```

and we should be able to curl it:

```
$ curl 172.17.0.2
```


