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

package unikontainers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/nubificus/urunc/pkg/network"
	"github.com/nubificus/urunc/pkg/unikontainers/hypervisors"
	"github.com/nubificus/urunc/pkg/unikontainers/unikernels"
	"github.com/vishvananda/netlink/nl"
	"golang.org/x/sys/unix"

	"github.com/nubificus/urunc/internal/constants"
	m "github.com/nubificus/urunc/internal/metrics"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
)

var Log = logrus.WithField("subsystem", "unikontainers")

var ErrQueueProxy = errors.New("This a queue proxy container")
var ErrNotUnikernel = errors.New("This is not a unikernel container")

// Unikontainer holds the data necessary to create, manage and delete unikernel containers
type Unikontainer struct {
	State   *specs.State
	Spec    *specs.Spec
	BaseDir string
	RootDir string
}

// New parses the bundle and creates a new Unikontainer object
func New(bundlePath string, containerID string, rootDir string) (*Unikontainer, error) {
	spec, err := loadSpec(bundlePath)
	if err != nil {
		return nil, err
	}

	containerName := spec.Annotations["io.kubernetes.cri.container-name"]
	if containerName == "queue-proxy" {
		logrus.Info("This is a queue-proxy container. Adding IP env.")
		configFile := filepath.Join(bundlePath, configFilename)
		err = handleQueueProxy(*spec, configFile)
		if err != nil {
			return nil, err
		}
		return nil, ErrQueueProxy
	}

	config, err := GetUnikernelConfig(bundlePath, spec)
	if err != nil {
		return nil, ErrNotUnikernel
	}

	confMap := config.Map()
	containerDir := filepath.Join(rootDir, containerID)

	state := &specs.State{
		Version:     spec.Version,
		ID:          containerID,
		Status:      "creating",
		Pid:         -1,
		Bundle:      bundlePath,
		Annotations: confMap,
	}
	return &Unikontainer{
		BaseDir: containerDir,
		RootDir: rootDir,
		Spec:    spec,
		State:   state,
	}, nil
}

// Get retrieves unikernel data from disk to create a Unikontainer object
func Get(containerID string, rootDir string) (*Unikontainer, error) {
	u := &Unikontainer{}
	containerDir := filepath.Join(rootDir, containerID)
	stateFilePath := filepath.Join(containerDir, stateFilename)
	state, err := loadUnikontainerState(stateFilePath)
	if err != nil {
		return nil, err
	}
	if state.Annotations[annotType] == "" {
		return nil, ErrNotUnikernel
	}
	u.State = state

	spec, err := loadSpec(state.Bundle)
	if err != nil {
		return nil, err
	}
	u.BaseDir = containerDir
	u.RootDir = rootDir
	u.Spec = spec
	return u, nil
}

// InitialSetup sets the Unikernel status as creating,
// creates the Unikernel base directory and
// saves the state.json file with the current Unikernel state
func (u *Unikontainer) InitialSetup() error {
	u.State.Status = specs.StateCreating
	// FIXME: should we really create this base dir
	err := os.MkdirAll(u.BaseDir, 0o755)
	if err != nil {
		return err
	}
	return u.saveContainerState()
}

// Create sets the Unikernel status as created,
// and saves the given PID in init.pid
func (u *Unikontainer) Create(pid int) error {
	err := writePidFile(filepath.Join(u.State.Bundle, initPidFilename), pid)
	if err != nil {
		return err
	}
	u.State.Pid = pid
	u.State.Status = specs.StateCreated
	return u.saveContainerState()
}

