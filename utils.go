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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/moby/sys/mount"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/vishvananda/netns"
)

// Argument check types for the `checkArgs` function.
const (
	exactArgs = iota // Checks for an exact number of arguments.
	minArgs          // Checks for a minimum number of arguments.
	maxArgs          // Checks for a maximum number of arguments.
)

// checkArgs checks the number of arguments provided in the command-line context
// against the expected number, based on the specified checkType.
func checkArgs(context *cli.Context, expected, checkType int) error {
	var err error
	cmdName := context.Command.Name

	switch checkType {
	case exactArgs:
		if context.NArg() != expected {
			err = fmt.Errorf("%s: %q requires exactly %d argument(s)", os.Args[0], cmdName, expected)
		}
	case minArgs:
		if context.NArg() < expected {
			err = fmt.Errorf("%s: %q requires a minimum of %d argument(s)", os.Args[0], cmdName, expected)
		}
	case maxArgs:
		if context.NArg() > expected {
			err = fmt.Errorf("%s: %q requires a maximum of %d argument(s)", os.Args[0], cmdName, expected)
		}
	}

	if err != nil {
		fmt.Printf("Incorrect Usage.\n\n")
		_ = cli.ShowCommandHelp(context, cmdName)
		return err
	}
	return nil
}

func logrusToStderr() bool {
	l, ok := logrus.StandardLogger().Out.(*os.File)
	return ok && l.Fd() == os.Stderr.Fd()
}

// fatal prints the error's details if it is a libcontainer specific error type
// then exits the program with an exit status of 1.
func fatal(err error) {
	fatalWithCode(err, 1)
}

func fatalWithCode(err error, ret int) {
	// Make sure the error is written to the logger.
	logrus.Error(err)
	if !logrusToStderr() {
		fmt.Fprintln(os.Stderr, err)
	}

	os.Exit(ret)
}

// LoadSpec returns the Spec found in the given bundle directory
func LoadSpec(bundleDir string) (*specs.Spec, error) {
	var spec specs.Spec
	absDir, err := filepath.Abs(bundleDir)
	if err != nil {
		return &spec, fmt.Errorf("failed to find absolute bundle path: %w", err)
	}
	data, err := os.ReadFile(filepath.Join(absDir, "config.json"))
	if err != nil {
		return &spec, fmt.Errorf("failed to read config.json: %w", err)
	}
	if err := json.Unmarshal(data, &spec); err != nil {
		return &spec, fmt.Errorf("failed to parse config.json: %w", err)
	}
	return &spec, nil
}

// reviseRootDir ensures that the --root option argument,
// if specified, is converted to an absolute and cleaned path,
// and that this path is sane.
func reviseRootDir(context *cli.Context) error {
	if !context.IsSet("root") {
		return nil
	}
	root, err := filepath.Abs(context.GlobalString("root"))
	if err != nil {
		return err
	}
	if root == "/" {
		// This can happen if --root argument is
		//  - "" (i.e. empty);
		//  - "." (and the CWD is /);
		//  - "../../.." (enough to get to /);
		//  - "/" (the actual /).
		return errors.New("option --root argument should not be set to /")
	}

	return context.GlobalSet("root", root)
}

func handleNonBimaContainer(context *cli.Context) error {
	bundleID := context.Args().First()
	root := context.GlobalString("root")
	ctrNamespace := filepath.Base(root)
	logFile := context.GlobalString("log")
	command := context.Command.FullName()
	args := context.Args().Tail()
	args = append([]string{bundleID}, args...)

	Log.WithFields(logrus.Fields{
		"cli root":     root,
		"cli log file": logFile,
		"cli command":  command,
		"cli args":     args,
		"namespace":    ctrNamespace,
	}).Info("CLI CONTEXT")

	bundle := filepath.Join("/run/containerd/io.containerd.runtime.v2.task/", ctrNamespace, bundleID)

	spec, err := LoadSpec(bundle)
	if err != nil {
		Log.WithError(err).WithField("bundle", bundle).Error("Couldn't load spec from bundle")
		return err
	}

	_, err = GetUnikernelConfig(bundle, spec)
	// this means no annotations or urunc.json was present
	if err != nil {
		Log.Info("This is a non-urunc container, executing with runc")
		return ReexecWithRunc()
	}
	Log.Info("This is a bima container! Proceeding...")
	return nil
}

func ReexecWithRunc() error {
	args := os.Args
	binPath, err := exec.LookPath("runc")
	if err != nil {
		Log.WithError(err).Error("Failed to find runc exexcutable")

	}
	args[0] = binPath
	env := os.Environ()
	err = syscall.Exec(args[0], args, env) //nolint: gosec
	if err != nil {
		Log.WithError(err).Error("Failed to execve ")
	}
	return err
}

