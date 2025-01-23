# Unikernel support

Unikernels are specialized, minimalistic operating systems constructed to run a
single application. By compiling only the necessary components of an OS into
the final image, unikernels offer improved performance, security, and smaller
footprints compared to traditional OS-based virtual machines.

One of the main goals of `urunc` is to bridge the gap between unikernels and
the cloud-native ecosystem. For that reason, `urunc` aims to support all the
available unikernel frameworks and similar technologies.

For the time being, `urunc` provides support for
[Unikraft](https://unikraft.org/) and
[Rumprun](https://github.com/cloudkernels/rumprun) unikernels.

## Unikraft

[Unikraft](https://unikraft.org/) is a POSIX-friendly and highly modular
unikernel framework designed to make it easier to build optimized, lightweight,
and high-performance unikernels. Unlike traditional monolithic unikernel
approaches, [Unikraft](https://unikraft.org/) allows developers to include only
the components necessary for their application, resulting in reduced footprint
and improved performance. At the same time, [Unikraft](https://unikraft.org/)
offers Linux binary compatibility allowing easier and effortless execution
of existing applications on top of [Unikraft](https://unikraft.org/).  With
support for various programming languages and environments,
[Unikraft](https://unikraft.org/) is ideal for building unikernels across a
wide range of use cases.

### VMMs and other sandbox monitors

[Unikraft](https://unikraft.org/) can boot on top of both Xen and KVM
hypervisors. Especially in the case of KVM, [Unikraft](https://unikraft.org/)
supports [Qemu](https://www.qemu.org/) and [AWS
Firecracker](https://github.com/firecracker-microvm/firecracker). In both
cases, it gets network access through virtio-net. In the case of storage, to
the best of our knowledge [Unikraft](https://unikraft.org/) supports two
options: a) 9pFS sharing a directory between the host and the unikernel and b)
initrd and therefore an initial RamFS.

### Unikraft and `urunc`

In the case of [Unikraft](https://unikraft.org/), `urunc` supports both network
and storage I/O over both Qemu and Firecracker VMMs. However, for the time
being, `urunc` only offers support for the initrd option of
[Unikraft](https://unikraft.org/) and not for shared-fs. On the other hand, the
shared-fs option is Work-In-Progress and we will soon provide an update about
this.

[Unikraft](https://unikraft.org/) maintains a
[catalog](https://github.com/unikraft/catalog) with available applications as
unikernel images. Check out our [packaging](../image-building) page on how to
get these images and run them on top of `urunc`.

An example of [Unikraft](https://unikraft.org/) on top of Qemu with `urunc`:

```bash
$ sudo nerdctl run --rm -ti --runtime io.containerd.urunc.v2 harbor.nbfc.io/nubificus/urunc/nginx-qemu-unikraft-initrd:latest unikernel
```

Another example of [Unikraft](https://unikraft.org/) on top of Firecracker with `urunc`:

```bash
$ sudo nerdctl run --rm -ti --runtime io.containerd.urunc.v2 harbor.nbfc.io/nubificus/urunc/nginx-firecracker-unikraft-initrd:latest unikernel
```

## Mirage

[MirageOS](https://github.com/mirage/mirage) is a library operating system that
constructs unikernels for secure, high-performance network applications across
various cloud computing and mobile platforms.
[MirageOS](https://github.com/mirage/mirage) uses the OCaml language, with
libraries that provide networking, storage and concurrency support that work
under Unix during development, but become operating system drivers when being
compiled for production deployment. We can easily set up and build
[MirageOS](https://github.com/mirage/mirage) unikernels with `mirage`, which can
be installed throgu the [Opam](https://opam.ocaml.org/) source package manager.
The framework is fully event-driven, with no support for preemptive threading.

[MirageOS](https://github.com/mirage/mirage) is characterized from the extremely
fast start up times (just a few milliseconds), small binaries (usually a few
megabytes), small footprint (requires a few megabytes of memory) and safe logic,
as it is completely written in OCaml.

### VMMs and other sandbox monitors

[MirageOS](https://github.com/mirage/mirage), as one of the first unikernel
frameworks, provides support for a variety of hypervisors and platforms. In
particular, [MirageOS](https://github.com/mirage/mirage) makes use of
[Solo5](https://github.com/Solo5/solo5) and can execute as a VM over KVM/Xen and
other OSes, such as BSD OSes (FreeBSD, OpenBSD) or even Muen. Especially for
KVM, [MirageOS](https://github.com/mirage/mirage) supports
[Qemu](https://www.qemu.org/) and [Solo5-hvt](https://github.com/Solo5/solo5).
It can access the network through virtio-net in the case of Qemu and using
Solo5's I/O interface in the case of Solo5. For storage,
[MirageOS](https://github.com/mirage/mirage) supports block-based storage
through virtio-block and Solo5's I/O in Qemu and Solo5 respectively.

Furthermore, [MirageOS](https://github.com/mirage/mirage) is also
possible to execute on top of [Solo5-spt](https://github.com/Solo5/solo5) a
sandbox monitor of Solo5 project that does not use hardware-assisted
virtualization. In that context,
[MirageOS](https://github.com/mirage/mirage) can access network and block
storage through Solo5's I/O interface.

### MirageOS and `urunc`

In the case of [MirageOS](https://github.com/mirage/mirage) `urunc`
provides support for Solo5-spt, Solo5-hvt and Qemu. For all
monitors of Solo5 `urunc` allows the access of both network and block storage
through Solo5's I/O interface and for Qemu through virtio-net and virtio-block.

For the time being, the block image that the
[MirageOS](https://github.com/mirage/mirage) unikernel access during its
execution should be placed inside the container image.

For more information on packaging
[MirageOS](https://github.com/mirage/mirage) unikernels for `urunc` take
a look at our [packaging](../image-building/) page.

An example of [MirageOS](https://github.com/mirage/mirage) on top of
Solo5-hvt using a block image inside the container's rootfs with 'urunc':

```bash
$ sudo nerdctl run --rm -ti --runtime io.containerd.urunc.v2 harbor.nbfc.io/nubificus/urunc/net-mirage-hvt:latest unikernel
```

An example of [MirageOS](https://github.com/mirage/mirage) on top of
Solo5-spt with 'urunc':

```bash
$ sudo nerdctl run --rm -ti --runtime io.containerd.urunc.v2 harbor.nbfc.io/nubificus/urunc/net-mirage-spt:latest unikernel
```

## Rumprun

[Rumprun](https://github.com/cloudkernels/rumprun) is a unikernel framework
based on NetBSD, providing support for a variety of POSIX-compliant
applications. [Rumprun](https://github.com/cloudkernels/rumprun) is
particularly useful for deploying existing POSIX applications with minimal
modifications. As a consequence of its design
[Rumprun](https://github.com/cloudkernels/rumprun) can be up-to-date with the
latest changes of NetBSD. However, the current repositories are not totally
up-to-date. The repository with the most recent NetBSD version is
[here](https://github.com/cloudkernels/rumprun).

In addition, [Rumprun](https://github.com/cloudkernels/rumprun) maintains a
[repository](https://github.com/cloudkernels/rumprun-packages) with all ported
applications that can be easily used on top of
[Rumprun](https://github.com/cloudkernels/rumprun).

### VMMs and other sandbox monitors

[Rumprun](https://github.com/cloudkernels/rumprun), as one of the oldest
unikernel frameworks, provides support for both Xen and KVM hypervisors.
Especially in the case of KVM,
[Rumprun](https://github.com/cloudkernels/rumprun) supports
[Qemu](https://www.qemu.org/) and [Solo5-hvt](https://github.com/Solo5/solo5).
It can access the network through virtio-net in the case of Qemu and using
Solo5's I/O interface in the case of Solo5. As far as we concern,
[Rumprun](https://github.com/cloudkernels/rumprun) only supports block
storage through virtio-block and Solo5's I/O in Qemu and Solo5
respectively.

Furthermore, [Rumprun](https://github.com/cloudkernels/rumprun) is also
possible to execute on top of [Solo5-spt](https://github.com/Solo5/solo5) a
sandbox monitor of Solo5 project that does not use hardware-assisted
virtualization. In that context,
[Rumprun](https://github.com/cloudkernels/rumprun) can access network and block
storage through Solo5's I/O interface.

### Rumprun and `urunc`

In the case of [Rumprun](https://github.com/cloudkernels/rumprun), `urunc`
provides support for Solo5-spt and Solo5-hvt, but not yet for Qemu. For all
monitors of Solo5 `urunc` allows the access of both network and block storage
through Solo5's I/O interface. In particular, `urunc` takes advantage of
[Rumprun](https://github.com/cloudkernels/rumprun) block storage and ext2
filesystem support and allows the mounting of the containerd's snapshot
directly in the unikernel. This is only possible using devmapper as a
snapshotter in containerd. For more information on setting up devmapper, please
take a look on our [installation guide](../installation#setup-thinpool-devmapper).

Except for devmapper, `urunc` also supports the option of adding a block image
inside the container image and attaching it to
[Rumprun](https://github.com/cloudkernels/rumprun).

For more information on packaging
[Rumprun](https://github.com/cloudkernels/rumprun) unikernels for `urunc` take
a look at our [packaging](../image-building/) page.

An example of [Rumprun](https://github.com/cloudkernels/rumprun) on top of
Solo5-hvt using a block image inside the container's rootfs with 'urunc':

```bash
$ sudo nerdctl run --rm -ti --runtime io.containerd.urunc.v2 harbor.nbfc.io/nubificus/urunc/redis-hvt-rumprun-block:latest unikernel
```

An example of [Rumprun](https://github.com/cloudkernels/rumprun) on top of
Solo5-spt using devmapper with 'urunc':

```bash
$ sudo nerdctl run --rm -ti --snapshotter devmapper --runtime io.containerd.urunc.v2 harbor.nbfc.io/nubificus/urunc/redis-spt-rumprun:latest unikernel
```

## Future unikernels and frameworks:

In the near future, we plan to add support for the following frameworks:

[Mewz](https://github.com/mewz-project/mewz): A unikernel designed
specifically for running Wasm applications and compatible with WASI.

[Linux](https://github.com/mewz-project/mewz): The widely known kernel that runs
almost everywhere. In the case of `urunc` we will support minimal Linux
configurations where the init process is the application.

[OSv](https://github.com/cloudius-systems/osv): An OS designed specifically to
run as a single application on top of a hypervisor. OSv is known for its
performance optimization and supports a wide range of programming languages,
including Java, Node.js, and Python.

