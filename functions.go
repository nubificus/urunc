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
	"net"
	"os"
	"os/exec"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"

	"github.com/urfave/cli"
)

// creates a Unikernel struct from bundle data, initializes it's base dir and state.json.
// Then initializes an exec.Cmd struct  with the identican arguments and the addition of the --reexec arg,
// If terminal is required, a PTY is created and connected to the new process, else stdout/stdin/stderr is used.
// The new process (exec.Cmd) is started.
// Then, the initSock is opened and a listener is attached. The reexec process will send a message once it is started.
// At this point, we close our initSock and send a START message to the reexec process through the uruncSock, so it can go on with the execution.
func setupUnikernelContainer(context *cli.Context) {
	Log.Info("Creating unikernel struct from cli context")
	unikernel, err := GetNewUnikernel(context)
	if err != nil {
		Log.WithError(err).Fatal("Failed to create Unikernel")
	}
	err = unikernel.Setup()

	if err != nil {
		Log.WithError(err).Fatal("failed to load spec from absolute bundle path")
	}

	self, err := os.Executable()
	if err != nil {
		Log.WithError(err).Fatal("failed to retrieve executable")
	}
	env := os.Environ()
	myArgs := os.Args[1:]
	myArgs = append(myArgs, "--reexec")
	cmd := &exec.Cmd{
		Path: self,
		Args: append([]string{"urunc"}, myArgs...),
		SysProcAttr: &syscall.SysProcAttr{
			Cloneflags: syscall.CLONE_NEWNET,
		},
		Env: env,
	}

	if unikernel.Spec.Process.Terminal {
		Log.Info("Container requires terminal")
		ptm, err := pty.Start(cmd)
		if err != nil {
			Log.WithError(err).Fatal("failed to create pty")
		}
		defer ptm.Close()

		socket := context.String("console-socket")
		// Connect to the socket in order to send the PTY file descriptor.
		conn, err := net.Dial("unix", socket)
		if err != nil {
			Log.WithError(err).Fatal("failed to dial console socket")
		}
		defer conn.Close()

		uc, ok := conn.(*net.UnixConn)
		if !ok {
			Log.Fatal("failed to cast unix socket")
		}
		defer uc.Close()

		// Send file descriptor over socket.
		oob := unix.UnixRights(int(ptm.Fd()))
		_, _, err = uc.WriteMsgUnix([]byte(ptm.Name()), oob, nil)
		if err != nil {
			Log.WithError(err).Fatal("failed to send file descriptor over socket")
		}
	} else {
		Log.Info("Container doesn't require terminal")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			Log.WithError(err).Fatal("failed to start reexec process")
		}
	}

	initSockAddr, err := unikernel.GetInitSockAddr(false)
	if err != nil {
		Log.WithError(err).Fatal("failed to get init socket")
	}
	initListener, err := net.Listen("unix", initSockAddr)
	if err != nil {
		Log.WithError(err).Fatal("failed to start listener for init socket")
	}

	initConn, err := initListener.Accept()
	if err != nil {
		Log.WithError(err).Fatal("init accept error")
	}
	defer initConn.Close()

	if err := awaitMessage(initConn, "BOOTED"); err != nil {
		Log.WithError(err).Fatal("await message error")
	}
	Log.Info("Container booted")
	if err := initConn.Close(); err != nil {
		Log.WithError(err).Fatal("Failed to close connection")
	}
	if err := initListener.Close(); err != nil {
		Log.WithError(err).Fatal("Failed to close listener")
	}
	// TODO: Investigate why unlink fails
	if err := syscall.Unlink(initSockAddr); err != nil {
		Log.WithError(err).Error("Failed to unlink socket")
	}

	// Retrieve reexec cmd's pid and write to file and state
	pid := cmd.Process.Pid
	err = unikernel.Create(pid)
	if err != nil {
		Log.WithError(err).Error("failed to update state")
		os.Exit(1) //nolint: gocritic
	}

	// Hack to make sure the urunc.sock socket is created
	// TODO: Fix this properly
	// time.Sleep(time.Millisecond * 50)

	//  Then connect to urunc sock address and send OK => for the reexec process to proceed
	sockAddr, err := unikernel.GetUruncSockAddr(true)
	if err != nil {
		Log.WithError(err).Error("failed to get urunc sockAddr")
		os.Exit(1)
	}
	conn, err := net.Dial("unix", sockAddr)
	if err != nil {
		Log.WithError(err).Error("failed to dial urunc sockAddr")
		os.Exit(1)
	}
	defer conn.Close()

	err = os.Chdir("/")
	if err != nil {
		Log.WithError(err).Error("failed to chdir")
	}
	if err := unikernel.ExecuteHooks("CreateRuntime"); err != nil {
		Log.WithError(err).Error("failed to execute CreateRuntime hooks")
		os.Exit(1)
	}

	// Notify the container that it can continue its initialization.
	if err := sendMessage(conn, "OK"); err != nil {
		Log.WithError(err).Error("failed to notify OK to reexec")
		os.Exit(1)
	}
	if err := unikernel.ExecuteHooks("CreateContainer"); err != nil {
		Log.WithError(err).Error("failed to execute CreateContainer hooks")
		os.Exit(1)
	}
}

