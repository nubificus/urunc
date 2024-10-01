# urunc: A Lightweight Container Runtime for Unikernels

To tighten the security aspects of execution environments, enabling seamless
integration of any kind of sandbox mechanism (software, or hardware) with
cloud-native architectures, we introduce `urunc`, a lightweight container
runtime, able to spawn applications built as unikernels.

Designed to fully leverage the container semantics and benefit from the OCI
tools and methodology, `urunc` aims to become a micro (μ)runc, while offering
compatibility with the Container Runtime Interface (CRI). 

By relying on software- and hardware-based sandboxing, urunc launches
unikernels packaged as OCI-compatible images, allowing developers and
administrators to deploy, and manage unikernel-based applications using
familiar cloud-native practices.

`urunc` supports various unikernels, executed on a number of hypervisors. Check
out the [roadmap](https://github.com/nubificus/urunc) we have for upcoming
features & platforms!

## How urunc works

The process of starting a new unikernel container with `urunc`, starts at the
higher-level runtime (`containerd`) level:

- `Containerd` unpacks the image into a supported snapshotter (eg `devmapper`)
  and invokes `urunc`.
- `urunc` parses the image's rootfs and annotations, initiating the required
  setup procedures. These include creating essential pipes for stdio and
  verifying the availability of the specified VMM.
- Subsequently, `urunc` spawns a new process within a distinct network
  namespace and awaits the completion of the setup phase.
- Once the setup is finished, `urunc` executes the sandbox process, replacing
  the container's init process with the sandbox process. The parameters for the
  unikernel are derived from the OCI image annotations and options provided within
  the unikernel itself. 
- Finally, `urunc` returns the process ID (PID) of the hypervisor process to
  `containerd`, effectively enabling it to handle the container's lifecycle
  management.

## Features

- [Hypervisor support](hypervisor-support)
- [Unikernel support](unikernel-support)
- [Integration with OCI images](image-building)
