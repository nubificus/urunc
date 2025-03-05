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
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"

	"github.com/creack/pty"
	"github.com/nubificus/urunc/pkg/unikontainers"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"golang.org/x/sys/unix"
	"github.com/opencontainers/runc/libcontainer/logs"
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
func createUnikontainer(context *cli.Context) (retErr error) {
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

	// Create socket for nsenter
	initSockParent, initSockChild, err := newSockPair("init")
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create init sockpair")
	}
	defer func() {
		err = initSockParent.Close()
		if err != nil {
			logrus.WithError(err).Errorf("failed to close parent socket pair")
		}
	}()

	// Create log pipe for nsenter
	// NOTE: We might want to switch form pipe to socketpair for logs too.
	logPipeParent, logPipeChild, err := os.Pipe()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create pipe for logs")
	}

	// get the data to send to nsenter
	nsenterInfo, err := unikontainer.FormatNsenterComm()
	if err != nil {
		logrus.WithError(err).Fatal("failed to format data for nsenter")
	}

	// Setup reexecCommand
	reexecCommand := createReexecCmd(initSockChild, logPipeChild)

	// Create a go func for logs from nsenter
	logsDone := logs.ForwardLogs(logPipeParent)

	// Start reexec process
	metrics.Capture(containerID, "TS03")
	// setup terminal if required and start reexec process
	// TODO: This part of code needs bette rhandling. It is not the
	// job of the urunc create to setup the terminal for reexec.
	// The main concern is the nsenter execution before the reexec.
	// We do not want in any scenario to mess up with nsenter.
	// Debugging is hell.
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

	// CLose child ends of sockets and pipes.
	err = initSockChild.Close()
	if err != nil {
		logrus.WithError(err).Errorf("failed to close child socket pair")
	}
	err = logPipeChild.Close()
	if err != nil {
		logrus.WithError(err).Errorf("failed to close child socket pair")
	}

	// Send data to nsenter 
	_, err = io.Copy(initSockParent, nsenterInfo)
	if err != nil {
		return fmt.Errorf("error copying nsenter info to socket: %w", err)
	}

	// Get pids from nsenter and reap dead children
	reexecPid, err := handleNsenterRet(initSockParent, reexecCommand)
	if err != nil {
		return err
	}

	if logsDone != nil {
		defer func() {
			// Wait for log forwarder to finish. This depends on
			// reexec closing the _LIBCONTAINER_LOGPIPE log fd.
			err := <-logsDone
			if err != nil && retErr == nil {
				retErr = fmt.Errorf("unable to forward init logs: %w", err)
			}
		}()
	}

	// Retrieve reexec cmd's pid and write to file and state
	containerPid := reexecPid
	metrics.Capture(containerID, "TS06")

	err = unikontainer.Create(containerPid)
	if err != nil {
		return err
	}

	// execute CreateRuntime hooks
	err = unikontainer.ExecuteHooks("CreateRuntime")
	if err != nil {
		return fmt.Errorf("failed to execute CreateRuntime hooks: %w", err)
	}
	metrics.Capture(containerID, "TS07")

	// send ACK to reexec process
	err = unikontainer.SendAckReexec()
	if err != nil {
		return fmt.Errorf("failed to send ACK to reexec process: %w", err)

	}
	metrics.Capture(containerID, "TS08")

	// execute CreateRuntime hooks
	err = unikontainer.ExecuteHooks("CreateContainer")
	if err != nil {
		return fmt.Errorf("failed to execute CreateRuntime hooks: %w", err)
	}
	metrics.Capture(containerID, "TS10")

	return nil
}