func (u *Unikontainer) Exec() error {
	// FIXME: We need to find a way to set the output file
	var metrics = m.NewZerologMetrics(constants.TimestampTargetFile)
	metrics.Capture(u.State.ID, "TS15")

	vmmType := u.State.Annotations[annotHypervisor]
	unikernelType := u.State.Annotations[annotType]
	unikernelVersion := u.State.Annotations[annotVersion]

	// TODO: Remove this when we chroot
	unikernelPath, err := filepath.Rel("/", u.State.Annotations[annotBinary])
	if err != nil {
		return err
	}

	// TODO: Remove this when we chroot
	var initrdPath string
	if u.State.Annotations[annotInitrd] != "" {
		initrdPath, err = filepath.Rel("/", u.State.Annotations[annotInitrd])
		if err != nil {
			return err
		}
	}

	// Make sure paths are clean
	bundleDir := filepath.Clean(u.State.Bundle)
	rootfsDir := filepath.Clean(u.Spec.Root.Path)
	if !filepath.IsAbs(rootfsDir) {
		rootfsDir = filepath.Join(bundleDir, rootfsDir)
	}

	// populate vmm args
	vmmArgs := hypervisors.ExecArgs{
		Container:     u.State.ID,
		UnikernelPath: unikernelPath,
		InitrdPath:    initrdPath,
		BlockDevice:   "",
		Seccomp:       true, // Enable Seccomp by default
		MemSizeB:      0,
		Environment:   os.Environ(),
	}

	// Check if memory limit was not set
	if u.Spec.Linux.Resources.Memory != nil {
		if u.Spec.Linux.Resources.Memory.Limit != nil {
			if *u.Spec.Linux.Resources.Memory.Limit > 0 {
				vmmArgs.MemSizeB = uint64(*u.Spec.Linux.Resources.Memory.Limit) // nolint:gosec
			}
		}
	}

	// Check if container is set to unconfined -- disable seccomp
	if u.Spec.Linux.Seccomp == nil {
		Log.Warn("Seccomp is disabled")
		vmmArgs.Seccomp = false
	}

	// populate unikernel params
	unikernelParams := unikernels.UnikernelParams{
		CmdLine: u.Spec.Process.Args,
		EnvVars: u.Spec.Process.Env,
	}
	if len(unikernelParams.CmdLine) == 0 {
		unikernelParams.CmdLine = strings.Fields(u.State.Annotations[annotCmdLine])
	}

	// handle network
	networkType := u.getNetworkType()
	Log.WithField("network type", networkType).Info("Retrieved network type")
	netManager, err := network.NewNetworkManager(networkType)
	if err != nil {
		return err
	}
	networkInfo, err := netManager.NetworkSetup(u.Spec.Process.User.UID, u.Spec.Process.User.GID)
	if err != nil {
		Log.Errorf("Failed to setup network :%v. Possibly due to ctr", err)
	}
	metrics.Capture(u.State.ID, "TS16")

	// if network info is nil, we didn't find eth0, so we are running with ctr
	if networkInfo != nil {
		vmmArgs.TapDevice = networkInfo.TapDevice
		vmmArgs.IPAddress = networkInfo.EthDevice.IP
		// The MAC address for the guest network device is the same as the
		// ethernet device inside the namespace
		vmmArgs.GuestMAC = networkInfo.EthDevice.MAC
		unikernelParams.EthDeviceIP = networkInfo.EthDevice.IP
		unikernelParams.EthDeviceMask = networkInfo.EthDevice.Mask
		unikernelParams.EthDeviceGateway = networkInfo.EthDevice.DefaultGateway
	} else {
		vmmArgs.TapDevice = ""
		vmmArgs.IPAddress = ""
		unikernelParams.EthDeviceIP = ""
		unikernelParams.EthDeviceMask = ""
		unikernelParams.EthDeviceGateway = ""
	}

	if initrdPath != "" {
		unikernelParams.RootFSType = "initrd"
	}

	unikernelParams.Version = unikernelVersion
	unikernel, err := unikernels.New(unikernelType)
	if err != nil {
		return err
	}

	// handle storage
	// useDevmapper will contain the value of either the annotation (if was set)
	// or from the environment variable. The annotation has more power than the
	// environment variable. However, if none of them is set, we do not take them
	// into consideration, meaning that if the rest of the checks are valid (e.g.
	// no block device in the container, devmapper is in use, unikernel supports
	// block/FS of devmapper) then we will use the devmapper as a block device
	// for the unikernel.
	useDevmapper := false
	useDevmapper, err = strconv.ParseBool(u.State.Annotations[annotUseDMBlock])
	if err != nil {
		Log.Errorf("Invalid value in useDMBlock: %s. Urunc will try to use it",
			u.State.Annotations[annotUseDMBlock])
		useDevmapper = true
	}
	if u.State.Annotations[annotBlock] != "" && unikernel.SupportsBlock() {
		// TODO: Remove this when we chroot
		vmmArgs.BlockDevice, err = filepath.Rel("/", u.State.Annotations[annotBlock])
		unikernelParams.RootFSType = "block"
		if err != nil {
			return err
		}
	}

	if unikernel.SupportsBlock() && vmmArgs.BlockDevice == "" && useDevmapper {
		rootFsDevice, err := getBlockDevice(rootfsDir)
		if err != nil {
			return err
		}
		if unikernel.SupportsFS(rootFsDevice.FsType) {
			err = prepareDMAsBlock(rootFsDevice.Path, unikernelPath, uruncJSONFilename, initrdPath)
			if err != nil {
				return err
			}
			vmmArgs.BlockDevice = rootFsDevice.Device
			unikernelParams.RootFSType = "block"
		}
	}
	metrics.Capture(u.State.ID, "TS17")

	// Set CWD the rootfs of the container
	err = os.Chdir(rootfsDir)
	if err != nil {
		return err
	}

	// get a new vmm
	vmm, err := hypervisors.NewVMM(hypervisors.VmmType(vmmType))
	if err != nil {
		return err
	}

	err = unikernel.Init(unikernelParams)
	if err == unikernels.ErrUndefinedVersion || err == unikernels.ErrVersionParsing {
		Log.WithError(err).Error("an error occurred while initializing the unikernel")
	} else if err != nil {
		return err
	}
	// build the unikernel command
	unikernelCmd, err := unikernel.CommandString()
	if err != nil {
		return err
	}
	vmmArgs.Command = unikernelCmd

	// update urunc.json state
	u.State.Status = "running"
	err = u.saveContainerState()
	if err != nil {
		return err
	}

	// execute hooks
	err = u.ExecuteHooks("StartContainer")
	if err != nil {
		return err
	}
	Log.Info("calling vmm execve")
	metrics.Capture(u.State.ID, "TS18")

	// We might not have write access to the container's rootfs if we switch
	// to a non-root user. Hence, create a file for FC's configuration
	// and change its permissions according to the new user.
	// TODO: We have to remove this then we chroot
	if vmmType == "firecracker" {
		fcf, err := os.Create(hypervisors.FCJsonFilename)
		if err != nil {
			return err
		}
		err = fcf.Chmod(0666)
		if err != nil {
			return err
		}
		err = fcf.Close()
		if err != nil {
			return err
		}
	}

	// Setup uid, gid and additional groups for the monitor process
	err = setupUser(u.Spec.Process.User)
	if err != nil {
		return err
	}

	// metrics.Wait()
	return vmm.Execve(vmmArgs, unikernel)
}

