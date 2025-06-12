// Copyright (c) 2023-2025, Nubificus LTD
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Parts of this file have been taken from
// https://github.com/opencontainers/runc/blob/8eb2f43047ce24f06a4cbfd9af4aaedab1062bfb/libcontainer/rootfs_linux.go
// which comes with an Apache 2.0 license. For more information check runc's
// licence.

package unikontainers

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
	"github.com/sirupsen/logrus"

	"github.com/opencontainers/runtime-spec/specs-go"
)

// pivotRootfs changes rootfs with pivot
// It should be called with CWD being the new rootfs
func pivotRootfs(newRoot string) error {
	// Set up directory of previous rootfs
	oldRoot := filepath.Join(newRoot, "/old_root")
	err := os.MkdirAll(oldRoot, 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", oldRoot, err)
	}

	err = unix.PivotRoot(".", "old_root")
	if err != nil {
		return fmt.Errorf("failed to pivot root: %w", err)
	}

	// Make sure we are in the new rootfs
	err = os.Chdir("/")
	if err != nil {
		return fmt.Errorf("failed to set CWD as /: %w", err)
	}

	// Make oldroot rslave to make sure our unmounts don't propagate to the
	// host (and thus bork the machine). We don't use rprivate because this is
	// known to cause issues due to races where we still have a reference to a
	// mount while a process in the host namespace are trying to operate on
	// something they think has no mounts (devicemapper in particular).
	err = unix.Mount("", "old_root", "", unix.MS_SLAVE|unix.MS_REC, "")
	if err != nil {
		return fmt.Errorf("failed to make old_root rslave: %w", err)
	}

	// Perform the unmount. MNT_DETACH allows us to unmount /proc/self/cwd.
	err = unix.Unmount("old_root", unix.MNT_DETACH)
	if err != nil {
		return fmt.Errorf("failed to unmount old_root: %w", err)
	}

	// We no longer need the old rootfs
	err = os.RemoveAll("old_root")
	if err != nil {
		return fmt.Errorf("failed to remobe old_root: %w", err)
	}

	return nil
}

// changeRoot changes the rootfs to rootfsDir. If pivot is true, then we will
// use pivot (requires mount namespaces), otherwise we will use chroot
func changeRoot(rootfsDir string, pivot bool) error {
	// Set CWD the rootfs of the container
	err := os.Chdir(rootfsDir)
	if err != nil {
		return err
	}

	if pivot {
		err = pivotRootfs(rootfsDir)
		if err != nil {
			return err
		}
	} else {
		err = unix.Chroot(".")
		if err != nil {
			return err
		}
	}

	// Set CWD the rootfs of the container to ensure we are in the new rootfs
	err = os.Chdir("/")
	if err != nil {
		return err
	}

	return nil
}

