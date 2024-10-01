# Unikernel support

Unikernels are specialized, minimalistic operating systems constructed to run a
single application. By compiling only the necessary components of an OS into
the final image, unikernels offer improved performance, security, and smaller
footprints compared to traditional OS-based virtual machines.

`urunc` currently supports:

Unikraft: Unikraft is a relatively new and highly modular unikernel framework
designed to make it easier to build optimized, lightweight, and
high-performance unikernels. Unlike traditional monolithic unikernel
approaches, Unikraft allows developers to include only the components necessary
for their application, resulting in reduced footprint and improved performance.
With support for various programming languages and environments, Unikraft is
ideal for building fast, secure unikernels across a wide range of use cases.

Rumprun: A unikernel framework based on NetBSD, providing support for a variety
of POSIX-compliant applications. Rumprun is particularly useful for deploying
existing POSIX applications with minimal modification.

In the near future, we plan to add support for the following frameworks: 

`OSv`: An OS designed specifically to run as a single application on top of a
hypervisor. OSv is known for its performance optimization and supports a wide
range of programming languages, including Java, Node.js, and Python.

`MirageOS`: A library operating system that constructs unikernels for the Xen
hypervisor. MirageOS is written in OCaml, offering a functional and modular
approach to building lightweight, secure unikernels.

`IncludeOS`: A minimalistic operating system for building C++ applications
directly as unikernels. IncludeOS offers simplicity and performance, making it
ideal for microservices and other resource-efficient applications.

In what follows, we describe the process to build a simple unikernel for the
supported frameworks. Once we get a binary image, we can then move on to the
[packaging](/image-building/) process to produce the OCI image that is
compatible with `urunc`.
