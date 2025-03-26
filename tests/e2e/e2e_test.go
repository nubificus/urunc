// Copyright (c) 2023-2025, Nubificus LTD
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

package urunce2etesting

import (
	"testing"
)

func TestNerdctl(t *testing.T) {
	tests := []containerTestArgs{
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/hello-hvt-rumprun:latest",
			Name:           "Hvt-rumprun-capture-hello",
			Devmapper:      true,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			ExpectOut:      "Hello world",
			TestFunc:       matchTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/redis-hvt-rumprun:latest",
			Name:           "Hvt-rumprun-ping-redis",
			Devmapper:      true,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       pingTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/redis-hvt-rumprun-block:latest",
			Name:           "Hvt-rumprun-ping-redis-with-block",
			Devmapper:      true,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       pingTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/net-hvt-mirage:latest",
			Name:           "Hvt-mirage-net",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       pingTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/redis-hvt-rumprun:latest",
			Name:           "Hvt-rumprun-with-seccomp",
			Devmapper:      true,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       seccompTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/redis-hvt-rumprun:latest",
			Name:           "Hvt-rumprun-without-seccomp",
			Devmapper:      true,
			Seccomp:        false,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       seccompTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/redis-spt-rumprun:latest",
			Name:           "Spt-rumprun-ping-redis",
			Devmapper:      true,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       pingTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/net-spt-mirage:latest",
			Name:           "Spt-mirage-net",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       pingTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/redis-qemu-unikraft-initrd:latest",
			Name:           "Qemu-unikraft-ping-redis",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       pingTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/nginx-qemu-unikraft-initrd:latest",
			Name:           "Qemu-unikraft-ping-nginx",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       pingTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/net-qemu-mirage:latest",
			Name:           "Qemu-mirage-net",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       pingTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/redis-qemu-unikraft-initrd:latest",
			Name:           "Qemu-unikraft-with-seccomp",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       seccompTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/redis-qemu-unikraft-initrd:latest",
			Name:           "Qemu-unikraft-without-seccomp",
			Devmapper:      false,
			Seccomp:        false,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       seccompTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/nginx-firecracker-unikraft-initrd:latest",
			Name:           "Firecracker-unikraft-ping-nginx",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       pingTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/nginx-firecracker-unikraft-initrd:latest",
			Name:           "Firecracker-unikraft-with-seccomp",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       seccompTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/nginx-firecracker-unikraft-initrd:latest",
			Name:           "Firecracker-unikraft-without-seccomp",
			Devmapper:      false,
			Seccomp:        false,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       seccompTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/nginx-firecracker-unikraft-initrd:latest",
			Name:           "Firecracker-unikraft-namespaces",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       namespaceTest,
		},
	}
	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			nerdctlTool := newNerdctlTool(tc)
			runTest(nerdctlTool, t)
		})
	}
}

func TestCtr(t *testing.T) {
	tests := []containerTestArgs{
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/hello-hvt-rumprun-nonet:latest",
			Name:           "Hvt-rumprun-hello",
			Devmapper:      true,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			ExpectOut:      "Hello world",
			TestFunc:       matchTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/hello-hvt-mirage:latest",
			Name:           "Hvt-mirage-hello",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			ExpectOut:      "Hola!",
			TestFunc:       matchTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/block-test-hvt-mirage:latest",
			Name:           "Hvt-mirage-block-test",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			ExpectOut:      "Total tests passed:  10",
			TestFunc:       matchTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/hello-spt-rumprun-nonet:latest",
			Name:           "Spt-rumprun-hello",
			Devmapper:      true,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			ExpectOut:      "Hello world",
			TestFunc:       matchTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/hello-spt-mirage:latest",
			Name:           "Spt-mirage-hello",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			ExpectOut:      "Hola!",
			TestFunc:       matchTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/block-test-spt-mirage:latest",
			Name:           "Spt-mirage-block-test",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			ExpectOut:      "Total tests passed:  10",
			TestFunc:       matchTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/hello-qemu-unikraft:latest",
			Name:           "Qemu-unikraft-hello",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			ExpectOut:      "\"Urunc\" \"Unikraft\" \"Qemu\"",
			TestFunc:       matchTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/hello-firecracker-unikraft:latest",
			Name:           "Firecracker-unikraft-hello",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			ExpectOut:      "\"Urunc\" \"Unikraft\" \"FC\"",
			TestFunc:       matchTest,
		},
	}
	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			ctrTool := newCtrTool(tc)
			runTest(ctrTool, t)
		})
	}
}

func TestCrictl(t *testing.T) {
	tests := []containerTestArgs{
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/redis-hvt-rumprun:latest",
			Name:           "Hvt-rumptun-redis",
			Devmapper:      true,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       pingTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/net-hvt-mirage:latest",
			Name:           "Hvt-mirage-net",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       pingTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/redis-spt-rumprun:latest",
			Name:           "Spt-rumptun-redis",
			Devmapper:      true,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       pingTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/net-spt-mirage:latest",
			Name:           "Spt-mirage-net",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       pingTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/redis-qemu-unikraft-initrd:latest",
			Name:           "Qemu-unikraft-redis",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       pingTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/net-qemu-mirage:latest",
			Name:           "Qemu-mirage-net",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       pingTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/nginx-firecracker-unikraft:latest",
			Name:           "Firecracker-unikraft-nginx",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       pingTest,
		},
		// TODO: We need to rewrite this test
		// Ideally we want to spawn a knative queue-proxy and then the unikernel
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/httpreply-firecracker-unikraft:latest",
			Name:           "Firecracker-unikraft-httpreply-static-net",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      true,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       httpStaticNetTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/nginx-firecracker-unikraft-initrd:latest",
			Name:           "Firecracker-unikraft-namespaces",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       namespaceTest,
		},
	}
	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			crictlTool := newCrictlTool(tc)
			runTest(crictlTool, t)
		})
	}
}

func TestDocker(t *testing.T) {
	tests := []containerTestArgs{
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/net-spt-mirage:latest",
			Name:           "Spt-mirage-net",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       pingTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/redis-qemu-unikraft-initrd:latest",
			Name:           "Qemu-unikraft-ping-redis",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       pingTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/nginx-qemu-unikraft-initrd:latest",
			Name:           "Qemu-unikraft-ping-nginx",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       pingTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/net-qemu-mirage:latest",
			Name:           "Qemu-mirage-net",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       pingTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/redis-qemu-unikraft-initrd:latest",
			Name:           "Qemu-unikraft-with-seccomp",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       seccompTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/redis-qemu-unikraft-initrd:latest",
			Name:           "Qemu-unikraft-without-seccomp",
			Devmapper:      false,
			Seccomp:        false,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       seccompTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/nginx-firecracker-unikraft-initrd:latest",
			Name:           "Firecracker-unikraft-ping-nginx",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       pingTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/nginx-firecracker-unikraft-initrd:latest",
			Name:           "Firecracker-unikraft-with-seccomp",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       seccompTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/nginx-firecracker-unikraft-initrd:latest",
			Name:           "Firecracker-unikraft-without-seccomp",
			Devmapper:      false,
			Seccomp:        false,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       seccompTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/nginx-firecracker-unikraft-initrd:latest",
			Name:           "Firecracker-unikraft-namespaces",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestFunc:       namespaceTest,
		},
	}
	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			dockerTool := newDockerTool(tc)
			runTest(dockerTool, t)
		})
	}
}
