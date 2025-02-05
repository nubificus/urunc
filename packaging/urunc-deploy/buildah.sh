#!/bin/bash
export PLATFORMS="linux/amd64"
export IMAGE="harbor.nbfc.io/nubificus/urunc/urunc-deploy"
export TAG="0.4.0-rc5"
buildah build --build-arg BRANCH=compat_kata_qemu --platform=$PLATFORMS --manifest "$IMAGE:$TAG" .
# buildah build --build-arg BRANCH=compat_kata_qemu --jobs=2 --platform=$PLATFORMS --manifest "$IMAGE:$TAG" .
buildah manifest push --all "$IMAGE:$TAG" "docker://$IMAGE:$TAG"