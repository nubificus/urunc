#!/bin/bash
export PLATFORMS="linux/amd64,linux/arm64"
export IMAGE="harbor.nbfc.io/nubificus/urunc/urunc-deploy"
export TAG="0.4.0-rc7"
docker buildx build --push --platform $PLATFORMS -t "$IMAGE:$TAG" -f Dockerfile.prebuilt .