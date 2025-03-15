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
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var startCommand = cli.Command{
	Name:  "start",
	Usage: "executes the user defined process in a created container",
	ArgsUsage: `<container-id>

Where "<container-id>" is your name for the instance of the container that you
are starting. The name you provide for the container instance must be unique on
your host.`,
	Description: `The start command executes the user defined process in a created container.`,
	Action: func(context *cli.Context) error {
		// FIXME: Remove or change level of log
		logrus.WithField("args", os.Args).Info("urunc INVOKED")
		if err := checkArgs(context, 1, exactArgs); err != nil {
			return err
		}
		return startUnikontainer(context)
	},
}

// We keep it as a separate function, since it is also called from
// the run command
func startUnikontainer(context *cli.Context) error {
	// No need to check if containerID is valid, because it will get
	// checked later. We just want it for the metrics
	containerID := context.Args().First()
	metrics.Capture(containerID, "TS11")

	// get Unikontainer data from state.json
	unikontainer, err := getUnikontainer(context)
	if err != nil {
		return err
	}
	metrics.Capture(containerID, "TS12")

	err = unikontainer.SendStartExecve()
	if err != nil {
		return err
	}
	metrics.Capture(containerID, "TS13")

	return unikontainer.ExecuteHooks("Poststart")
}
