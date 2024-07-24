// Copyright 2024 Nubificus LTD.

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
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetSockAddr(t *testing.T) {
	dir := "/tmp"
	name := "test.sock"
	expected := filepath.Join(dir, name)
	result := getSockAddr(dir, name)
	assert.Equal(t, expected, result, "Expected %s, but got %s", expected, result)
}

func TestGetInitSockAddr(t *testing.T) {
	containerDir := "/tmp/container"
	expected := filepath.Join(containerDir, initSock)
	result := getInitSockAddr(containerDir)
	assert.Equal(t, expected, result, "Expected %s, but got %s", expected, result)
}

func TestGetUruncSockAddr(t *testing.T) {
	containerDir := "/tmp/container"
	expected := filepath.Join(containerDir, uruncSock)
	result := getUruncSockAddr(containerDir)
	assert.Equal(t, expected, result, "Expected %s, but got %s", expected, result)
}

func TestEnsureValidSockAddr(t *testing.T) {
	validSockAddr := "/tmp/valid.sock"
	emptySockAddr := ""
	longSockAddr := string(make([]byte, 109))

	assert.NoError(t, ensureValidSockAddr(validSockAddr), "Expected no error for valid socket address")
	assert.Error(t, ensureValidSockAddr(emptySockAddr), "Expected error for empty socket address")
	assert.Error(t, ensureValidSockAddr(longSockAddr), "Expected error for long socket address")
}

func TestSockAddrExists(t *testing.T) {
	existingSockAddr := "/tmp/existing.sock"
	nonExistingSockAddr := "/tmp/non_existing.sock"

	// Create a temporary socket file
	f, err := os.Create(existingSockAddr)
	if err != nil {
		t.Fatalf("Failed to create temporary socket file: %v", err)
	}
	defer os.Remove(existingSockAddr)
	f.Close()

	assert.True(t, SockAddrExists(existingSockAddr), "Expected socket address to exist")
	assert.False(t, SockAddrExists(nonExistingSockAddr), "Expected socket address to not exist")
}

func TestSendIPCMessage(t *testing.T) {
	socketAddress := "/tmp/test.sock"
	message := ReexecStarted

	listener, err := net.Listen("unix", socketAddress)
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer os.Remove(socketAddress)
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			t.Errorf("Failed to accept connection: %v", err)
		}
		defer conn.Close()

		buf := make([]byte, len(message))
		n, err := conn.Read(buf)
		if err != nil {
			t.Errorf("Failed to read message: %v", err)
		}

		assert.Equal(t, string(message), string(buf[:n]), "Expected %s, but got %s", message, string(buf[:n]))
	}()

	time.Sleep(10 * time.Millisecond) // Ensure listener is ready

	err = SendIPCMessage(socketAddress, message)
	assert.NoError(t, err, "Expected no error in sending IPC message")
}

func TestSendIPCMessageWithRetry(t *testing.T) {
	socketAddress := "/tmp/test_retry.sock"
	message := ReexecStarted

	listener, err := net.Listen("unix", socketAddress)
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer os.Remove(socketAddress)
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			t.Errorf("Failed to accept connection: %v", err)
		}
		defer conn.Close()

		buf := make([]byte, len(message))
		n, err := conn.Read(buf)
		if err != nil {
			t.Errorf("Failed to read message: %v", err)
		}

		assert.Equal(t, string(message), string(buf[:n]), "Expected %s, but got %s", message, string(buf[:n]))
	}()

	err = sendIPCMessageWithRetry(socketAddress, message, true)
	assert.NoError(t, err, "Expected no error in sending IPC message with retry")
}

func TestCreateListener(t *testing.T) {
	socketAddress := "/tmp/test_create_listener.sock"

	listener, err := CreateListener(socketAddress, true)
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer os.Remove(socketAddress)
	defer listener.Close()

	assert.NotNil(t, listener, "Expected listener to be created")
}

func TestAwaitMessage(t *testing.T) {
	socketAddress := "/tmp/test_await_message.sock"
	expectedMessage := ReexecStarted

	listener, err := CreateListener(socketAddress, true)
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer os.Remove(socketAddress)
	defer listener.Close()

	go func() {
		conn, err := net.Dial("unix", socketAddress)
		if err != nil {
			t.Errorf("Failed to dial connection: %v", err)
		}
		defer conn.Close()

		_, err = conn.Write([]byte(expectedMessage))
		if err != nil {
			t.Errorf("Failed to send message: %v", err)
		}
	}()

	err = AwaitMessage(listener, expectedMessage)
	assert.NoError(t, err, "Expected no error in awaiting message")
}
