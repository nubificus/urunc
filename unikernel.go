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

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	uvmm "github.com/nubificus/urunc/pkg/hypervisors"
	unet "github.com/nubificus/urunc/pkg/network"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/vishvananda/netns"
	"golang.org/x/sys/unix"
)

type Unikernel struct {
	State   specs.State
	Spec    specs.Spec
	BaseDir string
}

// Reads cli context (bundle data) and creates a Unikernel struct based on that data.
func GetNewUnikernel(context *cli.Context) (*Unikernel, error) {
	// TODO: Check that bundle is not empty (eg in urunc delete)
	bundle, err := filepath.Abs(context.String("bundle"))
	if err != nil {
		return &Unikernel{}, err
	}
	spec, err := LoadSpec(bundle)
	if err != nil {
		return &Unikernel{}, err
	}
	config, err := GetUnikernelConfig(bundle, spec)
	if err != nil {
		return &Unikernel{}, err
	}
	confMap := config.Map()
	id := context.Args().First()
	if id == "" {
		return &Unikernel{}, errors.New("empty id")
	}

	rootDir := context.GlobalString("root")
	if rootDir == "" {
		rootDir = "/run/urunc"
	}
	containerDir := filepath.Join(rootDir, id)

	state := &specs.State{
		Version:     spec.Version,
		ID:          id,
		Status:      "creating",
		Pid:         -1,
		Bundle:      bundle,
		Annotations: confMap,
	}

	return &Unikernel{
		BaseDir: containerDir,
		Spec:    *spec,
		State:   *state,
	}, nil
}

// Reads cli context (bundle data) and retrieves the saved unikernel data from disk
func GetExistingUnikernel(context *cli.Context) (*Unikernel, error) {
	unikernel := &Unikernel{}
	id := context.Args().First()
	if id == "" {
		Log.Error()
		return unikernel, errors.New("empty id")
	}

	rootDir := context.GlobalString("root")
	if rootDir == "" {
		rootDir = "/run/urunc"
	}
	containerDir := filepath.Join(rootDir, id)
	Log.WithField("id", id).WithField("baseDir", containerDir).Info("Found unikernel base directory")
	stateFilePath := filepath.Join(containerDir, "ustate.json")
	Log.WithField("filePath", stateFilePath).Info("Reading state from ustate.json")
	state, err := LoadContainerState(stateFilePath)
	if err != nil {
		return unikernel, err
	}
	unikernel.State = state
	Log.WithField("status", unikernel.State.Status).WithField("PID", unikernel.State.Pid).Info("Loaded unikernel state")

	bundle := unikernel.State.Bundle
	spec, err := LoadSpec(bundle)
	if err != nil {
		return unikernel, err
	}

	unikernel.BaseDir = containerDir
	unikernel.Spec = *spec
	return unikernel, nil
}

// Sets the Unikernel status as creating, creates the Unikernel base directory and then saves the state.json file with the current Unikernel state
func (u *Unikernel) Setup() error {
	u.State.Status = "creating"
	err := os.MkdirAll(u.BaseDir, 0o755)
	if err != nil {
		return err
	}
	err = u.saveContainerState()
	if err != nil {
		return err
	}
	return nil
}

func (u *Unikernel) Create(pid int) error {
	Log.Info("write PID file: ", filepath.Join(u.State.Bundle, "init.pid"))
	err := createPidFile(filepath.Join(u.State.Bundle, "init.pid"), pid)
	if err != nil {
		return err
	}
	u.State.Pid = pid
	u.State.Status = "created"
	err = u.saveContainerState()
	if err != nil {
		return err
	}
	return nil
}

// Start connects to the urunc socket address and sends a "START" message
// to trigger the reexec process to actually execute the Unikernel. After sending the "START"
// message it executes Poststart hooks if defined.
//
// Returns an error if any of the steps fail.
func (u *Unikernel) Start() error {
	sockAddr, err := u.GetUruncSockAddr(true)
	if err != nil {
		Log.WithError(err).Fatal("failed to get urunc sockAddr")
	}
	conn, err := net.Dial("unix", sockAddr)
	if err != nil {
		Log.WithError(err).Fatal("failed to dial urunc sockAddr")
	}
	defer conn.Close()

	if err := sendMessage(conn, "START"); err != nil {
		Log.WithError(err).Fatal("failed to notify OK to reexec")
	}
	if err := u.ExecuteHooks("Poststart"); err != nil {
		Log.WithError(err).Fatal("failed to execute Poststart hooks")
	}

	return nil
}