// prepareMonRootfs prepares the rootfs where the monitor will execute. It
// essentially sets up the devices (KVM, snapshotter block device) that are required
// for the guest execution and any other files (e.g. binaries).
func prepareMonRootfs(monRootfs string, monitorPath string, dmPath string, needsKVM bool, needsTAP bool) error {
	err := fileFromHost(monRootfs, monitorPath, "", false)
	if err != nil {
		return err
	}

	// TODO: Remove these when we switch to static binaries
	monitorName := filepath.Base(monitorPath)
	if monitorName != "firecracker" {
		err = fileFromHost(monRootfs, "/lib", "", false)
		if err != nil {
			return err
		}

		err = fileFromHost(monRootfs, "/lib64", "", false)
		if err != nil {
			return err
		}

		err = fileFromHost(monRootfs, "/usr/lib", "", false)
		if err != nil {
			return err
		}
	}

	// TODO: Remove these when we switch to static binaries
	if len(monitorName) >= 4 && monitorName[:4] == "qemu" {
		qDataPath, err := findQemuDataDir("qemu")
		fmt.Println(qDataPath)
		if err != nil {
			return err
		}

		err = fileFromHost(monRootfs, qDataPath, "/usr/share/qemu", false)
		if err != nil {
			return err
		}

		// In urunc-deploy, we do not install seabios and hence
		// we do not need it. SO if we do not find, just ignore it.
		sBiosPath, err := findQemuDataDir("seabios")
		if err == nil {
			err = fileFromHost(monRootfs, sBiosPath, "/usr/share/seabios", false)
			if err != nil {
				return err
			}
		}
	}

	err = createTmpfs(monRootfs, "/dev", unix.MS_NOSUID|unix.MS_STRICTATIME, "755")
	if err != nil {
		return err
	}

	err = createTmpfs(monRootfs, "/tmp", unix.MS_NOSUID|unix.MS_NOEXEC|unix.MS_STRICTATIME, "1777")
	if err != nil {
		return err
	}

	err = setupDev(monRootfs, "/dev/null")
	if err != nil {
		return err
	}

	err = setupDev(monRootfs, "/dev/urandom")
	if err != nil {
		return err
	}

	if needsTAP || monitorName == "firecracker" {
		err = setupDev(monRootfs, "/dev/net/tun")
		if err != nil {
			return err
		}
	}

	if dmPath != "" {
		err = setupDev(monRootfs, dmPath)
		if err != nil {
			return err
		}
	}

	if needsKVM {
		err = setupDev(monRootfs, "/dev/kvm")
		if err != nil {
			return err
		}
	}

	return nil
}

// createTmpfs creates a new tmpfs at path inside monRootfs
// In particular, it is used for the creation of /tmp and /dev.
// This is necessary to create the required devices for the monitor execution,
// such as KVM, null, urandom etc.
func createTmpfs(monRootfs string, path string, flags uint64, mode string) error {
	dstPath := filepath.Join(monRootfs, path)
	mountType := "tmpfs"
	data := "mode=" + mode + ",size=65536k"

	err := os.MkdirAll(dstPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create %s dir: %w", path, err)
	}

	err = unix.Mount(mountType, dstPath, mountType, uintptr(flags), data)
	if err != nil {
		return fmt.Errorf("failed to mount %s tmpfs: %w", path, err)
	}

	// Remove propagation
	err = unix.Mount("", dstPath, "", unix.MS_PRIVATE, "")
	if err != nil {
		return fmt.Errorf("failed to create %s tmpfs: %w", path, err)
	}

	if mode == "1777" {
		err := os.Chmod(path, 01777)
		if err != nil {
			return fmt.Errorf("failed to chmod %s: %w", path, err)
		}
	}
	return nil
}

// SetupDev set ups one new device in the container's rootfs.
// This function will get the major and minor number of
// the device from the host's rootfs and it will replicate the device
// inside the container's rootfs. It also appends rw for other users
// in the permissions of the original file.
func setupDev(monRootfs string, devPath string) error {
	// Get info of the original file
	var devStat unix.Stat_t
	err := unix.Stat(devPath, &devStat)
	if err != nil {
		return fmt.Errorf("failed to stat dev %s: %w", devPath, err)
	}

	// mask file's mode
	mode := devStat.Mode & unix.S_IFMT
	if mode != unix.S_IFCHR && mode != unix.S_IFBLK {
		return fmt.Errorf("%s is not a device node", devPath)
	}
	// Get minor,major numbers
	rdev := devStat.Rdev
	major := unix.Major(uint64(rdev))
	minor := unix.Minor(uint64(rdev))

	newDev := unix.Mkdev(major, minor)

	// Set the correct target path
	relHostPath, err := filepath.Rel("/", devPath)
	if err != nil {
		return fmt.Errorf("failed to get relative path of %s to /: %w", devPath, err)
	}
	dstPath := filepath.Join(monRootfs, relHostPath)
	// If the device is not at /dev but further down the tree, create
	// the necessary directories
	if filepath.Dir(devPath) != "/dev" {
		dstDir := filepath.Dir(dstPath)
		err = os.MkdirAll(dstDir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dstDir, err)
		}
	}

	// Create the new device node
	err = unix.Mknod(dstPath, devStat.Mode, int(newDev)) //nolint: gosec
	if err != nil {
		return fmt.Errorf("failed to make device node %s: %w", dstPath, err)
	}

	// Set up permissions, adding rw for others to ensure that any user can
	// read/write them. This is helpful for non-root monitor execution and
	// removes the burdain of getting kvm/block group id
	permBits := devStat.Mode & 0o777
	permBits |= 0o006
	err = unix.Chmod(dstPath, permBits)
	if err != nil {
		return fmt.Errorf("failed to chmod %s: %w", dstPath, err)
	}

	// Set the owner as in the original file
	err = os.Chown(dstPath, int(devStat.Uid), int(devStat.Gid))
	if err != nil {
		return fmt.Errorf("failed to chown %s: %w", dstPath, err)
	}

	return nil
}

