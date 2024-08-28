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
	"context"
	"log"
	"log/syslog"

	"github.com/containerd/containerd/runtime/v2/runc/manager"
	_ "github.com/containerd/containerd/runtime/v2/runc/task/plugin"
	"github.com/containerd/containerd/runtime/v2/shim"
)

func main() {
	sysLogger, err := syslog.New(syslog.LOG_INFO|syslog.LOG_USER, "containerd-shim-urunc")
	if err != nil {
		log.Fatal("Failed to connect to syslog:", err)
	}
	log.SetOutput(sysLogger)
	log.Println("containerd-shim-urunc CALLED")
	sysLogger.Close()

	shim.RunManager(context.Background(), manager.NewShimManager("io.containerd.urunc.v2"))
}