func setupUser(user specs.User) error {
	runtime.LockOSThread()
	// Set the user for the current go routine to exec the Monitor
	AddGidsLen := len(user.AdditionalGids)
	if AddGidsLen > 0 {
		err := unix.Setgroups(convertUint32ToIntSlice(user.AdditionalGids, AddGidsLen))
		if err != nil {
			return fmt.Errorf("could not set Additional groups %v : %v", user.AdditionalGids, err)
		}
	}

	err := unix.Setgid(int(user.GID))
	if err != nil {
		return fmt.Errorf("could not set gid %d: %v", user.GID, err)
	}

	err = unix.Setuid(int(user.UID))
	if err != nil {
		return fmt.Errorf("could not set uid %d: %v", user.UID, err)
	}

	return nil
}

// Kill stops the VMM process, first by asking the VMM struct to stop
// and consequently by killing the process described in u.State.Pid
func (u *Unikontainer) Kill() error {
	vmmType := u.State.Annotations[annotHypervisor]
	vmm, err := hypervisors.NewVMM(hypervisors.VmmType(vmmType))
	if err != nil {
		return err
	}
	err = vmm.Stop(u.State.ID)
	if err != nil {
		return err
	}

	// Check if pid is running
	if syscall.Kill(u.State.Pid, syscall.Signal(0)) == nil {
		err = syscall.Kill(u.State.Pid, unix.SIGKILL)
		if err != nil {
			return err
		}
	}
	// If PID is running we need to kill the process
	// Once the process is dead, we need to enter the network namespace
	// and delete the TC rules and TAP device
	err = u.joinSandboxNetNs()
	if err != nil {
		Log.Errorf("failed to join sandbox netns: %v", err)
		return nil
	}
	// TODO: tap0_urunc should not be hardcoded
	err = network.Cleanup("tap0_urunc")
	if err != nil {
		Log.Errorf("failed to delete tap0_urunc: %v", err)
	}
	return nil
}

