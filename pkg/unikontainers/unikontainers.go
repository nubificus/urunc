// Copyright 2023 Nubificus LTD.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package unikontainers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/moby/sys/mount"
	"github.com/nubificus/urunc/pkg/network"
	"github.com/nubificus/urunc/pkg/unikontainers/hypervisors"
	"github.com/nubificus/urunc/pkg/unikontainers/unikernels"
	"github.com/vishvananda/netns"
	"golang.org/x/sys/unix"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
)

var Log = logrus.WithField("subsystem", "unikontainers")

// type ExecData struct {
// 	Container     string
// 	Unikernel     string
// 	UnikernelType string
// 	TapDev        string
// 	BlkDev        string
// 	CmdLine       string
// 	Environment   []string
// 	Network       network.UnikernelNetworkInfo
// }

// Unikontainer holds the data necessary to create, manage and delete unikernel containers
type Unikontainer struct {
	State   *specs.State
	Spec    *specs.Spec
	BaseDir string
}

// New parses the bundle and creates a new Unikontainer object
func New(bundlePath string, containerID string, rootDir string) (*Unikontainer, error) {
	spec, err := loadSpec(bundlePath)
	if err != nil {
		return nil, err
	}
	config, err := GetUnikernelConfig(bundlePath, spec)
	if err != nil {
		return nil, err
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
		Spec:    spec,
		State:   state,
	}, nil
}

// Get retrieves unikernel data from disk to create a Unikontainer object
func Get(containerID string, rootDir string) (*Unikontainer, error) {
	u := &Unikontainer{}
	containerDir := filepath.Join(rootDir, containerID)
	stateFilePath := filepath.Join(containerDir, "state.json")
	state, err := loadUnikontainerState(stateFilePath)
	if err != nil {
		return nil, err
	}
	u.State = state

	spec, err := loadSpec(state.Bundle)
	if err != nil {
		return nil, err
	}
	u.BaseDir = containerDir
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
	err := writePidFile(filepath.Join(u.State.Bundle, "init.pid"), pid)
	if err != nil {
		return err
	}
	u.State.Pid = pid
	u.State.Status = specs.StateCreated
	return u.saveContainerState()
}

func (u *Unikontainer) Exec() error {
	err := u.joinSandboxNetNs()
	if err != nil {
		return err
	}
	vmmType := u.State.Annotations["com.urunc.unikernel.hypervisor"]
	unikernelType := u.State.Annotations["com.urunc.unikernel.unikernelType"]
	rootfsDir := filepath.Join(u.State.Bundle, "rootfs")
	// TODO: Hardcoded, we might want to allow the user to specify the
	// address of the initrd file
	InitrdPath := filepath.Join(rootfsDir, "/initrd")
	unikernelAbsPath := filepath.Join(rootfsDir, u.State.Annotations["com.urunc.unikernel.binary"])

	// populate vmm args
	vmmArgs := hypervisors.ExecArgs{
		Container:     u.State.ID,
		UnikernelPath: unikernelAbsPath,
		Environment:   os.Environ(),
	}

	// populate unikernel params
	unikernelParams := unikernels.UnikernelParams{
		CmdLine: u.State.Annotations["com.urunc.unikernel.cmdline"],
	}

	// handle network
	networkInfo, err := network.Setup()
	if err != nil {
		return err
	}

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

	// handle storage
	// TODO: THis needs better handling
	// If we simply want to use the rootfs/initrd or share the FS with the
	// guest, we do not need to pass the container rootfs in the Unikernel.
	// THe user might already specified a specific file (initrd, block device,
	// or shared FS) to pass data to the guest.
	if unikernelType == "unikraft" {
		// For the time being, we only support initrd for Unikraft
		// Check if an initrd file exists in the container rootfs
		if _, err := os.Stat(InitrdPath); err == nil {
			vmmArgs.BlockDevice = InitrdPath
		} else {
			vmmArgs.BlockDevice = ""
		}
	} else {
		rootFsDevice, err := getBlockDevice(rootfsDir)
		if err != nil {
			return err
		}
		if rootFsDevice.IsBlock {
			Log.WithFields(logrus.Fields{"fstype": rootFsDevice.BlkDevice.Fstype,
				"mountpoint": rootFsDevice.BlkDevice.Mountpoint,
				"device":     rootFsDevice.BlkDevice.Device,
			}).Debug("Found block device")

			// extract unikernel
			// FIXME: This approach fills up /run with unikernel binaries and
			// urunc.json files for each unikernel instance we run
			err = u.extractUnikernelFromBlock("tmp")
			if err != nil {
				return err
			}
			// unmount block device
			// FIXME: umount and rm might need some retries
			err := mount.Unmount(rootfsDir)
			if err != nil {
				return err
			}
			// rename tmp to rootfs
			err = os.Remove(rootfsDir)
			if err != nil {
				return err
			}
			err = os.Rename(filepath.Join(u.State.Bundle, "tmp"), rootfsDir)
			if err != nil {
				return err
			}
			vmmArgs.BlockDevice = rootFsDevice.BlkDevice.Device
		}
	}

	// get a new vmm
	vmm, err := hypervisors.NewVMM(hypervisors.VmmType(vmmType))
	if err != nil {
		return err
	}

	// build the unikernel command
	unikernelCmd, err := unikernels.UnikernelCommand(unikernels.UnikernelType(unikernelType), unikernelParams)
	if err != nil {
		return err
	}
	vmmArgs.Command = unikernelCmd

	// update urunc.json state
	u.State.Status = "running"
	u.State.Pid = os.Getpid()
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
	return vmm.Execve(vmmArgs)
}

