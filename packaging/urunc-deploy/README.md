# urunc-deploy

TODO:

- k3s with containerd<2: INSTALL OK, UNINSTALL PENDING
- k8s with containerd<2: INSTALL OK, UNINSTALL PENDING
- k8s with containerd>2: INSTALL OK, UNINSTALL PENDING

## k3s quickstart

To install in a k3s cluster:

```bash
git clone https://github.com/nubificus/urunc.git
cd urunc
git checkout feat_urunc-deploy
cd packaging/urunc-deploy

kubectl apply -f urunc-rbac/base/urunc-rbac.yaml && kubectl apply -f urunc-deploy/base/urunc-deploy.yaml && kubectl apply -k urunc-deploy/overlays/k3s && echo "OK"
```

To delete from a k3s cluster:

```bash
kubectl delete -f urunc-deploy/base/urunc-deploy.yaml && kubectl delete -k urunc-deploy/overlays/k3s && kubectl delete -f urunc-deploy/base/urunc-deploy.yaml
```

## Verifying urunc-deploy success

Test the successful installation:

```bash
kubectl apply -f examples/nginx-urunc.yaml
```

## Building the container image

Due to some problems with `docker buildx`, we are currently using `buildah`. Please note that `buildah` requires
the package `qemu-user-static` to build multi-arch images.

In Ubuntu, you can install both using `sudo apt-get install -y buildah qemu-user-static`.

Then, to build the `urunc-deploy` image:

```bash
# export PLATFORMS="linux/arm64,linux/amd64"
export PLATFORMS="linux/amd64"
export IMAGE="harbor.nbfc.io/nubificus/urunc/urunc-deploy"
export TAG="0.4.0-rc5"
buildah build --build-arg BRANCH=compat_kata_qemu --jobs=2 --platform=$PLATFORMS --manifest "$IMAGE:$TAG" .
buildah manifest push --all "$IMAGE:$TAG" "docker://$IMAGE:$TAG"
```
