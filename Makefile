COMMIT := $(shell git describe --dirty --long --always)
VERSION := $(shell cat ./VERSION)
VERSION := $(VERSION)-$(COMMIT)
ARCH := $(shell dpkg --print-architecture)

default: build ;

prepare:
	@go mod tidy
	@go mod vendor
	
urunc: prepare
	GOOS=linux CGO_ENABLED=0 go build -ldflags "-s -w" -ldflags "-w" -ldflags "-linkmode 'external' -extldflags '-static'" \
          -ldflags "-X main.version=${VERSION}" -o dist/urunc_${ARCH} ./cmd/urunc

shim: prepare
	@sed -i 's/DefaultCommand = "runc"/DefaultCommand = "urunc"/g' vendor/github.com/containerd/go-runc/runc.go
	go build -o dist/containerd-shim-urunc-v2_${ARCH} ./cmd/containerd-shim-urunc-v2

build: urunc shim

install:
	mv dist/urunc_${ARCH} /usr/local/bin/urunc
	mv dist/containerd-shim-urunc-v2_${ARCH} /usr/local/bin/containerd-shim-urunc-v2

clean:
	@rm -fr dist/
	@rm -fr vendor/

uninstall:
	rm -f /usr/local/bin/urunc
	rm -f /usr/local/bin/containerd-shim-urunc-v2

urunc_aarch64: prepare
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "-s -w" -ldflags "-w" -ldflags "-linkmode 'external' -extldflags '-static'" \
			-ldflags "-X main.version=${VERSION}" -o dist/urunc_aarch64 ./cmd/urunc

shim_aarch64: prepare
	@sed -i 's/DefaultCommand = "runc"/DefaultCommand = "urunc"/g' vendor/github.com/containerd/go-runc/runc.go
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o dist/containerd-shim-urunc-v2_aarch64 ./cmd/containerd-shim-urunc-v2

urunc_amd64: prepare
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -ldflags "-w" -ldflags "-linkmode 'external' -extldflags '-static'" \
			-ldflags "-X main.version=${VERSION}" -o dist/urunc_amd64 ./cmd/urunc

shim_amd64: prepare
	@sed -i 's/DefaultCommand = "runc"/DefaultCommand = "urunc"/g' vendor/github.com/containerd/go-runc/runc.go
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o dist/containerd-shim-urunc-v2_amd64 ./cmd/containerd-shim-urunc-v2

all: urunc_aarch64 shim_aarch64 urunc_amd64 shim_amd64

test_nerdctl:
	@echo "Testing nerdctl"
	-@sudo GOFLAGS="-count=1" $$(which go) test ./tests/nerdctl -v
	@echo " "

test_ctr:
	@echo "Testing ctr"
	-@sudo GOFLAGS="-count=1" $$(which go) test ./tests/ctr -v
	@echo " "

test_crictl:
	@echo "Testing crictl"
	-@sudo GOFLAGS="-count=1" $$(which go) test ./tests/crictl -v
	@echo " "

test: test_nerdctl test_ctr test_crictl
	-@echo "DONE"