// Delete removes the containers base directory and its contents
func (u *Unikontainer) Delete() error {
	if u.isRunning() {
		return fmt.Errorf("cannot delete running unikernel: %s", u.State.ID)
	}
	unikernelType := u.State.Annotations[annotType]
	unikernel, err := unikernels.New(unikernelType)
	if err != nil {
		return err
	}
	useDevmapper := false
	useDevmapper, err = strconv.ParseBool(u.State.Annotations[annotUseDMBlock])
	if err != nil {
		useDevmapper = true
	}
	annotBlock := u.State.Annotations[annotBlock]
	if unikernel.SupportsBlock() && annotBlock == "" && useDevmapper {
		// Make sure paths are clean
		bundleDir := filepath.Clean(u.State.Bundle)
		rootfsDir := filepath.Clean(u.Spec.Root.Path)
		if !filepath.IsAbs(rootfsDir) {
			rootfsDir = filepath.Join(bundleDir, rootfsDir)
		}
		err := cleanupExtractedFiles(rootfsDir)
		if err != nil {
			return fmt.Errorf("cannot delete rootfs %s: %v", rootfsDir, err)
		}
	}
	return os.RemoveAll(u.BaseDir)
}

// joinSandboxNetns joins the network namespace of the sandbox (pause container).
// This function should be called only from a locked thread
// (i.e. runtime. LockOSThread())
func (u Unikontainer) joinSandboxNetNs() error {
	var netNsPath string
	// We want enter the network namespace of the container.
	// There are two possibilities:
	// 1. The unikernel was running inside a Pod and hence we need to join
	//    the namespace of the pause container
	// 2. The unikernel was running in its own network namespace (typical
	//    in docker, nerdctl etc.). If that is the case, then when the
	//    unikernel dies/exits the namespace will also die, since there will
	//    not be any process in that namespace. Therefore, the cleanup will
	//    happen automatically and we do not need to care about that.
	// Therefore, focus only in the first case above.
	for _, ns := range u.Spec.Linux.Namespaces {
		if ns.Type == specs.NetworkNamespace {
			if ns.Path == "" {
				// We had to create the network namespace, when
				// creating the container. Therefore, the namespace
				// will die along with the unikernel.
				return nil
			}
			err := checkValidNsPath(ns.Path)
			if err == nil {
				netNsPath = ns.Path
			} else {
				return err
			}
			break
		}
	}

	Log.WithFields(logrus.Fields{
		"path": netNsPath,
	}).Info("Joining network namespace")
	fd, err := unix.Open(netNsPath, unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		return fmt.Errorf("Error opening namespace path: %w", err)
	}
	err = unix.Setns(int(fd), unix.CLONE_NEWNET)
	if err != nil {
		return fmt.Errorf("Error joining namespace: %w", err)
	}
	Log.Info("Joined network namespace")
	return nil
}

// Saves current Unikernel state as baseDir/state.json for later use
func (u *Unikontainer) saveContainerState() error {
	// Propagate all annotations from spec to state to solve nerdctl hooks errors.
	// For more info: https://github.com/containerd/nerdctl/issues/133
	for key, value := range u.Spec.Annotations {
		if _, ok := u.State.Annotations[key]; !ok {
			u.State.Annotations[key] = value
		}
	}

	data, err := json.Marshal(u.State)
	if err != nil {
		return err
	}

	stateName := filepath.Join(u.BaseDir, stateFilename)
	return os.WriteFile(stateName, data, 0o644) //nolint: gosec
}

func (u *Unikontainer) ExecuteHooks(name string) error {
	// NOTICE: This wrapper function provides an easy way to toggle between
	// the sequential and concurrent hook execution. By default the hooks are executed concurrently.
	// To execute hooks sequentially, change the following line to:
	// if false
	//if true {
	//	return u.executeHooksConcurrently(name)
	//}
	return u.executeHooksSequentially(name)
}

