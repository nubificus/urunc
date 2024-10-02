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

var deleteCommand = cli.Command{
	Name:  "delete",
	Usage: "delete any resources held by the container often used with detached container",
	ArgsUsage: `<container-id>

Where "<container-id>" is the name for the instance of the container.

EXAMPLE:
For example, if the container id is "ubuntu01" and runc list currently shows the
status of "ubuntu01" as "stopped" the following will delete resources held for
"ubuntu01" removing "ubuntu01" from the runc list of containers:

       # runc delete ubuntu01`,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "force, f",
			Usage: "Forcibly deletes the container if it is still running (uses SIGKILL)",
		},
	},
	Action: func(context *cli.Context) error {
		logrus.WithField("args", os.Args).Info("urunc INVOKED")
		if err := checkArgs(context, 1, exactArgs); err != nil {
			return err
		}

		// get Unikontainer data from state.json
		unikontainer, err := getUnikontainer(context)
		if err != nil {
			return err
		}
		if context.Bool("force") {
			err := unikontainer.Kill()
			if err != nil {
				return err
			}
		}
		return unikontainer.Delete()
	},
}
