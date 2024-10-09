# urunc: A Lightweight Container Runtime for Sandboxed Applications

To tighten the security aspects of execution environments, enabling seamless
integration of any kind of sandbox mechanism (software, or hardware)with
cloud-native architectures, we introduce `urunc`, a lightweight container
runtime, able to spawn applications in various sandboxes.

Designed to fully leverage the container semantics and benefit from the OCI
tools and methodology, `urunc` aims to become a micro (μ)runc, while offering
compatibility with the Container Runtime Interface (CRI). 

By relying on software- and hardware-based sandboxing, urunc launches
applications provided by OCI-compatible images, allowing developers and
administrators to deploy, and manage their software using familiar cloud-native
practices.

At the moment, `urunc` supports unikernels, executed on a number of
hypervisors. Check out the [roadmap](https://github.com/nubificus/urunc) we
have for upcoming features & platforms!

## How urunc works

The process of starting a new application container with `urunc`, starts at the
higher-level runtime (`containerd`) level:

- `Containerd` unpacks the image into a supported snapshotter (eg `devmapper`)
  and invokes `urunc`.
- `urunc` parses the image's rootfs and annotations, initiating the required
  setup procedures. These include creating essential pipes for stdio and
  verifying the availability of the specified vmm.
- Subsequently, `urunc` spawns a new process within a distinct network
  namespace and awaits the completion of the setup phase.
- Once the setup is finished, `urunc` executes the sandbox process, replacing
  the container's init process with the sandbox process. The parameters for the
  sandbox process are derived from the container image annotations and options provided within
  the application image. 
- Finally, `urunc` returns the process ID (PID) of the sandbox process to
  `containerd`, effectively enabling it to handle the container's lifecycle
  management.

## Features

- [Lightweight sandboxing](hypervisor-support)
- [Unikernel support](unikernel-support)
- [Integration with OCI images](image-building)
