# Non-root execution of monitor

To enhance security, `urunc` supports running the monitor process (VMM or
seccomp monitor) as a non-root user. In this tutorial, we will walk through the
necessary steps to set up the environment and successfully execute the monitor
as a non-root user.

## Requirements

The vast majority of supported monitors use KVM and therefore require access to
`/dev/kvm`. This includes monitors like
[Solo5-hvt](https://github.com/Solo5/solo5), [Qemu](https://www.qemu.org), and
[Firecracker](https://github.com/firecracker-microvm/firecracker). In contrast,
[Solo5-spt](https://github.com/Solo5/solo5) does not require access to
`/dev/kvm`. As such, when spawning the monitor process, we must ensure that it
has the necessary permissions to access `/dev/kvm`.

Usually `/dev/kvm` has the following filesystem permissions:

```
$ ls -l /dev/kvm
crw-rw---- 1 root kvm 10, 232 Apr  3 08:10 /dev/kvm
```

In case the above permissions are different in your system, we strongly
recommend to perform the following steps:

```
$ sudo groupadd kvm -r
$ sudo chown root:kvm /dev/kvm
$ sudo chmod 660 /dev/kvm
```

An important information we need to obtain is the group ID of `kvm` group:

```
$ getent group kvm
kvm:x:108:ubuntu
```

## Running the monitor as non-root user

By default `urunc` will execute the monitor setting up the `uid`, `gid` and
`additionalGids` from the [container's OCI
configuration](https://github.com/opencontainers/runtime-spec/blob/main/config.md#posix-platform-user).
As a result, we simply need to instruct `urunc` to run a container as the desired
user. However, since most monitors require access to `/dev/kvm`, we must ensure
that the container is a member of the `kvm` group to grant the necessary
permissions.

### Docker and Nerdctl

In the case of docker and nerdctl, we can set the user and the groups of the
container with the `--user <uid>:<gid>` option and the additional groups using
`--group-add <gid>` for each additional group. Therefore, to run a KVM-enabled
monitor with `urunc` as `nobody`, we use the following command:

```
$ sudo nerdctl run  --user 65534:65534 --group-add 108 --runtime "io.containerd.urunc.v2" --rm -it harbor.nbfc.io/nubificus/urunc/nginx-firecracker-unikraft-initrd:latest
```

Pay attention to the `--group-add 108` option which instructs `urunc` to add
the group with id `108` as an additional group for the container. The
`108` id is the group id of `kvm` that we found previously.

On the other hand, if we are using a monitor that does not require access to
`/dev/kvm`, such as [Solo5-spt](https://github.com/Solo5/solo5), then we can
omit the `--group-add` command.  As a result, the command will transform to:

```
$ sudo nerdctl run  --user 65534:65534 --runtime "io.containerd.urunc.v2" --rm -it harbor.nbfc.io/nubificus/urunc/net-spt-mirage:latest
```

> Note The commands are the same for docker.


### In a k8s deployment

Similarly, in the case of Kubernetes, we can specify the monitor's process user
and groups by defining the container's user and groups. We can do that in
the `securityContext` field of the deployment yaml:

```
securityContext:
  runAsUser: 65534
  runAsGroup: 65534
  supplementalGroups: [108]
```

As previously mentioned, if we want to run a monitor that requires access to
`/dev/kvm`, we need to add the `kvm`'s groupid in the `supplementalGroups`.
Otherwise, we do not have to specify any supplementary group.

For more information regarding the Security Context of a Pod / Container take a
look at [Kubernetes's
documentation](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/).


## Caveats

For monitors that require access to `/dev/kvm`, we need to ensure that the
container (and, by extension, the monitor) is a member of the `kvm` group. This
is unavoidable. Additionally, it is essential that the `kvm`
group has the same ID across all nodes to make sure that all containers,
regardless of the node, can access `/dev/kvm`.

Another caveat of this setup is access to the device mapper snapshot. In some
cases, `urunc` uses the device mapper snapshot as a block device for the
unikernel. However, the snapshot is typically created by root and belongs to the
`disk` group. As a result, to use this feature of `urunc`, we will also need
to add the container to the `disk` group, similar to how we handle the `kvm`
group. Fortunately, there are options we can explore to address this issue, and
we are actively working towards a solution.