// ExecuteHooks executes concurrently any hooks found in spec based on name:
func (u *Unikontainer) executeHooksConcurrently(name string) error {
	// NOTICE: It is possible that the concurrent execution of the hooks may cause
	// some unknown problems down the line. Be sure to prioritize checking with sequential
	// hook execution when debugging.

	// More info for individual hooks can be found here:
	// https://github.com/opencontainers/runtime-spec/blob/main/config.md#posix-platform-hooks
	Log.Infof("Executing %s hooks", name)
	if u.Spec.Hooks == nil {
		return nil
	}
	hooks := map[string][]specs.Hook{
		// TODO: Prestart is deprecated
		"Prestart":        u.Spec.Hooks.Prestart, // nolint:staticcheck
		"CreateRuntime":   u.Spec.Hooks.CreateRuntime,
		"CreateContainer": u.Spec.Hooks.CreateContainer,
		"StartContainer":  u.Spec.Hooks.StartContainer,
		"Poststart":       u.Spec.Hooks.Poststart,
		"Poststop":        u.Spec.Hooks.Poststop,
	}[name]

	if len(hooks) == 0 {
		Log.WithFields(logrus.Fields{
			"id":    u.State.ID,
			"name:": name,
		}).Debug("No hooks")
		return nil
	}

	s, err := json.Marshal(u.State)
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	errChan := make(chan error, len(hooks))

	for _, hook := range hooks {
		wg.Add(1)
		go u.executeHook(hook, s, &wg, errChan)
	}
	wg.Wait()
	close(errChan)
	for err := range errChan {
		Log.WithField("error", err.Error()).Error("failed to execute hooks")
		return err
	}
	return nil
}

func (u *Unikontainer) executeHook(hook specs.Hook, state []byte, wg *sync.WaitGroup, errChan chan<- error) {
	defer wg.Done()

	var stdout, stderr bytes.Buffer
	cmd := exec.Cmd{
		Path:   hook.Path,
		Args:   hook.Args,
		Env:    hook.Env,
		Stdin:  bytes.NewReader(state),
		Stdout: &stdout,
		Stderr: &stderr,
	}

	Log.WithFields(logrus.Fields{
		"cmd":  cmd.String(),
		"path": hook.Path,
		"args": hook.Args,
		"env":  hook.Env,
	}).Info("executing hook")

	if err := cmd.Run(); err != nil {
		Log.WithFields(logrus.Fields{
			"id":     u.State.ID,
			"error":  err.Error(),
			"cmd":    cmd.String(),
			"stderr": stderr.String(),
			"stdout": stdout.String(),
		}).Error("failed to execute hook")
		errChan <- fmt.Errorf("failed to execute hook '%s': %w", cmd.String(), err)
	}
}

// ExecuteHooks executes sequentially any hooks found in spec based on name:
func (u *Unikontainer) executeHooksSequentially(name string) error {
	// NOTICE: This function is left on purpose to aid future debugging efforts
	// in case concurrent hook execution causes unexpected errors.

	// More info for individual hooks can be found here:
	// https://github.com/opencontainers/runtime-spec/blob/main/config.md#posix-platform-hooks
	Log.Infof("Executing %s hooks", name)
	if u.Spec.Hooks == nil {
		return nil
	}

	hooks := map[string][]specs.Hook{
		// TODO: Prestart is deprecated
		"Prestart":        u.Spec.Hooks.Prestart, // nolint:staticcheck
		"CreateRuntime":   u.Spec.Hooks.CreateRuntime,
		"CreateContainer": u.Spec.Hooks.CreateContainer,
		"StartContainer":  u.Spec.Hooks.StartContainer,
		"Poststart":       u.Spec.Hooks.Poststart,
		"Poststop":        u.Spec.Hooks.Poststop,
	}[name]

	Log.Infof("Found %d %s hooks", len(hooks), name)

	if len(hooks) == 0 {
		Log.WithFields(logrus.Fields{
			"id":    u.State.ID,
			"name:": name,
		}).Debug("No hooks")
		return nil
	}

	s, err := json.Marshal(u.State)
	if err != nil {
		return err
	}
	for _, hook := range hooks {
		var stdout, stderr bytes.Buffer
		cmd := exec.Cmd{
			Path:   hook.Path,
			Args:   hook.Args,
			Env:    hook.Env,
			Stdin:  bytes.NewReader(s),
			Stdout: &stdout,
			Stderr: &stderr,
		}

		if err := cmd.Run(); err != nil {
			Log.WithFields(logrus.Fields{
				"id":     u.State.ID,
				"name:":  name,
				"error":  err.Error(),
				"stderr": stderr.String(),
				"stdout": stdout.String(),
			}).Error("failed to execute hooks")
			return fmt.Errorf("failed to execute %s hook '%s': %w", name, cmd.String(), err)
		}
	}
	return nil
}