func createPidFile(path string, pid int) error {
	var (
		tmpDir  = filepath.Dir(path)
		tmpName = filepath.Join(tmpDir, "."+filepath.Base(path))
	)
	f, err := os.OpenFile(tmpName, os.O_RDWR|os.O_CREATE|os.O_EXCL|os.O_SYNC, 0o666)
	if err != nil {
		return err
	}
	_, err = f.WriteString(strconv.Itoa(pid))
	f.Close()
	if err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func ensureValidSockAddr(sockAddr string, mustExist bool) error {
	Log.WithFields(logrus.Fields{"socket": sockAddr, "mustExist": mustExist}).Info("Checking for socket")
	if sockAddr == "" {
		Log.Error("socket address ", sockAddr, " is empty")
		return fmt.Errorf("socket address '%s' is empty", sockAddr)
	}
	if len(sockAddr) > 108 {
		Log.Error("socket address ", sockAddr, " is too long")
		return fmt.Errorf("socket address '%s' is too long", sockAddr)
	}

	if _, err := os.Stat(sockAddr); mustExist && errors.Is(err, fs.ErrNotExist) {
		Log.WithField("addr: ", sockAddr).Error("Path for sockAddr not exists")

		dir := strings.ReplaceAll(sockAddr, "urunc.sock", "")
		files, err := os.ReadDir(dir)
		if err != nil {
			Log.Error(err.Error())
		}

		Log.Info("Found ", len(files), "files in ", dir)
		for _, file := range files {
			Log.Info(file.Name(), " : ", file.IsDir())
		}
		return fmt.Errorf("socket address '%s' does not exist", sockAddr)
	}
	return nil
}

func awaitMessage(conn net.Conn, expectedMessage string) error {
	Log.Info("Awaiting for message")
	buf := make([]byte, len(expectedMessage))
	n, err := conn.Read(buf)
	if err != nil {
		return fmt.Errorf("failed to read from socket: %w", err)
	}

	msg := string(buf[0:n])
	if msg != expectedMessage {
		return fmt.Errorf("received unexpected message: %s", msg)
	}

	return nil
}

func sendMessage(conn net.Conn, message string) error {
	// TODO: Verify no race condition without sleep and remove
	// time.Sleep(2 * time.Second)
	Log.Info("Send message with sleep")
	if _, err := conn.Write([]byte(message)); err != nil {
		return fmt.Errorf("failed to send message '%s': %w", message, err)
	}

	return nil
}

func NetInterfaces() {
	// Find all network interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		Log.Error("Error finding network interfaces:", err)
		return
	}

	// Print the details of each interface
	for _, i := range interfaces {
		Log.Info("Interface: ", i.Name)

		// Find the addresses of this interface
		addrs, err := i.Addrs()
		if err != nil {
			Log.Error("Error finding addresses for interface:", err)
			continue
		}

		// Print the addresses
		for _, addr := range addrs {
			Log.Info("  Address: ", addr)
		}
	}
}

func NetnsInfo() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Save the current network namespace
	origns, _ := netns.Get()
	// defer origns.Close()
	Log.Info("original netns is: ", origns.String())
	ifaces, _ := net.Interfaces()
	info := fmt.Sprintf("original netns interfaces: %v\n", ifaces)
	Log.Info(info)
}

func ListNetNs() ([]string, error) {
	// Get a list of all files in the "/var/run/netns" directory
	files, err := os.ReadDir("/var/run/netns")
	if err != nil {
		return nil, err
	}

	// Create a slice to store the names of the network namespaces
	namespaces := make([]string, 0, len(files))

	// Add the names of the network namespaces to the slice
	for _, file := range files {
		namespaces = append(namespaces, file.Name())
	}

	return namespaces, nil
}

func CopyFile(sourceFile string, targetDir string) error {
	// Open the source file
	source, err := os.Open(sourceFile)
	if err != nil {
		return err
	}
	defer source.Close()

	// Create the target directory if it doesn't exist
	err = os.MkdirAll(targetDir, 0755)
	if err != nil {
		return err
	}

	// Extract the filename from the source file path
	_, filename := filepath.Split(sourceFile)

	// Create the target file
	targetPath := filepath.Join(targetDir, filename)
	target, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer target.Close()

	// Copy the contents of the source file to the target file
	_, err = io.Copy(target, source)
	if err != nil {
		return err
	}
	return nil
}

func DeleteFile(filename string) error {
	// Use the os.Remove function to delete the file
	err := os.Remove(filename)
	if err != nil {
		return err
	}
	return nil
}

func MoveFile(sourceFile string, targetDir string) error {
	err := CopyFile(sourceFile, targetDir)
	if err != nil {
		return err
	}
	err = DeleteFile(sourceFile)
	if err != nil {
		return err
	}
	return nil
}

func UnmountBlockDevice(devicePath string) error {
	// Call the umount system call to unmount the block device
	err := mount.Unmount(devicePath)
	if err != nil {
		return fmt.Errorf("failed to unmount block device %s: %v", devicePath, err)
	}

	return nil
}

func getInitPid(filePath string) (float64, error) {
	// Open the JSON file for reading
	file, err := os.Open(filePath)
	if err != nil {
		return 0, nil
	}
	defer file.Close()

	// Decode the JSON data into a map[string]interface{}
	var jsonData map[string]interface{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&jsonData); err != nil {
		return 0, nil

	}

	// Extract the specific value "init_process_pid"
	initProcessPID, found := jsonData["init_process_pid"].(float64) // Assuming it's a numeric value
	if !found {
		return 0, nil
	}

	return initProcessPID, nil
}