// fileFromHost set ups a mirror of file from the host's rootfs inside the
// container's rootfs. Also, it preserves the permissions and ownership of the
// file in the host's rootfs.
// if withCopy is set then copy the file, otherwise
// bind mount it.
// In the context of monitor binaries a copy is considered safer, since
// none of the monitor processes will share memory with other processes
// of the same monitor. On the other hand, a copy is slower and consumes
// more space.
func fileFromHost(monRootfs string, hostPath string, target string, withCopy bool) error {
	uniklog.WithFields(logrus.Fields{
		"hostPath":  hostPath,
		"target":    target,
		"monRootfs": monRootfs,
		"withCopy":  withCopy,
	}).Debug("Preparing file from host")

	// Get the info of the original file
	var fileInfo unix.Stat_t
	err := unix.Stat(hostPath, &fileInfo)
	if err != nil {
		return fmt.Errorf("failed to stat file %s: %w", hostPath, err)
	}
	mode := fileInfo.Mode

	if target == "" {
		// Set the correct path
		target, err = filepath.Rel("/", hostPath)
		if err != nil {
			return fmt.Errorf("failed to get relative path of %s to /: %w", hostPath, err)
		}
	}
	dstPath := filepath.Join(monRootfs, target)

	if (mode & unix.S_IFMT) != unix.S_IFDIR {
		dstDir := filepath.Dir(dstPath)
		uniklog.WithFields(logrus.Fields{
			"hostPath":  hostPath,
			"dstDir":    dstDir,
			"dstPath": dstPath,
			"withCopy":  withCopy,
		}).Debug("in IF")
		uniklog.WithField("fileInfo", fileInfo).Debugf("File info for host file: %+v", fileInfo)

		if withCopy {
			err = copyFile(hostPath, dstPath)
			if err != nil {
				return fmt.Errorf("failed to copy file %s -> %s: %w", hostPath, dstPath, err)
			}
		} else {
			err = bindMountFile(hostPath, dstDir, dstPath, fileInfo.Mode, false)
			if err != nil {
				return fmt.Errorf("failed to bind mount file in if %s: %w", hostPath, err)
			}
		}
	} else {
		err = bindMountFile(hostPath, dstPath, "", 0, true)
		if err != nil {
			return fmt.Errorf("failed to bind mount file %s: %w", hostPath, err)
		}
	}


	// Set up the permissions and ownership of the original file.
	err = unix.Chmod(dstPath, fileInfo.Mode)
	if err != nil {
		return fmt.Errorf("failed to chmod %s: %w", dstPath, err)
	}

	err = os.Chown(dstPath, int(fileInfo.Uid), int(fileInfo.Gid))
	if err != nil {
		return fmt.Errorf("failed to chown %s: %w", dstPath, err)
	}

	return nil
}