func createUnikernelContainer(context *cli.Context) {
	unikernelName := context.Args().First()
	Log.WithField("name", unikernelName).Info("Creating unikernel resources")

	unikernel, err := GetExistingUnikernel(context)
	if err != nil {
		Log.Error("Create error: ", err.Error())
		os.Exit(-1)
	}

	Log.Info("Opening init.sock to notify 1st process")
	initSockAddr, err := unikernel.GetInitSockAddr(true)
	if err != nil {
		Log.WithError(err).Error("failed to get init socket")
		os.Exit(-1)
	}
	initConn, err := net.Dial("unix", initSockAddr)
	if err != nil {
		Log.WithError(err).Error("failed to dial init socket")
		os.Exit(-1)
	}
	defer initConn.Close()

	Log.Info("Opening urunc.sock to listen 1st process")
	// Create a new socket to allow communication with this container.
	sockAddr, err := unikernel.GetUruncSockAddr(false)
	if err != nil {
		Log.WithError(err).Error("failed to get urunc socket addr")
		os.Exit(-1) //nolint: gocritic
	}

	listener, err := net.Listen("unix", sockAddr)
	if err != nil {
		Log.WithError(err).Error("listen error")
		os.Exit(-1)
	}
	defer listener.Close()

	// Notify the host that we are alive.
	if err := sendMessage(initConn, "BOOTED"); err != nil {
		Log.WithError(err).Error("failed to notify setup BOOTED")
		os.Exit(-1)
	}
	Log.Info("Notified 1st process we have successfully booted")

	conn, err := listener.Accept()
	if err != nil {
		Log.WithError(err).Error("accept error")
		os.Exit(-1)
	}
	defer conn.Close()

	if err := awaitMessage(conn, "OK"); err != nil {
		Log.WithError(err).Error("error awaiting OK message")
		os.Exit(-1)
	}

	Log.Info("ready to start vmm")
	conn, err = listener.Accept()
	if err != nil {
		Log.WithError(err).Error("accept error")
		os.Exit(-1)
	}
	defer conn.Close()

	if err := awaitMessage(conn, "START"); err != nil {
		Log.WithError(err).Error("error awaiting START message")
		os.Exit(-1)
	}
	Log.Info("Starting vmm")

	if err := unikernel.ExecuteHooks("Prestart"); err != nil {
		Log.WithError(err).Error("failed to execute Prestart hooks")
		os.Exit(1)
	}

	if err := unikernel.Execve(); err != nil {
		Log.WithError(err).Error("failed to execve")
		os.Exit(-1)
	}
	os.Exit(1)
}

func startUnikernelContainer(context *cli.Context) error {
	unikernelName := context.Args().First()
	Log.Info("Start: ", unikernelName)
	unikernel, err := GetExistingUnikernel(context)
	Log.Info("Terminal: ", unikernel.Spec.Process.Terminal)
	if err != nil {
		return err
	}

	Log.WithField("State annotations", unikernel.State.Annotations).Info("ANNOTATIONS")
	Log.WithField("Spec annotations", unikernel.Spec.Annotations).Info("ANNOTATIONS")

	return unikernel.Start()
}

func deleteUnikernelContainer(context *cli.Context) error {
	unikernelName := context.Args().First()
	Log.Info("Delete: ", unikernelName)
	unikernel, err := GetExistingUnikernel(context)
	if err != nil {
		return err
	}
	if err := unikernel.ExecuteHooks("Poststop"); err != nil {
		return err
	}
	return unikernel.Delete()
}

func killUnikernelContainer(context *cli.Context) error {
	unikernelName := context.Args().First()
	Log.Info("Kill: ", unikernelName)
	unikernel, err := GetExistingUnikernel(context)
	if err != nil {
		Log.WithError(err).Error("Failed to GetExistingUnikernel for kill")
		return err
	}
	if err := unikernel.ExecuteHooks("Poststop"); err != nil {
		return err
	}
	return unikernel.Kill()
}
