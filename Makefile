# Copyright (c) 2023-2024, Nubificus LTD
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Versioning variables
COMMIT         := $(shell git describe --dirty --long --always)
VERSION        := $(shell cat $(CURDIR)/VERSION)-$(COMMIT)

# Path variables
#
# Use absolute paths just for sanity.
#? BUILD_DIR Directory to place produced binaries (default: ${CWD}/dist)
BUILD_DIR      ?= ${CURDIR}/dist
VENDOR_DIR     := ${CURDIR}/vendor
#? PREFIX Directory to install urunc and shim (default: /usr/local/bin)
PREFIX         ?= /usr/local/bin

# Architecture specific variables
#
# Do not allow building of unsupported or untested host architectures
#? ARCH Target architecture (default: Host arch)
ifeq ($(origin ARCH), undefined)
    UNAME_ARCH := $(shell uname -m)
    ifeq ($(UNAME_ARCH),x86_64)
        ARCH := amd64
    else ifeq ($(UNAME_ARCH),aarch64)
        ARCH := arm64
    else
        $(error Unsupported architecture: $(UNAME_ARCH))
    endif
endif

# Binary variables
URUNC_BIN      := $(BUILD_DIR)/urunc
SHIM_BIN       := $(BUILD_DIR)/containerd-shim-urunc-v2

# Golang variables
#? GO go binary to use (default: go)
GO             ?= go
GO_FLAGS       := GOOS=linux
GO_FLAGS       += CGO_ENABLED=0
TEST_FLAGS     := "-count=1"
TEST_OPTS      += -timeout 3m

# Linking variables
LDFLAGS_COMMON := -X main.version=$(VERSION)
LDFLAGS_STATIC := --extldflags -static
LDFLAGS_OPT    := -s -w