// bindMountFile bind mounts a file/directory to a new path
func bindMountFile(hostPath string, dstDir string, dstPath string, perm uint32, isDir bool) error {
	uniklog.WithFields(logrus.Fields{
		"hostPath":  hostPath,
		"dstDir":    dstDir,
		"dstPath":    dstPath,
		"perm": perm,
		"isDir":  isDir,
	}).Debug("bindMountFile")

	err := os.MkdirAll(dstDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dstDir, err)
	}

	if !isDir {
		dstFile, err1 := unix.Open(dstPath, unix.O_CREAT, perm)
		if err1 != nil {
			return fmt.Errorf("failed to create file %s: %w", dstPath, err)
		}
		unix.Close(dstFile)
		err = unix.Mount(hostPath, dstPath, "", unix.MS_BIND|unix.MS_PRIVATE, "")
	} else {
		err = unix.Mount(hostPath, dstDir, "", unix.MS_BIND|unix.MS_PRIVATE, "")
	}
	if err != nil {
		return fmt.Errorf("failed to bind mount %s: %w", dstPath, err)
	}

	return nil
}

// mapRootfsPropagationFlag retrieves the propagation flags of the rootfs
// from the container's configuration
func mapRootfsPropagationFlag(value string) (int, error) {
	mountPropagationMapping := map[string]int{
		"rprivate":    unix.MS_PRIVATE | unix.MS_REC,
		"private":     unix.MS_PRIVATE,
		"rslave":      unix.MS_SLAVE | unix.MS_REC,
		"slave":       unix.MS_SLAVE,
		"rshared":     unix.MS_SHARED | unix.MS_REC,
		"shared":      unix.MS_SHARED,
		"runbindable": unix.MS_UNBINDABLE | unix.MS_REC,
		"unbindable":  unix.MS_UNBINDABLE,
	}

	propagation, exists := mountPropagationMapping[value]
	if !exists {
		return 0, fmt.Errorf("rootfsPropagation=%s is not supported", value)
	}

	return propagation, nil
}

// rootfsParentMountPrivate ensures rootfs parent mount is private.
// This is needed for two reasons:
//   - pivot_root() will fail if parent mount is shared;
//   - when we bind mount rootfs, if its parent is not private, the new mount
//     will propagate (leak!) to parent namespace and we don't want that.
//
// Revisit this: We can remove this if we decide to create our own rootfs for
// the execution of the monitor, since we can make sure that the mount will be private.
func rootfsParentMountPrivate(path string) error {
	var err error
	// Assuming path is absolute and clean.
	// Any error other than EINVAL means we failed,
	// and EINVAL means this is not a mount point, so traverse up until we
	// find one.
	for {
		err = unix.Mount("", path, "", unix.MS_PRIVATE, "")
		if err == nil {
			return nil
		}
		if err != unix.EINVAL || path == "/" {
			break
		}
		path = filepath.Dir(path)
	}

	return fmt.Errorf("Could not remount as private the parent mount of %s", path)
}

// prepareRoot prepares the directory of the container's rootfs to safely pivot
// chroot to it.
func prepareRoot(path string, rootfsPropagation string) error {
	flag := unix.MS_SLAVE | unix.MS_REC
	if rootfsPropagation != "" {
		var err error

		flag, err = mapRootfsPropagationFlag(rootfsPropagation)
		if err != nil {
			return err
		}
	}

	err := unix.Mount("", "/", "", uintptr(flag), "")
	if err != nil {
		return err
	}

	err = rootfsParentMountPrivate(path)
	if err != nil {
		return err
	}

	return unix.Mount(path, path, "bind", unix.MS_BIND|unix.MS_REC, "")
}

// containsNS checks of the container's configuration contains a specific namespace
func containsNS(namespaces []specs.LinuxNamespace, nsType specs.LinuxNamespaceType) bool {
	for _, ns := range namespaces {
		if ns.Type == nsType {
			return true
		}
	}

	return false
}

// findQemuDataDir tries to find the location of data and BIOS files for Qemu.
// At first checks /usr/local/share and if it does not exist, it falls back to
// /usr/share. If /usr/local/share is a soft link, it will find its target.
func findQemuDataDir(basename string) (string, error) {
	qdPath := filepath.Join("/usr/local/share/", basename)
	info, err := os.Lstat(qdPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("failed to get info of %s: %w", qdPath, err)
		}
		return filepath.Join("/usr/share/", basename), nil
	}
	if info.Mode()&os.ModeSymlink != 0 {
		qdPath, err = os.Readlink(qdPath)
		if err != nil {
			return "", fmt.Errorf("failed to get target of %s %w", qdPath, err)
		}
	} else {
		qdPath = filepath.Join("/usr/share/", basename)
	}

	return qdPath, nil
}

