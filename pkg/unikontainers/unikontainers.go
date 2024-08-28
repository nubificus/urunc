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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"

	"github.com/nubificus/urunc/pkg/network"
	"github.com/nubificus/urunc/pkg/unikontainers/hypervisors"
	"github.com/nubificus/urunc/pkg/unikontainers/unikernels"
	"github.com/vishvananda/netns"
	"golang.org/x/sys/unix"

	"github.com/nubificus/urunc/internal/constants"
	m "github.com/nubificus/urunc/internal/metrics"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
)

var Log = logrus.WithField("subsystem", "unikontainers")

var ErrQueueProxy = errors.New("This a queue proxy container")
var ErrNotUnikernel = errors.New("This is not a unikernel container")

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

	containerName := spec.Annotations["io.kubernetes.cri.container-name"]
	if containerName == "queue-proxy" {
		logrus.Info("This is a queue-proxy container. Adding IP env.")
		configFile := filepath.Join(bundlePath, "config.json")
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
	// FIXME: We need to find a way to set the output file
	var metrics = m.NewZerologMetrics(constants.TimestampTargetFile)
	err := u.joinSandboxNetNs()
	if err != nil {
		return err
	}

	metrics.Capture(u.State.ID, "TS16")

	vmmType := u.State.Annotations["com.urunc.unikernel.hypervisor"]
	unikernelType := u.State.Annotations["com.urunc.unikernel.unikernelType"]
	unikernelPath := u.State.Annotations["com.urunc.unikernel.binary"]
	initrdPath := u.State.Annotations["com.urunc.unikernel.initrd"]
	rootfsDir := filepath.Join(u.State.Bundle, "rootfs")
	unikernelAbsPath := filepath.Join(rootfsDir, unikernelPath)
	initrdAbsPath := ""
	if initrdPath != "" {
		initrdAbsPath = filepath.Join(rootfsDir, initrdPath)
	}

	// populate vmm args
	vmmArgs := hypervisors.ExecArgs{
		Container:     u.State.ID,
		UnikernelPath: unikernelAbsPath,
		InitrdPath:    initrdAbsPath,
		BlockDevice:   "",
		Seccomp:       true, // Enable Seccomp by default
		Environment:   os.Environ(),
	}

	// Check if container is set to unconfined -- disable seccomp
	if u.Spec.Linux.Seccomp == nil {
		Log.Warn("Seccomp is disabled")
		vmmArgs.Seccomp = false
	}

	// populate unikernel params
	unikernelParams := unikernels.UnikernelParams{
		CmdLine: u.State.Annotations["com.urunc.unikernel.cmdline"],
	}

	// handle network
	netManager, err := network.NewNetworkManager(u.getNetworkType())
	if err != nil {
		return err
	}
	networkInfo, err := netManager.NetworkSetup()
	if err != nil {
		Log.Errorf("Failed to setup network :%v. Possibly due to ctr", err)
	}
	metrics.Capture(u.State.ID, "TS17")

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

	if initrdAbsPath != "" {
		unikernelParams.RootFSType = "initrd"
	} else {
		unikernelParams.RootFSType = ""
	}
	unikernel, err := unikernels.New(unikernels.UnikernelType(unikernelType))
	if err != nil {
		return err
	}

	// handle storage
	// useDevmapper will contain the value of either the annotation (if was set)
	// or from the environment variable. The annotation has more power than the
	// environment variable. However, if none of them is set, we do not take them
	// into consideration, meaning that if the rest of the checks are valid (e.g.
	// no block device in the container, devmapper is in use, unikernel supports
	// block/FS of devmapper) then we wiil use the devmapper as a block device
	// for the unikernel.
	useDevmapper := false
	useDevmapper, err = strconv.ParseBool(u.State.Annotations["com.urunc.unikernel.useDMBlock"])
	if err != nil {
		Log.Errorf("Invalide value in useDMBlock: %s. Urunc will try to use it",
			u.State.Annotations["com.urunc.unikernel.useDMBlock"])
		useDevmapper = true
	}
	if u.State.Annotations["com.urunc.unikernel.block"] != "" && unikernel.SupportsBlock() {
		vmmArgs.BlockDevice = filepath.Join(rootfsDir, u.State.Annotations["com.urunc.unikernel.block"])
	}

	if unikernel.SupportsBlock() && vmmArgs.BlockDevice == "" && useDevmapper {
		rootFsDevice, err := getBlockDevice(rootfsDir)
		if err != nil {
			return err
		}
		if unikernel.SupportsFS(rootFsDevice.FsType) {
			err = prepareDMAsBlock(u.State.Bundle, unikernelPath, "urunc.json", initrdPath)
			if err != nil {
				return err
			}
			vmmArgs.BlockDevice = rootFsDevice.Device
		}
	}
	metrics.Capture(u.State.ID, "TS18")

	// get a new vmm
	vmm, err := hypervisors.NewVMM(hypervisors.VmmType(vmmType))
	if err != nil {
		return err
	}

	err = unikernel.Init(unikernelParams)
	if err != nil {
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
	metrics.Capture(u.State.ID, "TS19")

	// metrics.Wait()
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
	unikernelType := u.State.Annotations["com.urunc.unikernel.unikernelType"]
	unikernel, err := unikernels.New(unikernels.UnikernelType(unikernelType))
	if err != nil {
		return err
	}
	useDevmapper := false
	useDevmapper, err = strconv.ParseBool(u.State.Annotations["com.urunc.unikernel.useDMBlock"])
	if err != nil {
		useDevmapper = true
	}
	annotBlock := u.State.Annotations["com.urunc.unikernel.block"]
	if unikernel.SupportsBlock() && annotBlock == "" && useDevmapper {
		err := cleanupExtractedFiles(u.State.Bundle)
		if err != nil {
			return fmt.Errorf("cannot delete bundle %s: %v", u.State.Bundle, err)
		}
	} else {
		fmt.Printf("Komple")
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

// ExecuteHooks executes concurrently any hooks found in spec based on name:
func (u *Unikontainer) ExecuteHooks(name string) error {
	// NOTICE: It is possible that the concurrent execution of the hooks may cause
	// some unknown problems down the line. Be sure to prioritize checking with sequential
	// hook execution when debugging.

	// More info for individual hooks can be found here:
	// https://github.com/opencontainers/runtime-spec/blob/main/config.md#posix-platform-hooks
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
func (u *Unikontainer) ExecuteHooksSequentially(name string) error {
	// NOTICE: This function is left on purpose to aid future debugging efforts
	// in case concurrent hook execution causes unexpected errors.

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

// getNetworkType checks if current container is a knative user-container
func (u Unikontainer) getNetworkType() string {
	if u.Spec.Annotations["io.kubernetes.cri.container-name"] == "user-container" {
		return "static"
	}
	return "dynamic"
}