# Source files variables
#
# Add all urunc specific go packages as dependency for building
# or the shimAny change ina go file will result to rebuilding urunc
URUNC_SRC      := $(wildcard $(CURDIR)/cmd/urunc/*.go)
URUNC_SRC      += $(wildcard $(CURDIR)/pkg/unikontainers/*.go)
URUNC_SRC      += $(wildcard $(CURDIR)/pkg/unikontainers/hypervisors/*.go)
URUNC_SRC      += $(wildcard $(CURDIR)/pkg/unikontainers/unikernels/*.go)
URUNC_SRC      += $(wildcard $(CURDIR)/pkg/network/*.go)
SHIM_SRC       := $(wildcard $(CURDIR)/cmd/containerd-shim-urunc-v2/*.go)

# Linking variables
#? LINT_CNTR_TOOL Tool to run the linter container (default: nerdctl)
LINT_CNTR_TOOL ?= nerdctl
LINT_CNTR_OPTS ?= run --rm -it -v $(CURDIR):/app -w /app
#? LINT_IMG The linter image to use (default: golangci/golangci-lint:v1.53.3)
LINT_CNTR_IMG  ?= golangci/golangci-lint:v1.53.3
LINT_CNTR_CMD  ?= golangci-lint run -v --timeout=5m

# Install dependencies variables
#
# If we have already built either static or dynamic version of urunc
# we do not have to rebuild it, but instead we can use whichever
# version is available to install it. However, the dynamic version
# has always a preference.
INSTALL_DEPS   =  $(shell test -e $(URUNC_BIN)_static_$(ARCH) \
                          && echo $(URUNC_BIN)_static_$(ARCH) && exit \
                          || test -e $(URUNC_BIN)_dynamic_$(ARCH) \
                             && echo $(URUNC_BIN)_dynamic_$(ARCH) \
                             || echo $(URUNC_BIN)_static_$(ARCH))

# Main Building rules
#
# By default we opt to build static binaries targeting the host archiotecture.
# However, we build shim as a dynamically-linked binary.

## default Build shim and urunc statically for host arch.(default).
.PHONY: default
default: static shim

## shim Build only the shim for host arch..
.PHONY: shim
shim:  $(SHIM_BIN)_$(ARCH)

## static Build urunc statically for host arch.
.PHONY: static
static: $(URUNC_BIN)_static_$(ARCH)

## dynamic Build urunc as dynamically-linked binary for host arch.
.PHONY: dynamic
dynamic: $(URUNC_BIN)_dynamic_$(ARCH)

## all Build shim and urunc statically for all amd64 and aarch64
.PHONY: all
all: $(SHIM_BIN)_arm64 $(SHIM_BIN)_amd64 $(URUNC_BIN)_static_amd64 $(URUNC_BIN)_static_arm64

# Just an alias for $(VENDOR_DIR) for easie invocation
## prepare Run go mod vendor and veridy.
prepare: $(VENDOR_DIR)

# Add tidy as order-only prerequisite. In that way, since tidy does not
# produce any file and executes all the time, we avoid the execution
# of $(VENDOR_DIR) rule, if the file already exists
$(VENDOR_DIR):
	$(GO) mod tidy
	$(GO) mod vendor
	$(GO) mod verify

# Add tidy and as order-only prerequisite. In that way, since tidy and
# vendor do notproduce any file and execute all the time,
# we avoid the rebuilding of urunc if it has previously built and the
# source files have not changed.
$(URUNC_BIN)_static_%: $(URUNC_SRC) | prepare
	$(GO_FLAGS) GOARCH=$* $(GO) build \
		-ldflags "$(LDFLAGS_COMMON) $(LDFLAGS_STATIC) $(LDFLAGS_OPT)" \
		-o $(URUNC_BIN)_static_$* $(CURDIR)/cmd/urunc

$(URUNC_BIN)_dynamic_%: $(URUNC_SRC) | prepare
	$(GO_FLAGS) GOARCH=$* $(GO) build \
		-ldflags "$(LDFLAGS_COMMON) $(LDFLAGS_OPT)" \
		-o $(URUNC_BIN)_dynamic_$* $(CURDIR)/cmd/urunc

$(SHIM_BIN)_%: $(SHIM_SRC) | prepare
	@sed -i 's/DefaultCommand = "runc"/DefaultCommand = "urunc"/g' \
		$(VENDOR_DIR)/github.com/containerd/go-runc/runc.go
	$(GO_FLAGS) GOARCH=$* $(GO) build \
		-o $(SHIM_BIN)_$* $(CURDIR)/cmd/containerd-shim-urunc-v2

## install Install urunc and shim in PREFIX
.PHONY: install
install: $(SHIM_BIN)_$(ARCH) $(INSTALL_DEPS)
	install -D -m0755 $(word 2,$^) $(PREFIX)/urunc
	install -D -m0755 $< $(PREFIX)/containerd-shim-urunc-v2

## uninstall Remove urunc and shim from PREFIX
.PHONY: uninstall
uninstall:
	rm -f $(PREFIX)/urunc
	rm -f $(PREFIX)/containerd-shim-urunc-v2

## distclean Remove build and vendor directories
.PHONY: distclean
distclean: clean
	rm -fr $(VENDOR_DIR)

## clean build directory
.PHONY: clean
clean:
	rm -fr $(BUILD_DIR)

# Linting targets
## lint Run the lint test using a golang container
.PHONY: lint
lint:
	$(LINT_CNTR_TOOL) $(LINT_CNTR_OPTS) $(LINT_CNTR_IMG) $(LINT_CNTR_CMD)

# Testing targets
## test Run all tests
.PHONY: test
test: unittest e2etest

## unittest Run all unit tests
.PHONY: unittest
unittest: test_unikontainers

## e2etest Run all end-to-end tests
.PHONY: e2etest
e2etest: test_nerdctl test_ctr test_crictl

## test_unikontainers Run unit tests for unikontainers package
test_unikontainers:
	@echo "Unit testing in unikontainers"
	@GOFLAGS=$(TEST_FLAGS) $(GO) test $(TEST_OPTS) ./pkg/unikontainers -v
	@echo " "

## test_nerdctl Run all end-to-end tests with nerdctl
.PHONY: test_nerdctl
test_nerdctl:
	@echo "Testing nerdctl"
	@GOFLAGS=$(TEST_FLAGS) $(GO) test $(TEST_OPTS) ./tests/e2e -run TestNerdctl -v
	@echo " "

## test_ctr Run all end-to-end tests with ctr
.PHONY: test_ctr
test_ctr:
	@echo "Testing ctr"
	@GOFLAGS=$(TEST_FLAGS) $(GO) test $(TEST_OPTS) ./tests/e2e -run TestCtr -v
	@echo " "

## test_crictl Run all end-to-end tests with crictl
.PHONY: test_crictl
test_crictl:
	@echo "Testing crictl"
	@GOFLAGS=$(TEST_FLAGS) $(GO) test $(TEST_OPTS) ./tests/e2e -run TestCrictl -v
	@echo " "

## test_nerdctl_[pattern] Run all end-to-end tests with nerdctl that match pattern
.PHONY: test_nerdctl_%
test_nerdctl_%:
	@echo "Testing nerdctl"
	@GOFLAGS=$(TEST_FLAGS) $(GO) test $(TEST_OPTS) ./tests/e2e -v -run "TestNerdctl/$*"
	@echo " "

## test_ctr_[pattern] Run all end-to-end tests with ctr that match pattern
.PHONY: test_ctr_%
test_ctr_%:
	@echo "Testing ctr"
	@GOFLAGS=$(TEST_FLAGS) $(GO) test $(TEST_OPTS) ./tests/e2e -v -run "TestCtr/$*"
	@echo " "

## test_crictl_[pattern] Run all end-to-end tests with crictl that match pattern
.PHONY: test_crictl_%
test_crictl_%:
	@echo "Testing crictl"
	@GOFLAGS=$(TEST_FLAGS) $(GO) test $(TEST_OPTS) ./tests/e2e -v -run "TestCrictl/$*"
	@echo " "

## help Show this help message
help:
	@echo 'Usage: make <target> <flags>'
	@echo 'Targets:'
	@grep -w "^##" $(MAKEFILE_LIST) | sed -n 's/^## /\t/p' | sed -n 's/ /\@/p' | column -s '\@' -t
	@echo 'Flags:'
	@grep -w "^#?" $(MAKEFILE_LIST) | sed -n 's/^#? /\t/p' | sed -n 's/ /\@/p' | column -s '\@' -t
