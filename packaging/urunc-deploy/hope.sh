#!/bin/bash
export PLATFORMS="linux/amd64,linux/arm64"
export PLATFORMS="linux/arm64"
export IMAGE="harbor.nbfc.io/nubificus/urunc/urunc-deploy"
export TAG="0.4.0-rc6-buildah"
# docker buildx build --build-arg BRANCH=compat_kata_qemu --platform $PLATFORMS --push -t $IMAGE:$TAG .
sudo buildah build --jobs=1 --build-arg BRANCH=compat_kata_qemu --platform=$PLATFORMS --manifest "$IMAGE:$TAG" .
# buildah build --build-arg BRANCH=compat_kata_qemu --jobs=2 --platform=$PLATFORMS --manifest "$IMAGE:$TAG" .
sudo buildah manifest push --all "$IMAGE:$TAG" "docker://$IMAGE:$TAG"