func createReexecCmd(initSock *os.File, logPipe *os.File) *exec.Cmd {
	selfPath := "/proc/self/exe"
	reexecCommand := &exec.Cmd{
		Path: selfPath,
		Args: append(os.Args, "--reexec"),
		Env: os.Environ(),
	}
	// Set files that we want to pass to children. In particular,
	// we need to pass a socketpair for the communication with the nsenter
	// and a log pipe to get logs from nsenter.
	// NOTE: Currently we only pass two files to children. In the future
	// we might need to refactor the following code, in case we need to
	// pass more than just these files.
	reexecCommand.ExtraFiles =  append(reexecCommand.ExtraFiles, initSock)
	reexecCommand.ExtraFiles =  append(reexecCommand.ExtraFiles, logPipe)
	// The hardcoded value here refers to the first open file descriptor after
	// the stdio file descriptors. Therefore, since the initSockChild was the
	// first file we addedd in ExtraFiles, its file descriptor should be 2+1=3,
	// since 0 is stdin, 1 is stdout and 2 is stderr. Similarly, the logPipeChild
	// should be right after initSockChild, hence 4
	// NOTE: THis might need bette rhandling in the future.
	reexecCommand.Env = append(reexecCommand.Env, "_LIBCONTAINER_INITPIPE=3")
	reexecCommand.Env = append(reexecCommand.Env, "_LIBCONTAINER_LOGPIPE=4")
	logLevel := strconv.Itoa(int(logrus.GetLevel()))
	if logLevel != "" {
		reexecCommand.Env = append(reexecCommand.Env, "_LIBCONTAINER_LOGLEVEL="+logLevel)
	}

	return reexecCommand
}

func handleNsenterRet(initSock *os.File, reexec *exec.Cmd) (int, error) {
	var pid struct {
		Stage2Pid int `json:"stage2_pid"`
		Stage1Pid int `json:"stage1_pid"`
	}
	decoder := json.NewDecoder(initSock)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&pid); err != nil {
		return -1, fmt.Errorf("error reading pid from init pipe: %w", err)
	}

	// Clean up the zombie parent process
	Stage1Process, _ := os.FindProcess(pid.Stage1Pid)
	// Ignore the error in case the child has already been reaped for any reason
	_, _ = Stage1Process.Wait()

	status, err := reexec.Process.Wait()
	if err != nil {
		_ = reexec.Wait()
		return -1, fmt.Errorf("nsenter error: %w", err)
	}
	if !status.Success() {
		_ = reexec.Wait()
		return -1, fmt.Errorf("nsenter unsuccessful exit: %w", err)
	}

	return pid.Stage2Pid, nil
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

	logFd, err := strconv.Atoi(os.Getenv("_LIBCONTAINER_LOGPIPE"))
	if err != nil {
		return fmt.Errorf("unable to convert _LIBCONTAINER_LOGPIPE: %w", err)
	}
	logPipe := os.NewFile(uintptr(logFd), "logpipe")
	err = logPipe.Close()
	if err != nil {
		return fmt.Errorf("close log pipe: %w", err)
	}
	initFd, err := strconv.Atoi(os.Getenv("_LIBCONTAINER_INITPIPE"))
	if err != nil {
		return fmt.Errorf("unable to convert _LIBCONTAINER_INITPIPE: %w", err)
	}
	initPipe := os.NewFile(uintptr(initFd), "initpipe")
	err = initPipe.Close()
	if err != nil {
		return fmt.Errorf("close init pipe: %w", err)
	}

	// We have already made sure in main.go that root is not nil
	rootDir := context.GlobalString("root")
	baseDir := filepath.Join(rootDir, containerID)

	metrics.Capture(containerID, "TS05")

	// wait AckReexec message on urunc.sock from parent process
	socketPath := unikontainers.GetUruncSockAddr(baseDir)
	err = unikontainers.ListenAndAwaitMsg(socketPath, unikontainers.AckReexec)
	if err != nil {
		return err
	}
	metrics.Capture(containerID, "TS09")

	// get Unikontainer data from state.json
	// TODO: We need to find a better way to synchronize and make sure
	// the pid is written from urunc` create.
	unikontainer, err := getUnikontainer(context)
	if err != nil {
		return err
	}

	// wait StartExecve message on urunc.sock from urunc start process
	err = unikontainers.ListenAndAwaitMsg(socketPath, unikontainers.StartExecve)
	if err != nil {
		return err
	}
	metrics.Capture(containerID, "TS14")

	// execute Prestart hooks
	err = unikontainer.ExecuteHooks("Prestart")
	if err != nil {
		return err
	}

	// execve
	return unikontainer.Exec()
}
