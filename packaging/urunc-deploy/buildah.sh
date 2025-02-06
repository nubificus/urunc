#!/bin/bash
export PLATFORMS="linux/amd64,linux/arm64"
export IMAGE="harbor.nbfc.io/nubificus/urunc/urunc-deploy"
export TAG="0.4.0-rc6"
docker buildx build --build-arg BRANCH=compat_kata_qemu --platform $PLATFORMS --push -t $IMAGE:$TAG .