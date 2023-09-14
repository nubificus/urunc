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

package constants

const (
	// StateCreating indicates that the container is being created.
	StateCreating string = "creating"
	// StateCreated indicates that the runtime has finished the create operation.
	StateCreated string = "created"
	// StateRunning indicates that the container process has executed the
	// user-specified program but has not exited.
	StateRunning string = "running"
	// StateStopped indicates that the container process has exited.
	StateStopped string = "stopped"
)
