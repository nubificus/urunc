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
	"fmt"
	"net"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/nubificus/urunc/pkg/unikontainers"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"golang.org/x/sys/unix"
)

var createUsage = `<container-id>
Where "<container-id>" is your name for the instance of the container that you
are starting. The name you provide for the container instance must be unique on
your host.`
var createDescription = `
The create command creates an instance of a container for a bundle. The bundle
is a directory with a specification file named "` + specConfig + `" and a root
filesystem.`

var createCommand = cli.Command{
	Name:        "create",
	Usage:       "create a container",
	ArgsUsage:   createUsage,
	Description: createDescription,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "bundle, b",
			Value: "",
			Usage: `path to the root of the bundle directory, defaults to the current directory`,
		},
		cli.StringFlag{
			Name:  "console-socket",
			Value: "",
			Usage: "path to an AF_UNIX socket which will receive a file descriptor referencing the master end of the console's pseudoterminal",
		},
		cli.StringFlag{
			Name:  "pid-file",
			Value: "",
			Usage: "specify the file to write the process id to",
		},
		cli.BoolFlag{
			Name: "reexec",
		},
	},
	Action: func(context *cli.Context) error {
		// FIXME: Remove or change level of log
		logrus.WithField("args", os.Args).Info("urunc INVOKED")
		if err := checkArgs(context, 1, exactArgs); err != nil {
			return err
		}
		err := handleNonBimaContainer(context)
		if err != nil {
			return err
		}

		if !context.Bool("reexec") {
			containerID := context.Args().First()
			nowTime := time.Now().UnixNano()
			metrics.Log(fmt.Sprintf("%s,TS00,%d", containerID, nowTime))
			return createUnikontainer(context)
		}
		containerID := context.Args().First()
		nowTime := time.Now().UnixNano()
		metrics.Log(fmt.Sprintf("%s,TS04,%d", containerID, nowTime))
		return reexecUnikontainer(context)
	},
}

// createUnikontainer creates a Unikernel struct from bundle data, initializes it's base dir and state.json,
// setups terminal if required and spawns reexec process,
// waits for reexec process to notify, executes CreateRuntime hooks,
// sends ACK to reexec process and executes CreateContainer hooks
func createUnikontainer(context *cli.Context) error {
	containerID := context.Args().First()
	rootDir := context.GlobalString("root")
	if rootDir == "" {
		rootDir = "/run/urunc"
	}
	// in the case of urunc create, the bundle is either given or the current directory
	// FIXME: remove the next 2 lines from create and add to commands that require it for CRI
	// ctrNamespace := filepath.Base(rootDir)
	// bundlePath := filepath.Join("/run/containerd/io.containerd.runtime.v2.task/", ctrNamespace, containerID)

	bundlePath := context.String("bundle")

	// new unikernel from bundle
	unikontainer, err := unikontainers.New(bundlePath, containerID, rootDir)
	if err != nil {
		return err
	}
	nowTime := time.Now().UnixNano()
	metrics.Log(fmt.Sprintf("%s,TS01,%d", containerID, nowTime))

	err = unikontainer.InitialSetup()
	if err != nil {
		return err
	}

	nowTime = time.Now().UnixNano()
	metrics.Log(fmt.Sprintf("%s,TS02,%d", containerID, nowTime))
	// Setup a listener for init socket before the creation of reexec process
	sockAddr := unikontainer.GetInitSockAddr()
	listener, err := unikontainers.CreateListener(sockAddr, true)
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

	// create reexec process
	selfBinary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to retrieve urunc executable: %w", err)
	}
	myArgs := os.Args[1:]
	myArgs = append(myArgs, "--reexec")
	reexecCommand := &exec.Cmd{
		Path: selfBinary,
		Args: append([]string{selfBinary}, myArgs...),
		SysProcAttr: &syscall.SysProcAttr{
			Cloneflags: syscall.CLONE_NEWNET,
		},
		Env: os.Environ(),
	}

	// setup terminal if required and start reexec process
	if unikontainer.Spec.Process.Terminal {
		nowTime = time.Now().UnixNano()
		ptm, err := pty.Start(reexecCommand)
		if err != nil {
			logrus.WithError(err).Fatal("failed to create pty")
		}
		defer ptm.Close()
		consoleSocket := context.String("console-socket")
		conn, err := net.Dial("unix", consoleSocket)
		if err != nil {
			logrus.WithError(err).Fatal("failed to dial console socket")
		}
		defer conn.Close()

		uc, ok := conn.(*net.UnixConn)
		if !ok {
			logrus.Fatal("failed to cast unix socket")
		}
		defer uc.Close()

		// Send file descriptor over socket.
		oob := unix.UnixRights(int(ptm.Fd()))
		_, _, err = uc.WriteMsgUnix([]byte(ptm.Name()), oob, nil)
		if err != nil {
			logrus.WithError(err).Fatal("failed to send PTY file descriptor over socket")
		}
		metrics.Log(fmt.Sprintf("%s,TS03,%d", containerID, nowTime))
	} else {
		reexecCommand.Stdin = os.Stdin
		reexecCommand.Stdout = os.Stdout
		reexecCommand.Stderr = os.Stderr
		nowTime = time.Now().UnixNano()
		err := reexecCommand.Start()
		if err != nil {
			logrus.WithError(err).Fatal("failed to start reexec process")
		}
		metrics.Log(fmt.Sprintf("%s,TS03,%d", containerID, nowTime))
	}

	// Wait for reexec process to notify us
	err = unikontainers.AwaitMessage(listener, unikontainers.ReexecStarted)
	if err != nil {
		return err
	}
	nowTime = time.Now().UnixNano()
	metrics.Log(fmt.Sprintf("%s,TS07,%d", containerID, nowTime))
	// Retrieve reexec cmd's pid and write to file and state
	pid := reexecCommand.Process.Pid
	err = unikontainer.Create(pid)
	if err != nil {
		return err
	}

	// execute CreateRuntime hooks
	err = unikontainer.ExecuteHooks("CreateRuntime")
	if err != nil {
		return fmt.Errorf("failed to execute CreateRuntime hooks: %w", err)
	}
	nowTime = time.Now().UnixNano()
	metrics.Log(fmt.Sprintf("%s,TS08,%d", containerID, nowTime))
	// send ACK to reexec process
	err = unikontainer.SendAckReexec()
	if err != nil {
		return fmt.Errorf("failed to send ACK to reexec process: %w", err)

	}
	nowTime = time.Now().UnixNano()
	metrics.Log(fmt.Sprintf("%s,TS09,%d", containerID, nowTime))
	// execute CreateRuntime hooks
	err = unikontainer.ExecuteHooks("CreateContainer")
	if err != nil {
		return fmt.Errorf("failed to execute CreateRuntime hooks: %w", err)
	}
	nowTime = time.Now().UnixNano()
	metrics.Log(fmt.Sprintf("%s,TS11,%d", containerID, nowTime))
	return nil
}

