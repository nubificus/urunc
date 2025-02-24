// Copyright (c) 2023-2024, Nubificus LTD
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

package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"syscall"

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

		if !context.Bool("reexec") {
			return createUnikontainer(context)
		}

		return reexecUnikontainer(context)
	},
}

// createUnikontainer creates a Unikernel struct from bundle data,
// initializes it's base dir and state.json,
// setups terminal if required and spawns reexec process,
// waits for reexec process to notify, executes CreateRuntime hooks,
// sends ACK to reexec process and executes CreateContainer hooks
func createUnikontainer(context *cli.Context) error {
	containerID := context.Args().First()
	if containerID == "" {
		return fmt.Errorf("container id cannot be empty")
	}
	metrics.Capture(containerID, "TS00")

	// We have already made sure in main.go that root is not nil
	rootDir := context.GlobalString("root")

	// bundle option cli option is optional. Therefore the bundle directory
	// is either the CWD or the one defined in the cli option
	bundlePath := context.String("bundle")
	if bundlePath == "" {
		var err error
		bundlePath, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	// new unikernel from bundle
	unikontainer, err := unikontainers.New(bundlePath, containerID, rootDir)
	if err != nil {
		if errors.Is(err, unikontainers.ErrQueueProxy) ||
			errors.Is(err, unikontainers.ErrNotUnikernel) {
			// Exec runc to handle non unikernel containers
			return runcExec()
		}
		return err
	}
	metrics.Capture(containerID, "TS01")

	err = unikontainer.InitialSetup()
	if err != nil {
		return err
	}

	metrics.Capture(containerID, "TS02")

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

	reexecCommand := createReexecCmd()
	reexecCommand.Args = append(os.Args, "--reexec")
	reexecCommand.Env = os.Environ()

	metrics.Capture(containerID, "TS03")
	// setup terminal if required and start reexec process
	if unikontainer.Spec.Process.Terminal {
		ptm, err := pty.Start(reexecCommand)
		if err != nil {
			logrus.WithError(err).Fatal("failed to setup pty and start reexec process")
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

	} else {
		reexecCommand.Stdin = os.Stdin
		reexecCommand.Stdout = os.Stdout
		reexecCommand.Stderr = os.Stderr
		err := reexecCommand.Start()
		if err != nil {
			logrus.WithError(err).Fatal("failed to start reexec process")
		}
	}

	// Wait for reexec process to notify us
	err = unikontainers.AwaitMessage(listener, unikontainers.ReexecStarted)
	if err != nil {
		return err
	}
	metrics.Capture(containerID, "TS07")

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
	metrics.Capture(containerID, "TS08")

	// send ACK to reexec process
	err = unikontainer.SendAckReexec()
	if err != nil {
		return fmt.Errorf("failed to send ACK to reexec process: %w", err)

	}
	metrics.Capture(containerID, "TS09")

	// execute CreateRuntime hooks
	err = unikontainer.ExecuteHooks("CreateContainer")
	if err != nil {
		return fmt.Errorf("failed to execute CreateRuntime hooks: %w", err)
	}
	metrics.Capture(containerID, "TS11")

	return nil
}

func createReexecCmd() *exec.Cmd {
	// create reexec process
	selfPath := "/proc/self/exe"
	reexecCommand := &exec.Cmd{
		Path: selfPath,
		SysProcAttr: &syscall.SysProcAttr{
			Cloneflags: syscall.CLONE_NEWNET,
		},
	}

	return reexecCommand
}

// reexecUnikontainer gets a Unikernel struct from state.json,
// sends ReexecStarted message to init.sock,
// waits AckReexec message on urunc.sock,
// waits StartExecve message on urunc.sock,
// executes Prestart hooks and finally execve's the unikernel vmm.
func reexecUnikontainer(context *cli.Context) error {
	// No need to check if containerID is valid, because it will get
	// checked later. We just want it for the metrics
	containerID := context.Args().First()
	metrics.Capture(containerID, "TS04")

	// get Unikontainer data from state.json
	unikontainer, err := getUnikontainer(context)
	if err != nil {
		return err
	}

	metrics.Capture(containerID, "TS05")

	// send ReexecStarted message to init.sock to parent process
	err = unikontainer.SendReexecStarted()
	if err != nil {
		return err
	}
	metrics.Capture(containerID, "TS06")

	// wait AckReexec message on urunc.sock from parent process
	socketPath := unikontainer.GetUruncSockAddr()
	err = unikontainer.ListenAndAwaitMsg(socketPath, unikontainers.AckReexec)
	if err != nil {
		return err
	}
	metrics.Capture(containerID, "TS10")

	// wait StartExecve message on urunc.sock from urunc start process
	err = unikontainer.ListenAndAwaitMsg(socketPath, unikontainers.StartExecve)
	if err != nil {
		return err
	}
	metrics.Capture(containerID, "TS15")

	unikontainer.State.Pid = os.Getpid()
	err = unikontainer.Create(unikontainer.State.Pid)
	if err != nil {
		return err
	}
	// execute Prestart hooks
	err = unikontainer.ExecuteHooks("Prestart")
	if err != nil {
		return err
	}

	// execve
	return unikontainer.Exec()
}
