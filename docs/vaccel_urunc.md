# Run Unikraft vaccel with urunc

To run unikraft vaccel unikernels with urunc we need to build three different components: a) the urunc with vaccel, b) qemu with support for vaccel and c) a unikraft unikernels with vaccel

## Building urunc

Building urunc with enabled vaccel support does not differ at all with a normal build. The only difference is that we need to use the `qemu_vaccel` branch. Therefore, we install Go version 1.18+ and simply run make inside a cloned urunc repository. Then we can install urunc with simply running `make install`.

## Building qemu with vaccel

We maintain a downstream [repository](https://github.com/cloudkernels/qemu-vaccel.git) with the virtio-vaccel backend implementation. Therefore, we need to clone the repo and switch to `unikraft_vaccelrt` branch.

```
git clone https://github.com/cloudkernels/qemu-vaccel.git -b unikraft_vaccelrt
cd qemu-vaccel
git submodule update --init
./configure --extra-cflags="-I /.local/include -Wno-strict-prototypes" --extra-ldflags="-L/.local/lib" --target-list=x86_64-softmmu --enable-virtfs
make -j$(nproc)
make install
```
It is important to note that we already have already installed vaccel 
We can do that with:
```
wget -q https://s3.nbfc.io/nbfc-assets/github/vaccelrt/master/x86_64/Release-deb/vaccel-0.5.0-Linux.deb && dpkg -i vaccel-0.5.0-Linux.deb
```

## Building a unikraft unikernel with support for vaccel

For building a unikraft vaccel unikernel we can follow the instructions [here](https://github.com/cloudkernels/unikraft_vaccel_examples). The main difference is that we need to change the unikraft config file to the following one, in order to use initrd instead of shared-fs, since urunc does not support shared-fs yet.

```
#
# Automatically generated file; DO NOT EDIT.
# Unikraft/0.10.0~bc0a2657 Configuration
#
CONFIG_UK_FULLVERSION="0.10.0~bc0a2657"
CONFIG_UK_CODENAME="Phoebe"
CONFIG_UK_ARCH="x86_64"
CONFIG_UK_BASE="/store/cmainas_dev/uvaccel/unikraft"
CONFIG_UK_APP="/store/cmainas_dev/uvaccel/apps/unikraft_vaccel_examples"
CONFIG_UK_DEFNAME="unikraft_vaccel_examples"

#
# Architecture Selection
#
CONFIG_ARCH_X86_64=y
# CONFIG_ARCH_ARM_64 is not set
# CONFIG_ARCH_ARM_32 is not set
# CONFIG_MARCH_X86_64_NATIVE is not set
CONFIG_MARCH_X86_64_GENERIC=y
# CONFIG_MARCH_X86_64_NOCONA is not set
# CONFIG_MARCH_X86_64_CORE2 is not set
# CONFIG_MARCH_X86_64_COREI7 is not set
# CONFIG_MARCH_X86_64_COREI7AVX is not set
# CONFIG_MARCH_X86_64_COREI7AVXI is not set
# CONFIG_MARCH_X86_64_ATOM is not set
# CONFIG_MARCH_X86_64_K8 is not set
# CONFIG_MARCH_X86_64_K8SSE3 is not set
# CONFIG_MARCH_X86_64_AMDFAM10 is not set
# CONFIG_MARCH_X86_64_BTVER1 is not set
# CONFIG_MARCH_X86_64_BDVER1 is not set
# CONFIG_MARCH_X86_64_BDVER2 is not set
# CONFIG_MARCH_X86_64_BDVER3 is not set
# CONFIG_MARCH_X86_64_BTVER2 is not set
CONFIG_STACK_SIZE_PAGE_ORDER=4
# end of Architecture Selection

#
# Platform Configuration
#
CONFIG_PLAT_KVM=y

#
# Console Options
#
CONFIG_KVM_KERNEL_SERIAL_CONSOLE=y
CONFIG_KVM_KERNEL_VGA_CONSOLE=y
CONFIG_KVM_DEBUG_SERIAL_CONSOLE=y
CONFIG_KVM_DEBUG_VGA_CONSOLE=y

#
# Serial console configuration
#
CONFIG_KVM_SERIAL_BAUD_115200=y
# CONFIG_KVM_SERIAL_BAUD_57600 is not set
# CONFIG_KVM_SERIAL_BAUD_38400 is not set
# CONFIG_KVM_SERIAL_BAUD_19200 is not set
# end of Serial console configuration
# end of Console Options

CONFIG_KVM_MAX_IRQ_HANDLER_ENTRIES=8
CONFIG_KVM_PCI=y
CONFIG_VIRTIO_BUS=y

#
# Virtio
#
CONFIG_VIRTIO_PCI=y
CONFIG_VIRTIO_ACCEL=y
# end of Virtio

CONFIG_UKPLAT_ALLOW_GIC=y
# CONFIG_PLAT_LINUXU is not set
# CONFIG_PLAT_XEN is not set

#
# Platform Interface Options
#
# CONFIG_UKPLAT_MEMRNAME is not set
CONFIG_UKPLAT_LCPU_MAXCOUNT=1
# CONFIG_PAGING is not set
# end of Platform Interface Options

CONFIG_HZ=100
# end of Platform Configuration

#
# Library Configuration
#
CONFIG_LIBDEVFS=y
CONFIG_LIBDEVFS_AUTOMOUNT=y
# CONFIG_LIBDEVFS_DEV_NULL is not set
# CONFIG_LIBDEVFS_DEV_ZERO is not set
CONFIG_LIBDEVFS_DEV_STDOUT=y
# CONFIG_LIBFDT is not set
# CONFIG_LIBISRLIB is not set
CONFIG_LIBVACCELRT=y
CONFIG_LIBNOLIBC=y
CONFIG_LIBNOLIBC_UKDEBUG_ASSERT=y
# CONFIG_LIBPOSIX_EVENT is not set
# CONFIG_LIBPOSIX_FUTEX is not set
# CONFIG_LIBPOSIX_LIBDL is not set
# CONFIG_LIBPOSIX_PROCESS is not set
# CONFIG_LIBPOSIX_SOCKET is not set
# CONFIG_LIBPOSIX_SYSINFO is not set
# CONFIG_LIBPOSIX_USER is not set
CONFIG_LIBRAMFS=y
# CONFIG_LIBSYSCALL_SHIM is not set
# CONFIG_LIBUBSAN is not set
# CONFIG_LIBUK9P is not set
CONFIG_LIBUKALLOC=y
# CONFIG_LIBUKALLOC_IFMALLOC is not set
# CONFIG_LIBUKALLOC_IFSTATS is not set
CONFIG_LIBUKALLOCBBUDDY=y
# CONFIG_LIBUKALLOCPOOL is not set
# CONFIG_LIBUKALLOCREGION is not set
CONFIG_LIBUKARGPARSE=y
# CONFIG_LIBUKARGPARSE_TEST is not set
# CONFIG_LIBUKBLKDEV is not set
CONFIG_LIBUKBOOT=y
# CONFIG_LIBUKBOOT_BANNER_NONE is not set
# CONFIG_LIBUKBOOT_BANNER_MINIMAL is not set
# CONFIG_LIBUKBOOT_BANNER_CLASSIC is not set
CONFIG_LIBUKBOOT_BANNER_POWEREDBY=y
# CONFIG_LIBUKBOOT_BANNER_POWEREDBY_ANSI is not set
# CONFIG_LIBUKBOOT_BANNER_POWEREDBY_ANSI2 is not set
# CONFIG_LIBUKBOOT_BANNER_POWEREDBY_EA is not set
# CONFIG_LIBUKBOOT_BANNER_POWEREDBY_EAANSI is not set
# CONFIG_LIBUKBOOT_BANNER_POWEREDBY_EAANSI2 is not set
# CONFIG_LIBUKBOOT_BANNER_POWEREDBY_U8 is not set
# CONFIG_LIBUKBOOT_BANNER_POWEREDBY_U8ANSI is not set
# CONFIG_LIBUKBOOT_BANNER_POWEREDBY_U8ANSI2 is not set
CONFIG_LIBUKBOOT_MAXNBARGS=60
CONFIG_LIBUKBOOT_INITBBUDDY=y
# CONFIG_LIBUKBOOT_INITREGION is not set
# CONFIG_LIBUKBOOT_NOALLOC is not set
CONFIG_LIBUKBUS=y
CONFIG_LIBUKCPIO=y
CONFIG_LIBUKDEBUG=y
CONFIG_LIBUKDEBUG_PRINTK=y
CONFIG_LIBUKDEBUG_PRINTK_INFO=y
# CONFIG_LIBUKDEBUG_PRINTK_WARN is not set
# CONFIG_LIBUKDEBUG_PRINTK_ERR is not set
# CONFIG_LIBUKDEBUG_PRINTK_CRIT is not set
# CONFIG_LIBUKDEBUG_PRINTD is not set
# CONFIG_LIBUKDEBUG_NOREDIR is not set
CONFIG_LIBUKDEBUG_REDIR_PRINTD=y
# CONFIG_LIBUKDEBUG_REDIR_PRINTK is not set
CONFIG_LIBUKDEBUG_PRINT_TIME=y
# CONFIG_LIBUKDEBUG_PRINT_CALLER is not set
CONFIG_LIBUKDEBUG_PRINT_SRCNAME=y
# CONFIG_LIBUKDEBUG_ANSI_COLOR is not set
CONFIG_LIBUKDEBUG_ENABLE_ASSERT=y
# CONFIG_LIBUKDEBUG_TRACEPOINTS is not set
# CONFIG_LIBUKFALLOC is not set
# CONFIG_LIBUKFALLOCBUDDY is not set
CONFIG_LIBUKLIBPARAM=y
CONFIG_LIBUKLOCK=y
CONFIG_LIBUKLOCK_SEMAPHORE=y
CONFIG_LIBUKLOCK_MUTEX=y
# CONFIG_LIBUKLOCK_MUTEX_METRICS is not set
# CONFIG_LIBUKMMAP is not set
# CONFIG_LIBUKMPI is not set
# CONFIG_LIBUKNETDEV is not set
# CONFIG_LIBUKRING is not set
# CONFIG_LIBUKRUST is not set
CONFIG_LIBUKSCHED=y
CONFIG_LIBUKSCHEDCOOP=y
CONFIG_LIBUKSGLIST=y
# CONFIG_LIBUKSIGNAL is not set
# CONFIG_LIBUKSP is not set
# CONFIG_LIBUKSTORE is not set
# CONFIG_LIBUKSWRAND is not set
# CONFIG_LIBUKTEST is not set
CONFIG_LIBUKTIME=y
CONFIG_LIBUKTIMECONV=y
CONFIG_LIBVFSCORE=y

#
# vfscore: Configuration
#
CONFIG_LIBVFSCORE_PIPE_SIZE_ORDER=16
CONFIG_LIBVFSCORE_AUTOMOUNT_ROOTFS=y
# CONFIG_LIBVFSCORE_ROOTFS_RAMFS is not set
# CONFIG_LIBVFSCORE_ROOTFS_9PFS is not set
CONFIG_LIBVFSCORE_ROOTFS_INITRD=y
# CONFIG_LIBVFSCORE_ROOTFS_CUSTOM is not set
CONFIG_LIBVFSCORE_ROOTFS="initrd"
# end of vfscore: Configuration

CONFIG_HAVE_BOOTENTRY=y
CONFIG_HAVE_TIME=y
CONFIG_HAVE_SCHED=y
# end of Library Configuration

#
# Build Options
#
# CONFIG_OPTIMIZE_NONE is not set
CONFIG_OPTIMIZE_PERF=y
# CONFIG_OPTIMIZE_SIZE is not set

#
# Hint: Specify a CPU type to get most benefits from performance optimization
#
CONFIG_OPTIMIZE_NOOMITFP=y
# CONFIG_OPTIMIZE_DEADELIM is not set
# CONFIG_OPTIMIZE_LTO is not set
# CONFIG_DEBUG_SYMBOLS_LVL0 is not set
# CONFIG_DEBUG_SYMBOLS_LVL1 is not set
# CONFIG_DEBUG_SYMBOLS_LVL2 is not set
CONFIG_DEBUG_SYMBOLS_LVL3=y
# CONFIG_OPTIMIZE_WARNISERROR is not set
# CONFIG_OPTIMIZE_SYMFILE is not set
CONFIG_OPTIMIZE_COMPRESS=y
# CONFIG_RECORD_BUILDTIME is not set
CONFIG_CROSS_COMPILE=""
CONFIG_LLVM_TARGET_ARCH=""
# end of Build Options

#
# Application Options
#
CONFIG_APPVACCELTEST_DEPENDENCIES=y
# CONFIG_APPVACCELTEST_NOOP is not set
# CONFIG_APPVACCELTEST_SGEMM is not set
# CONFIG_APPVACCELTEST_SGEMM_GOP is not set
CONFIG_APPVACCELTEST_IMG_CLASS=y
# CONFIG_APPVACCELTEST_IMG_CLASS_GOP is not set
# CONFIG_APPVACCELTEST_IMG_DTCT is not set
# CONFIG_APPVACCELTEST_IMG_DTCT_GOP is not set
# CONFIG_APPVACCELTEST_IMG_SEGM is not set
# CONFIG_APPVACCELTEST_IMG_SEGM_GOP is not set
# CONFIG_APPVACCELTEST_IMG_POSE is not set
# CONFIG_APPVACCELTEST_IMG_POSE_GOP is not set
# CONFIG_APPVACCELTEST_IMG_DPTH is not set
# CONFIG_APPVACCELTEST_IMG_DPTH_GOP is not set
# CONFIG_APPVACCELTEST_EXEC is not set
# CONFIG_APPVACCELTEST_EXEC_GOP is not set
# CONFIG_APPVACCELTEST_MINMAX is not set
# CONFIG_APPVACCELTEST_MINMAX_GOP is not set
# CONFIG_APPVACCELTEST_LENET is not set
# CONFIG_APPVACCELTEST_BSCHOLES is not set
# end of Application Options

CONFIG_UK_NAME="unikraft_vaccel_examples"
``` 

We can create the initrd for the unikernel by running inside a data directory which contains our data:
```
find -depth -print | tac | bsdcpio -o --format newc > ../data.cpio
```

At last, we need to create a unikernel OCI image using bima. We can do that with the following command:
```
sudo bima build -t qemu/unikraft-vaccel-classify:latest -f Containerfile .
```

A sample Containerfile file is the following one:
```
FROM scratch
COPY build/unikraft_vaccel_examples_kvm-x86_64 /unikernel/classify
COPY data.cpio /unikernel/initrd

LABEL com.urunc.unikernel.binary=/unikernel/classify
LABEL "com.urunc.unikernel.initrd"=/unikernel/initrd
LABEL "com.urunc.unikernel.cmdline"='classify German-Shepherd-dog-Alsatian.jpg 1'
LABEL "com.urunc.unikernel.unikernelType"="unikraft"
LABEL "com.urunc.unikernel.hypervisor"="qemu"
```
