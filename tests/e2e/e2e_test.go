package uruncE2ETesting

import (
	"testing"
)

type testMethod func(testSpecificArgs) error

type containerTestArgs struct {
	Name string
	Image string
	Devmapper bool
	Seccomp bool
	Skippable bool
	TestFunc testMethod
	TestArgs testSpecificArgs
}

type testSpecificArgs struct {
	ContainerID string
	Seccomp bool
	Expected string
}

//func TestsWithNerdctl(t *testing.T) {
func TestNerdctl(t *testing.T) {
	tests := []containerTestArgs {
		{
			Image : "harbor.nbfc.io/nubificus/urunc/hello-hvt-rump:latest",
			Name : "hvt-rumprun-capture-hello",
			Devmapper : true,
			Seccomp : true,
			Skippable: false,
			TestArgs : testSpecificArgs {
				Expected : "Hello world",
			},
			TestFunc: matchTest,
		},
		{
			Image : "harbor.nbfc.io/nubificus/urunc/redis-hvt-rump:latest",
			Name : "hvt-rumprun-ping-redis",
			Devmapper : true,
			Seccomp : true,
			Skippable: false,
			TestFunc: pingTest,
		},
		{
			Image : "harbor.nbfc.io/nubificus/urunc/redis-hvt-rump:latest",
			Name : "hvt-rumprun-with-seccomp",
			Devmapper : true,
			Seccomp : true,
			Skippable: false,
			TestFunc: seccompTest,
		},
		{
			Image : "harbor.nbfc.io/nubificus/urunc/redis-hvt-rump:latest",
			Name : "hvt-rumprun-without-seccomp",
			Devmapper : true,
			Seccomp : false,
			Skippable: false,
			TestFunc: seccompTest,
		},
		{
			Image : "harbor.nbfc.io/nubificus/urunc/redis-spt-rump:latest",
			Name : "spt-rumprun-ping-redis",
			Devmapper : true,
			Seccomp : true,
			Skippable: false,
			TestFunc: pingTest,
		},
		{
			Image : "harbor.nbfc.io/nubificus/urunc/redis-qemu-unikraft-initrd:latest",
			Name : "qemu-unikraft-ping-redis",
			Devmapper : false,
			Seccomp : true,
			Skippable: false,
			TestFunc: pingTest,
		},
		{
			Image : "harbor.nbfc.io/nubificus/urunc/nginx-qemu-unikraft:latest",
			Name : "qemu-unikraft-ping-nginx",
			Devmapper : false,
			Seccomp : true,
			Skippable: false,
			TestFunc: pingTest,
		},
		{
			Image : "harbor.nbfc.io/nubificus/urunc/redis-qemu-unikraft-initrd:latest",
			Name : "qemu-unikraft-with-seccomp",
			Devmapper : false,
			Seccomp : true,
			Skippable: false,
			TestFunc: seccompTest,
		},
		{
			Image : "harbor.nbfc.io/nubificus/urunc/redis-qemu-unikraft-initrd:latest",
			Name : "qemu-unikraft-without-seccomp",
			Devmapper : false,
			Seccomp : false,
			Skippable: false,
			TestFunc: seccompTest,
		},
		{
			Image : "harbor.nbfc.io/nubificus/urunc/nginx-fc-unik:latest",
			Name : "fc-unikraft-ping-nginx",
			Devmapper : false,
			Seccomp : true,
			Skippable: false,
			TestFunc: pingTest,
		},
		{
			Image : "harbor.nbfc.io/nubificus/urunc/nginx-fc-unik:latest",
			Name : "fc-unikraft-with-seccomp",
			Devmapper : false,
			Seccomp : true,
			Skippable: false,
			TestFunc: seccompTest,
		},
		{
			Image : "harbor.nbfc.io/nubificus/urunc/nginx-fc-unik:latest",
			Name : "fc-unikraft-without-seccomp",
			Devmapper : false,
			Seccomp : false,
			Skippable: false,
			TestFunc: seccompTest,
		},
	}
	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			err := nerdctlRunTest(tc)
			if err != nil {
				t.Fatal(err.Error())
			}
		})
	}
}

//func TestsWithCtr(t *testing.T) {
func TestCtr(t *testing.T) {
	tests := []containerTestArgs {
		{
			Image : "harbor.nbfc.io/nubificus/urunc/hello-hvt-nonet:latest",
			Name : "hvt-rumprun-hello",
			Devmapper : true,
			Seccomp : true,
			Skippable: false,
			TestArgs : testSpecificArgs {
				Expected : "Hello world",
			},
			TestFunc: matchTest,
		},
		{
			Image : "harbor.nbfc.io/nubificus/urunc/hello-spt-nonet:latest",
			Name : "spt-rumprun-hello",
			Devmapper : true,
			Seccomp : true,
			Skippable: false,
			TestArgs : testSpecificArgs {
				Expected : "Hello world",
			},
			TestFunc: matchTest,
		},
		{
			Image : "harbor.nbfc.io/nubificus/urunc/qemu-unikraft-hello:latest",
			Name : "qemu-unikraft-hello",
			Devmapper : false,
			Seccomp : true,
			Skippable: false,
			TestArgs : testSpecificArgs {
				Expected : "\"Urunc\" \"Unikraft\" \"Qemu\"",
			},
			TestFunc: matchTest,
		},
		{
			Image : "harbor.nbfc.io/nubificus/urunc/fc-unikraft-hello:latest",
			Name : "fc-unikraft-hello",
			Devmapper : false,
			Seccomp : true,
			Skippable: false,
			TestArgs : testSpecificArgs {
				Expected : "\"Urunc\" \"Unikraft\" \"FC\"",
			},
			TestFunc: matchTest,
		},
	}
	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			err := pullImage(tc.Image)
			if err != nil {
				t.Fatal(err.Error())
			}
			err = ctrRunTest(tc)
			if err != nil {
				t.Fatal(err.Error())
			}
		})
	}
}