// loadUnikontainerState returns a specs.State object containing the info
// found in stateFilePath
func loadUnikontainerState(stateFilePath string) (*specs.State, error) {
	var state specs.State
	data, err := os.ReadFile(stateFilePath)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &state)
	if err != nil {
		return nil, err
	}
	return &state, nil
}

// FormatNsenterInfo encodes namespace info in netlink binary format
// as a io.Reader, in order to send the info to nsenter.
// The implementation is inspired from:
// https://github.com/opencontainers/runc/blob/c8737446d2f99c1b7f2fcf374a7ee5b4519b2051/libcontainer/container_linux.go#L1047
func (u *Unikontainer) FormatNsenterInfo() (rdr io.Reader, retErr error) {
	r := nl.NewNetlinkRequest(int(initMsg), 0)

	// Our custom messages cannot bubble up an error using returns, instead
	// they will panic with the specific error type, netlinkError. In that
	// case, recover from the panic and return that as an error.
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(netlinkError); ok {
				retErr = e.error
			} else {
				panic(r)
			}
		}
	}()

	const numNS = 8
	var writePaths bool
	var writeFlags bool
	var cloneFlags uint32
	var nsPaths [numNS]string // We have 8 namespaces right now
	// We need to set the namespace paths in a specific order.
	// The order should be: user, ipc, uts, net, pid, mount, cgroup, time
	// Therefore, the first element of the above array holds the path of user
	// namespace, while the last element, the time namespace path
	// Order does not matter in clone flags
	for _, ns := range u.Spec.Linux.Namespaces {
		// If the path is empty, then we have to create it.
		// Otherwise, we store the path to the respective element
		// of the array.
		switch ns.Type {
		// Comment out User namespace for the time being and just ignore them
		// They require better handling for cleaning up and we will address
		// it in another iteration.
		// TODO User namespace
		// case specs.UserNamespace:
		// 	if ns.Path == "" {
		// 		cloneFlags |= unix.CLONE_NEWUSER
		// 	} else {
		// 		err := checkValidNsPath(ns.Path)
		// 		if err == nil {
		// 			nsPaths[0] = "user:" + ns.Path
		// 		} else {
		// 			return nil, err
		// 		}
		// 	}
		case specs.IPCNamespace:
			if ns.Path == "" {
				cloneFlags |= unix.CLONE_NEWIPC
			} else {
				err := checkValidNsPath(ns.Path)
				if err == nil {
					nsPaths[1] = "ipc:" + ns.Path
				} else {
					return nil, err
				}
			}
		case specs.UTSNamespace:
			if ns.Path == "" {
				cloneFlags |= unix.CLONE_NEWUTS
			} else {
				err := checkValidNsPath(ns.Path)
				if err == nil {
					nsPaths[2] = "uts:" + ns.Path
				} else {
					return nil, err
				}
			}
		case specs.NetworkNamespace:
			if ns.Path == "" {
				cloneFlags |= unix.CLONE_NEWNET
			} else {
				err := checkValidNsPath(ns.Path)
				if err == nil {
					nsPaths[3] = "net:" + ns.Path
				} else {
					return nil, err
				}
			}
		case specs.PIDNamespace:
			if ns.Path == "" {
				cloneFlags |= unix.CLONE_NEWPID
			} else {
				err := checkValidNsPath(ns.Path)
				if err == nil {
					nsPaths[4] = "pid:" + ns.Path
				} else {
					return nil, err
				}
			}
		case specs.MountNamespace:
			if ns.Path == "" {
				cloneFlags |= unix.CLONE_NEWNS
			} else {
				err := checkValidNsPath(ns.Path)
				if err == nil {
					nsPaths[5] = "mnt:" + ns.Path
				} else {
					return nil, err
				}
			}
		case specs.CgroupNamespace:
			if ns.Path == "" {
				cloneFlags |= unix.CLONE_NEWCGROUP
			} else {
				err := checkValidNsPath(ns.Path)
				if err == nil {
					nsPaths[6] = "cgroup:" + ns.Path
				} else {
					return nil, err
				}
			}
		case specs.TimeNamespace:
			if ns.Path == "" {
				cloneFlags |= unix.CLONE_NEWTIME
			} else {
				err := checkValidNsPath(ns.Path)
				if err == nil {
					nsPaths[7] = "time:" + ns.Path
				} else {
					return nil, err
				}
			}
		default:
			Log.Warn("Unsupported namespace: ", ns.Type, " .It will get ignored")
			continue
		}
		if ns.Path == "" {
			writeFlags = true
		} else {
			writePaths = true
		}
	}

	if writeFlags {
		r.AddData(&int32msg{
			Type:  cloneFlagsAttr,
			Value: uint32(cloneFlags),
		})
	}

	var nsStringBuilder strings.Builder
	if writePaths {
		for i := 0; i < numNS; i++ {
			if nsPaths[i] != "" {
				if nsStringBuilder.Len() > 0 {
					nsStringBuilder.WriteString(",")
				}
				nsStringBuilder.WriteString(nsPaths[i])
			}
		}

		r.AddData(&bytemsg{
			Type:  nsPathsAttr,
			Value: []byte(nsStringBuilder.String()),
		})

	}

	// Setup uid/gid mappings only in the case we need to create a new
	// user namespace. As far as I understand (and I might be very wrong),
	// we can set up the uid/gid mappings only once in a user namespace.
	// Therefore, if we enter a user namespace and try to set the uid/gid
	// mappings, we will get EPERM. Therefore, it is important to note that
	// according to runc, when the config instructs us to use an existing
	// user namespace, the uid/gid mappings should be empty and hence
	// inherit the ones that are already set. Check:
	// https://github.com/opencontainers/runc/blob/e0e22d33eabc4dc280b7ca0810ed23049afdd370/libcontainer/specconv/spec_linux.go#L1036

	// TODO: Add it when we add user namespaces
	// if nsPaths[0] == "" {
	// 	// write uid mappings
	// 	if len(u.Spec.Linux.UIDMappings) > 0 {
	// 		// TODO: Rootless
	// 		b, err := encodeIDMapping(u.Spec.Linux.UIDMappings)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		r.AddData(&bytemsg{
	// 			Type:  uidmapAttr,
	// 			Value: b,
	// 		})
	// 	}
	// 	// write gid mappings
	// 	if len(u.Spec.Linux.GIDMappings) > 0 {
	// 		b, err := encodeIDMapping(u.Spec.Linux.GIDMappings)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		r.AddData(&bytemsg{
	// 			Type:  gidmapAttr,
	// 			Value: b,
	// 		})
	// 		// TODO: Rootless
	// 	}
	// }

	return bytes.NewReader(r.Serialize()), nil
}