func (u *Unikernel) Execve() error {
	// Work-around to join the Pause containers network namespace in k8s pods
	sandboxID := u.Spec.Annotations["io.kubernetes.cri.sandbox-id"]
	if sandboxID != "" {
		ctrNamespace := filepath.Base(filepath.Dir(u.BaseDir))
		sandboxStatePath := filepath.Join("/run/containerd/runc", ctrNamespace, sandboxID, "state.json")
		Log.WithFields(logrus.Fields{
			"sandboxID":        sandboxID,
			"basedir":          u.BaseDir,
			"bundle":           u.State.Bundle,
			"containerId":      u.State.ID,
			"sandboxStatePath": sandboxStatePath,
		}).Info("Joining sandbox's netns")
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
	}
	Log.WithFields(logrus.Fields{
		"unikernelType": u.State.Annotations["com.urunc.unikernel.unikernelType"],
		"hypervisor":    u.State.Annotations["com.urunc.unikernel.hypervisor"],
	}).Info("Preparing exec data")
	vmmType := u.State.Annotations["com.urunc.unikernel.hypervisor"]

	vmm, err := uvmm.NewVMM(uvmm.VmmType(vmmType))
	if err != nil {
		return err
	}
	execData := uvmm.ExecData{
		Container:   u.State.ID,
		Unikernel:   "",
		TapDev:      "",
		BlkDev:      "",
		CmdLine:     "",
		Environment: os.Environ(),
	}

	Log.Info("vmm.Path:", vmm.Path())
	unikernelPath := filepath.Join(u.State.Bundle, "rootfs")
	Log.Info("unikernelPath", unikernelPath)
	Log.Info("Annotation com.urunc.unikernel.binary: ", u.State.Annotations["com.urunc.unikernel.binary"])

	unikernelPath = filepath.Join(unikernelPath, u.State.Annotations["com.urunc.unikernel.binary"])
	Log.Info("UnikernelPath 2: ", unikernelPath)
	execData.Unikernel = unikernelPath
	// pass cmdline
	execData.CmdLine = u.State.Annotations["com.urunc.unikernel.cmdline"]

	// let's check if rootfs is block or not
	block, err := GetBlockDevice(filepath.Join(u.State.Bundle, "rootfs"))
	if err != nil {
		return err
	}
	if block.IsBlock {
		Log.WithFields(logrus.Fields{"fstype": block.BlkDevice.Fstype,
			"mountpoint": block.BlkDevice.Mountpoint,
			"device":     block.BlkDevice.Device,
		}).Info("Found block device")
		err = u.handleBlkDevice(block)
		if err != nil {
			Log.WithError(err).Error("failed to handle block device")
			return err
		}
		Log.Info("succeeded to handle block device")

		execData.BlkDev = block.BlkDevice.Device
	}
	Log.Info("creating tap device")
	// let's create the tap device
	networkInfo, err := unet.Setup()
	if err != nil {
		return err
	}
	Log.WithField("tap", networkInfo.TapDevice).Info("Created tap device")
	execData.TapDev = networkInfo.TapDevice
	execData.Network = *networkInfo

	u.State.Status = "running"
	u.State.Pid = os.Getpid()
	err = u.saveContainerState()
	if err != nil {
		return err
	}
	Log.Info("updated ustate.json")
	if err := u.ExecuteHooks("StartContainer"); err != nil {
		Log.WithError(err).Error("failed to execute StartContainer hooks")
		os.Exit(1)
	}
	Log.Error("calling vmm execve")
	return vmm.Execve(execData)
}

// TODO: This never gets called. Remove(?)
func (u *Unikernel) Save() {
	_ = u.saveContainerState()
}

func (u *Unikernel) Kill() error {
	vmmType := u.State.Annotations["com.urunc.unikernel.type"]
	if vmmType == string(uvmm.HedgeVmm) {
		vmm, err := uvmm.NewVMM(uvmm.VmmType(vmmType))
		if err != nil {
			return err
		}
		return vmm.Stop(u.State.ID)
	}
	if syscall.Kill(u.State.Pid, syscall.Signal(0)) == nil {
		return syscall.Kill(u.State.Pid, unix.SIGKILL)
	}
	return nil
}

func (u *Unikernel) Delete() error {
	// Attempt to kill before deleting
	err := u.Kill()
	if err != nil {
		Log.Error(err.Error())
	}

	Log.WithField("baseDir", u.BaseDir).Info("To be deleted")

	err = os.RemoveAll(u.BaseDir)
	if err != nil {
		Log.WithError(err).Error("Failed to delete baseDir")
		return err
	}
	return nil
}

