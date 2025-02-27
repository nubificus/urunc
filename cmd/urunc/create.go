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
	"io"
	"net"
	"os"
	"os/exec"
	"syscall"
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

	// NOTE: We might want to switch form pipe to socketpair for logs too.
	logPipeParent, logPipeChild, err := os.Pipe()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create pipe for logs")
	}
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
	reexecCommand.ExtraFiles =  append(reexecCommand.ExtraFiles, initSockChild)
	reexecCommand.ExtraFiles =  append(reexecCommand.ExtraFiles, logPipeChild)
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

	logsDone := logs.ForwardLogs(logPipeParent)
	nsenterInfo, err := unikontainer.FormatNsenterComm()
	if err != nil {
		logrus.WithError(err).Fatal("failed to format data for nsenter")
	}

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

	err = initSockChild.Close()
	if err != nil {
		logrus.WithError(err).Errorf("failed to close child socket pair")
	}
	err = logPipeChild.Close()
	if err != nil {
		logrus.WithError(err).Errorf("failed to close child socket pair")
	}

	if nsenterInfo != nil {
		wbytes, err := io.Copy(initSockParent, nsenterInfo)
		logrus.Info("Wrote ", wbytes)
		if err != nil {
			return fmt.Errorf("error copying nsenter info to pipe: %w", err)
		}
	}

	//data, _ := io.ReadAll(initSockParent) // Read raw data
	//decoder := json.NewDecoder(bytes.NewReader(data))
	decoder := json.NewDecoder(initSockParent)
	decoder.DisallowUnknownFields()
	var pid struct {
		Stage2Pid int `json:"stage2_pid"`
		Stage1Pid int `json:"stage1_pid"`
	}
	if err := decoder.Decode(&pid); err != nil {
		return fmt.Errorf("error reading pid from init pipe: %w", err)
	}

	// Clean up the zombie parent process
	Stage1Process, _ := os.FindProcess(pid.Stage1Pid)
	// Ignore the error in case the child has already been reaped for any reason
	_, _ = Stage1Process.Wait()

	status, err := reexecCommand.Process.Wait()
	if err != nil {
		_ = reexecCommand.Wait()
		return fmt.Errorf("nsenter error: %w", err)
	}
	if !status.Success() {
		_ = reexecCommand.Wait()
		return fmt.Errorf("nsenter unsuccessful exit: %w", err)
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

	// Wait for reexec process to notify us
	err = unikontainers.AwaitMessage(listener, unikontainers.ReexecStarted)
	if err != nil {
		return err
	}
	metrics.Capture(containerID, "TS07")

	// Retrieve reexec cmd's pid and write to file and state
	containerPid := pid.Stage2Pid
	err = unikontainer.Create(containerPid)
	//pid := reexecCommand.Process.Pid
	//err = unikontainer.Create(pid)
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

	// get Unikontainer data from state.json
	// Reload state in order to get the pid written from urunc create
	// TODO: We need to find a better way to synchronize and make sure
	// the pid is written from urunc` create.
	unikontainer, err = getUnikontainer(context)
	if err != nil {
		return err
	}

	// wait StartExecve message on urunc.sock from urunc start process
	err = unikontainer.ListenAndAwaitMsg(socketPath, unikontainers.StartExecve)
	if err != nil {
		return err
	}
	metrics.Capture(containerID, "TS15")

	// execute Prestart hooks
	err = unikontainer.ExecuteHooks("Prestart")
	if err != nil {
		return err
	}

	// execve
	return unikontainer.Exec()
}

// newSockPair returns a new SOCK_STREAM unix socket pair.
func newSockPair(name string) (parent, child *os.File, err error) {
	fds, err := unix.Socketpair(unix.AF_LOCAL, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return nil, nil, err
	}
	return os.NewFile(uintptr(fds[1]), name+"-p"), os.NewFile(uintptr(fds[0]), name+"-c"), nil
}