func mountVolumes(rootfsPath string, mounts []specs.Mount) error {
	for _, m := range mounts {
		// skip non-bind mounts
		if m.Type != "bind" {
			continue
		}

		var mountFlags int
		var propFlag int
		var makeCopy bool

		// Parse flags
		for _, o := range m.Options {
			if f, err := mapMountFlag(o); err == nil {
				mountFlags |= f
			} else if f, err := mapRootfsPropagationFlag(o); err == nil {
				propFlag = f
				// apply only if its private
				if o == "private" || o == "rprivate" {
					makeCopy = true
				}
			}
		}

		// Determine if source is a file or directory
		var stat unix.Stat_t
		if err := unix.Stat(m.Source, &stat); err != nil {
			return fmt.Errorf("failed to stat %s: %w", m.Source, err)
		}

		isDir := (stat.Mode & unix.S_IFMT) == unix.S_IFDIR
		// apply only if its a file & the mountflag is private
		withCopy := makeCopy && !isDir


		// Mount the file or dir
		err := fileFromHost(rootfsPath, m.Source, m.Destination, withCopy)
		if err != nil {
			return fmt.Errorf("failed to process bind mount %s -> %s: %w", m.Source, m.Destination, err)
		}

		// Apply propagation flag (if any)
		if (isDir && propFlag != 0) {
			dst := filepath.Join(rootfsPath, m.Destination)
			if err := unix.Mount(dst, dst, "", uintptr(propFlag), ""); err != nil {
				return fmt.Errorf("failed to set propagation flag %d for %s: %w", propFlag, dst, err)
			}
		}
	}

	return nil
}

// mapMountFlag retrieves the mount flags of a mount entry
// from the container's configuration
func mapMountFlag(value string) (int, error) {
	mountFlagsMapping := map[string]int{
		"async":         unix.MS_SYNCHRONOUS,
		"atime":         unix.MS_NOATIME,
		"bind":          unix.MS_BIND,
		"defaults":      0,
		"dev":           unix.MS_NODEV,
		"diratime":      unix.MS_NODIRATIME,
		"dirsync":       unix.MS_DIRSYNC,
		"exec":          unix.MS_NOEXEC,
		"iversion":      unix.MS_I_VERSION,
		"lazytime":      unix.MS_LAZYTIME,
		"loud":          unix.MS_SILENT,
		"mand":          unix.MS_MANDLOCK,
		"noatime":       unix.MS_NOATIME,
		"nodev":         unix.MS_NODEV,
		"nodiratime":    unix.MS_NODIRATIME,
		"noexec":        unix.MS_NOEXEC,
		"noiversion":    unix.MS_I_VERSION,
		"nolazytime":    unix.MS_LAZYTIME,
		"nomand":        unix.MS_MANDLOCK,
		"norelatime":    unix.MS_RELATIME,
		"nostrictatime": unix.MS_STRICTATIME,
		"nosuid":        unix.MS_NOSUID,
		"nosymfollow":   unix.MS_NOSYMFOLLOW,
		"rbind":         unix.MS_BIND | unix.MS_REC,
		"relatime":      unix.MS_RELATIME,
		"remount":       unix.MS_REMOUNT,
		"ro":            unix.MS_RDONLY,
		"rw":            unix.MS_RDONLY,
		"silent":        unix.MS_SILENT,
		"strictatime":   unix.MS_STRICTATIME,
		"suid":          unix.MS_NOSUID,
		"sync":          unix.MS_SYNCHRONOUS,
		"symfollow":     unix.MS_NOSYMFOLLOW,
	}

	flag, exists := mountFlagsMapping[value]
	if !exists {
		return 0, fmt.Errorf("mount flag =%s is not supported", value)
	}

	return flag, nil
}
