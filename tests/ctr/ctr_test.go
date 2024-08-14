package uruncE2ETesting

import (
	"testing"
)

type testMethod func(testSpecificArgs) error

var matchTest testMethod = nil

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
			err = runTest(tc)
			if err != nil {
				t.Fatal(err.Error())
			}
		})
	}
}