func GetUruncSockAddr(baseDir string) string {
	return getSockAddr(baseDir, uruncSock)
}

// ListeAndAwaitMsg opens a new connection to UruncSock and
// waits for the expectedMsg message
func ListenAndAwaitMsg(sockAddr string, msg IPCMessage) error {
	listener, err := CreateListener(sockAddr, true)
	if err != nil {
		return err
	}
	defer func() {
		err = listener.Close()
		if err != nil {
			logrus.WithError(err).Error("failed to close listener")
		}
	}()
	defer func() {
		err = syscall.Unlink(sockAddr)
		if err != nil {
			logrus.WithError(err).Errorf("failed to unlink %s", sockAddr)
		}
	}()
	return AwaitMessage(listener, msg)
}

// SendAckReexec sends an AckReexec message to UruncSock
func (u *Unikontainer) SendAckReexec() error {
	sockAddr := getUruncSockAddr(u.BaseDir)
	return sendIPCMessageWithRetry(sockAddr, AckReexec, true)
}

// SendStartExecve sends an StartExecve message to UruncSock
func (u *Unikontainer) SendStartExecve() error {
	sockAddr := getUruncSockAddr(u.BaseDir)
	return sendIPCMessageWithRetry(sockAddr, StartExecve, true)
}

// isRunning returns true if the PID is alive or hedge.ListVMs returns our containerID
func (u *Unikontainer) isRunning() bool {
	vmmType := hypervisors.VmmType(u.State.Annotations[annotType])
	if vmmType != hypervisors.HedgeVmm {
		return syscall.Kill(u.State.Pid, syscall.Signal(0)) == nil
	}
	hedge := hypervisors.Hedge{}
	state := hedge.VMState(u.State.ID)
	return state == "running"
}

// getNetworkType checks if current container is a knative user-container
func (u Unikontainer) getNetworkType() string {
	if u.Spec.Annotations["io.kubernetes.cri.container-name"] == "user-container" {
		return "static"
	}
	return "dynamic"
}