// reexecUnikontainer gets a Unikernel struct from state.json,
// sends ReexecStarted message to init.sock,
// waits AckReexec message on urunc.sock,
// waits StartExecve message on urunc.sock,
// executes Prestart hooks and finally execve's the unikernel vmm.
func reexecUnikontainer(context *cli.Context) error {
	containerID := context.Args().First()
	rootDir := context.GlobalString("root")
	if rootDir == "" {
		rootDir = "/run/urunc"
	}

	// get Unikontainer data from state.json
	unikontainer, err := unikontainers.Get(containerID, rootDir)
	if err != nil {
		return err
	}
	nowTime := time.Now().UnixNano()
	metrics.Log(fmt.Sprintf("%s,TS05,%d", containerID, nowTime))
	// send ReexecStarted message to init.sock to parent process
	err = unikontainer.SendReexecStarted()
	if err != nil {
		return err
	}
	nowTime = time.Now().UnixNano()
	metrics.Log(fmt.Sprintf("%s,TS06,%d", containerID, nowTime))
	// wait AckReexec message on urunc.sock from parent process
	socketPath := unikontainer.GetUruncSockAddr()
	err = unikontainer.ListenAndAwaitMsg(socketPath, unikontainers.AckReexec)
	if err != nil {
		return err
	}
	nowTime = time.Now().UnixNano()
	metrics.Log(fmt.Sprintf("%s,TS10,%d", containerID, nowTime))
	// wait StartExecve message on urunc.sock from urunc start process
	err = unikontainer.ListenAndAwaitMsg(socketPath, unikontainers.StartExecve)
	if err != nil {
		return err
	}
	nowTime = time.Now().UnixNano()
	metrics.Log(fmt.Sprintf("%s,TS15,%d", containerID, nowTime))

	// execute Prestart hooks
	err = unikontainer.ExecuteHooks("Prestart")
	if err != nil {
		return err
	}

	// execve
	return unikontainer.Exec()
}