// Saves current Unikernel state as baseDir/ustate.json for later use
func (u *Unikernel) saveContainerState() error {
	// Propagate all annotations from spec to state to solve nerdctl hooks errors.
	// For more info: https://github.com/containerd/nerdctl/issues/133
	for key, value := range u.Spec.Annotations {
		if _, ok := u.State.Annotations[key]; !ok {
			u.State.Annotations[key] = value
		}
	}

	data, err := json.Marshal(u.State)
	if err != nil {
		return fmt.Errorf("failed to serialize container state: %w", err)
	}
	Log.Info("state", u.State)
	Log.Info("spec", u.Spec)

	stateName := filepath.Join(u.BaseDir, "ustate.json")
	if err := os.WriteFile(stateName, data, 0o644); err != nil { //nolint: gosec
		return fmt.Errorf("failed to save container state: %w", err)
	}
	return nil
}

func (u *Unikernel) GetInitSockAddr(mustExist bool) (string, error) {
	if mustExist {
		Log.Info("trying to find existing init.sock")
	} else {
		Log.Info("trying to find or create init.sock")
	}
	Log.Info(u.BaseDir, "/init.sock")
	initSockAddr := filepath.Join(u.BaseDir, "init.sock")
	return initSockAddr, ensureValidSockAddr(initSockAddr, mustExist)
}

func (u *Unikernel) GetUruncSockAddr(mustExist bool) (string, error) {
	Log.Info(u.BaseDir, "/urunc.sock")
	uruncSockAddr := filepath.Join(u.BaseDir, "urunc.sock")
	return uruncSockAddr, ensureValidSockAddr(uruncSockAddr, mustExist)
}

func (u *Unikernel) ExecuteHooks(name string) error {
	// TODO: correctly implement the hook part of the lifecycle specification as described in
	// https://github.com/opencontainers/runtime-spec/blob/main/runtime.md#lifecycle

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
	if name == "CreateRuntime" {
		currentNs, err := netns.GetFromPid(os.Getpid())
		if err != nil {
			Log.WithField("currentNs", "").Info("Failed to get current netns")
		}
		Log.WithField("currentNs", currentNs.String()).Info("Got netns before executing hook")
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

func LoadContainerState(stateFilePath string) (specs.State, error) {
	var state specs.State
	data, err := os.ReadFile(stateFilePath)
	if err != nil {
		return specs.State{}, err
	}

	err = json.Unmarshal(data, &state)
	if err != nil {
		return specs.State{}, err
	}
	return state, nil
}

// Copies unikernel/[binary] and urunc.json file in a tmp dir. Unmounts the block device from rootfs.
// Deletes rootfs and renames tmp to rootfs
func (u *Unikernel) handleBlkDevice(block RootFs) error {
	baseTargetDir := filepath.Join(u.State.Bundle, "tmp")
	currentUnikernelPath := filepath.Join(u.State.Bundle, "rootfs", u.State.Annotations["com.urunc.unikernel.binary"])
	targetUnikernelPath := filepath.Join(baseTargetDir, u.State.Annotations["com.urunc.unikernel.binary"])
	targetUnikernelDir := filepath.Dir(targetUnikernelPath)
	err := os.MkdirAll(targetUnikernelDir, 0755)
	if err != nil {
		return err
	}

	err = MoveFile(currentUnikernelPath, targetUnikernelDir)
	if err != nil {
		return err
	}

	currentConfigPath := filepath.Join(u.State.Bundle, "rootfs", "urunc.json")
	targetConfigPath := filepath.Join(baseTargetDir, "urunc.json")
	targetConfigDir := filepath.Dir(targetConfigPath)
	err = os.MkdirAll(targetConfigDir, 0755)
	if err != nil {
		return err
	}
	err = MoveFile(currentConfigPath, targetConfigDir)
	if err != nil {
		return err
	}
	err = os.RemoveAll(currentUnikernelPath)
	if err != nil {
		return err
	}
	err = os.RemoveAll(currentConfigPath)
	if err != nil {
		return err
	}
	err = UnmountBlockDevice(block.BlkDevice.Mountpoint)
	if err != nil {
		return err
	}
	// Hack to wait for unmount
	cwd, _ := os.Getwd()
	if cwd != u.State.Bundle {
		err = os.Chdir(u.State.Bundle)
		if err != nil {
			return err
		}
	}
	for i := 1; i < 10; i++ {
		if err := os.Remove(block.BlkDevice.Mountpoint); err == nil {
			break
		}
		Log.Info("error removing rootfs: ", err.Error())
		time.Sleep(time.Millisecond * 20)
	}
	if err != nil {
		return err
	}
	return os.Rename(baseTargetDir, block.BlkDevice.Mountpoint)
}
