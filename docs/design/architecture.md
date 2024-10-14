title: Architecture
------

This document describes the high-level architecture of `urunc`, along with the
design choices and limitations.

## Overview

`urunc` is a container runtime designed to bridge the gap between traditional
unikernels and containerized environments. It enables seamless integration with
cloud-native architectures by leveraging familiar OCI (Open Container
Initiative) tools and methodologies. By acting as a unikernel container runtime
compatible with the Container Runtime Interface (CRI), `urunc` allows unikernels
to be managed like containers, opening up possibilities for lightweight,
secure, and high-performance application deployment.

With its support for a variety of unikernel projects and hypervisors, such as
`solo5-hvt`, `solo5-spt`, `rumprun`, `unikraft`, and `osv`, `urunc` is highly
adaptable to different use cases, ranging from serverless computing to edge
deployment.

Key Features:

- Cloud-native: Seamless integration with container orchestration tools such as
  Kubernetes, using OCI images.

- Unikernel support: `urunc` supports `solo5`, `unikraft`, `rumprun`, and `osv`
  unikernels, making it versatile across various unikernel projects.

- Hypervisor support: It supports `QEMU`, `Firecracker`, `solo5-hvt`, and
  `solo5-spt`, offering users the ability to choose the most appropriate
  execution environment for their unikernels.

## Architecture

### Execution flow

The process of starting a unikernel with `urunc` from an OCI-compatible image is
as follows:

- Image Unpacking: When a unikernel container is started, containerd or a
  similar container management system unpacks the OCI image onto a block device
  or the file system.

- Runtime Invocation: containerd invokes `urunc` with the unpacked image,
  triggering the initialization process. `urunc` reads the root filesystem of the
  image and the associated metadata (annotations) to determine how to execute the
  unikernel.

- Setup and Namespace Creation: `urunc` sets up the environment for the
  unikernel, including creating pipes for standard I/O, setting up network
  namespaces, and configuring any required resources such as block devices or
  network interfaces.

- Hypervisor Execution: Depending on the specified unikernel type and
  annotations, `urunc` selects the appropriate hypervisor (e.g., QEMU, solo5-hvt,
  or solo5-spt) and starts the unikernel binary. The unikernel runs inside its
  own isolated environment, interacting with external systems through the
  namespaces and devices configured by `urunc`.

- Lifecycle Management: Once the unikernel is running, `urunc` returns the
  process ID (PID) of the hypervisor or unikernel process to containerd, which
  manages the container's lifecycle (e.g., stopping, restarting, or deleting the
  container).

### Supported Hypervisors

`urunc` currently supports the following hypervisors:

- `Firecracker`: Amazon's Serverless hypervisor, based on `rust-vmm`.
- `QEMU`: A widely-used open-source hypervisor that provides excellent
compatibility and flexibility for unikernel deployment.
- `solo5-hvt`: A lightweight hypervisor tailored for the efficient execution of
solo5 unikernels.
- `solo5-spt`: A secure, lightweight sandboxing platform for solo5 unikernels,
providing an additional layer of security and isolation.

### Supported Unikernels

- `Solo5`: A minimalistic unikernel base designed for secure, sandboxed
execution. It works with solo5-hvt and solo5-spt as hypervisors.
- `Unikraft`: A customizable unikernel framework allowing developers to build
lightweight, application-specific OSes.
- `Rumprun`: A unikernel built on the rump kernel framework, which is capable of
running unmodified POSIX-compliant applications as unikernels.
- `OSv`: A lightweight operating system designed specifically for running cloud
applications and virtualized environments.

### Image Format and Annotations

To support unikernels in a containerized environment, `urunc` requires specific
metadata embedded in OCI container images. These images must include the
unikernel binary, configuration files, and any necessary annotations that
dictate how the unikernel should be run.

The container image is structured similarly to a traditional OCI image, with a
few notable differences:

- Base Image: The base image (FROM scratch) is typically not necessary for
  unikernels, as they do not rely on a traditional Linux distribution. However,
  in the future, `urunc` may support certain base images.

- `COPY` Instructions: Similar to Dockerfiles, the `COPY` instruction is used to
  add the unikernel binary and configuration files into the container image.
  Currently, only one `COPY` operation per instruction is supported.

- Annotations: A set of required annotations is used by `urunc` to determine how
  to run the unikernel. These annotations are added to the OCI image and
  include:

    - `com.urunc.unikernel.unikernelType`: Specifies the type of the unikernel (e.g., `rumprun`, `unikraft`, `solo5`).
    - `com.urunc.unikernel.hypervisor`: Specifies the hypervisor that will run the unikernel (e.g., `qemu`, `solo5-hvt`, `solo5-spt`).
    - `com.urunc.unikernel.binary`: The path to the unikernel binary inside the container image.
    - `com.urunc.unikernel.cmdline`: The command-line arguments for the unikernel, including any networking or block device configuration.
    - `com.urunc.unikernel.cmdline`: The version of the unikernel framework used to build the unikernel.


#### OCI Image Compatibility

Although `urunc`-formatted unikernel images are not designed to be executed by
other container runtimes, they can still be stored and distributed via generic
container registries, such as Docker Hub or Harbor. This ensures compatibility
with standard cloud-native workflows for building, shipping, and deploying
applications.

## Use-cases

### Serverless Computing

Unikernels are ideal for serverless computing due to their minimal footprint,
fast boot times, and specialization for single-purpose tasks. By using `urunc`,
developers can deploy unikernel-based serverless applications in a cloud-native
manner, taking full advantage of orchestration platforms such as Kubernetes.

### Edge Computing

In environments with limited resources, such as edge devices, unikernels offer
a lightweight alternative to full-blown virtual machines or containers. With
`urunc`, multiple unikernel-based applications can run in isolated sandboxes on
the same device, ensuring high performance, security, and efficient resource
utilization.