// Kill stops the VMM process, first by asking the VMM struct to stop
// and consequently by killing the process described in u.State.Pid
func (u *Unikontainer) Kill() error {
	vmmType := u.State.Annotations["com.urunc.unikernel.hypervisor"]
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
		return syscall.Kill(u.State.Pid, unix.SIGKILL)
	}
	return nil
}

// Delete removes the containers base directory and its contents
func (u *Unikontainer) Delete() error {
	if u.isRunning() {
		return fmt.Errorf("cannot delete running unikernel: %s", u.State.ID)
	}
	unikernelType := u.State.Annotations["com.urunc.unikernel.unikernelType"]
	if unikernelType != "unikraft" {
		err := os.RemoveAll(u.State.Bundle)
		if err != nil {
			return fmt.Errorf("cannot delete bundle %s: %v", u.State.Bundle, err)
		}
	}
	return os.RemoveAll(u.BaseDir)
}

// joinSandboxNetns finds the sandbox id of the container, retrieves the sandbox's init pid,
// finds the init pid netns and joins it
func (u Unikontainer) joinSandboxNetNs() error {
	sandboxID := u.Spec.Annotations["io.kubernetes.cri.sandbox-id"]
	if sandboxID == "" {
		return nil
	}
	ctrNamespace := filepath.Base(filepath.Dir(u.BaseDir))
	sandboxStatePath := filepath.Join("/run/containerd/runc", ctrNamespace, sandboxID, "state.json")
	sandboxInitPid, err := getInitPid(sandboxStatePath)
	if err != nil {
		return err
	}
	sandboxInitNetns, err := netns.GetFromPid(int(sandboxInitPid))
	if err != nil {
		return err
	}
	Log.WithFields(logrus.Fields{
		"sandboxInitPid":   sandboxInitPid,
		"sandboxInitNetns": sandboxInitNetns,
	}).Info("Joining sandbox's netns")
	err = netns.Set(sandboxInitNetns)
	if err != nil {
		return err
	}
	Log.Info("Joined sandbox's netns")
	return nil
}

// extractUnikernelFromBlock creates target directory inside the bundle and moves unikernel & urunc.json
// FIXME: This approach fills up /run with unikernel binaries and urunc.json files for each unikernel we run
func (u Unikontainer) extractUnikernelFromBlock(target string) error {
	// create bundle/tmp directory and moves unikernel binary and urunc.json
	tmpDir := filepath.Join(u.State.Bundle, target)
	unikernel := u.State.Annotations["com.urunc.unikernel.binary"]

	currentUnikernelPath := filepath.Join(u.State.Bundle, "rootfs", unikernel)
	targetUnikernelPath := filepath.Join(tmpDir, unikernel)
	targetUnikernelDir, _ := filepath.Split(targetUnikernelPath)

	err := moveFile(currentUnikernelPath, targetUnikernelDir)
	if err != nil {
		return err
	}

	currentConfigPath := filepath.Join(u.State.Bundle, "rootfs", "urunc.json")
	return moveFile(currentConfigPath, tmpDir)
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

	stateName := filepath.Join(u.BaseDir, "state.json")
	return os.WriteFile(stateName, data, 0o644) //nolint: gosec
}

// ExecuteHooks executes any hooks found in spec based on name:
func (u *Unikontainer) ExecuteHooks(name string) error {
	// More info for individual hooks can be found here:
	// https://github.com/opencontainers/runtime-spec/blob/main/config.md#posix-platform-hooks
	Log.Info("Executing ", name, " hooks")
	if u.Spec.Hooks == nil {
		return nil
	}

	hooks := map[string][]specs.Hook{
		"Prestart":        u.Spec.Hooks.Prestart,
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

func (u *Unikontainer) GetInitSockAddr() string {
	return getSockAddr(u.BaseDir, initSock)
}

func (u *Unikontainer) GetUruncSockAddr() string {
	return getSockAddr(u.BaseDir, uruncSock)
}

// ListeAndAwaitMsg opens a new connection to UruncSock and
// waits for the expectedMsg message
func (u *Unikontainer) ListenAndAwaitMsg(sockAddr string, msg IPCMessage) error {
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

// SendReexecStarted sends an ReexecStarted message to InitSock
func (u *Unikontainer) SendReexecStarted() error {
	sockAddr := getInitSockAddr(u.BaseDir)
	return sendIPCMessageWithRetry(sockAddr, ReexecStarted, true)
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
	vmmType := hypervisors.VmmType(u.State.Annotations["com.urunc.unikernel.type"])
	if vmmType != hypervisors.HedgeVmm {
		return syscall.Kill(u.State.Pid, syscall.Signal(0)) == nil
	}
	hedge := hypervisors.Hedge{}
	state := hedge.VMState(u.State.ID)
	return state == "running"
}
